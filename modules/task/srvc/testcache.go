package srvc

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// CacheDir is the local directory used to cache decompressed test files.
	CacheDir = "/var/cache/proglv/testfiles"

	// MaxCacheSize is the maximum cache size in bytes (10 GiB).
	MaxCacheSize = 10 * 1024 * 1024 * 1024

	// MaxConcurrentDownloads is the cap on parallel cache fills.
	MaxConcurrentDownloads = 10
)

// TestFileCache caches decompressed test files on disk with a size limit and LRU eviction.
type TestFileCache struct {
	cacheDir string
	mu       sync.RWMutex
	sem      chan struct{} // semaphore for download concurrency
}

type CacheEntry struct {
	Path     string
	Size     int64
	LastUsed time.Time
}

// NewTestFileCache returns a test-file cache rooted at CacheDir, or a temp dir if that cannot be created.
func NewTestFileCache() *TestFileCache {
	cacheDir := CacheDir
	// Fallback to temp dir if cache dir can't be created
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		cacheDir = filepath.Join(os.TempDir(), "proglv-testfiles")
		os.MkdirAll(cacheDir, 0755)
	}

	return &TestFileCache{
		cacheDir: cacheDir,
		sem:      make(chan struct{}, MaxConcurrentDownloads),
	}
}

func (c *TestFileCache) GetTestFile(ctx context.Context, sha256Hash string, downloadURL string, logger *slog.Logger) ([]byte, error) {
	// Check cache first
	if content, found := c.getFromCache(sha256Hash); found {
		logger.Debug("test file cache hit", "sha256", sha256Hash[:8]+"...")
		return content, nil
	}

	logger.Debug("test file cache miss, downloading", "sha256", sha256Hash[:8]+"...")

	// Acquire semaphore for download
	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// Download and decompress
	content, err := c.downloadAndDecompress(ctx, downloadURL)
	if err != nil {
		return nil, err
	}

	// Store in cache
	c.storeInCache(sha256Hash, content, logger)

	return content, nil
}

func (c *TestFileCache) getFromCache(sha256Hash string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	filePath := c.getFilePath(sha256Hash)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, false
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false
	}

	go c.touchFile(filePath)

	return content, true
}

func (c *TestFileCache) downloadAndDecompress(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	compressed, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	content, err := decompressWithZstd(compressed)
	if err != nil {
		return nil, fmt.Errorf("decompress: %w", err)
	}

	return content, nil
}

func (c *TestFileCache) storeInCache(sha256Hash string, content []byte, logger *slog.Logger) {
	c.mu.Lock()
	defer c.mu.Unlock()

	filePath := c.getFilePath(sha256Hash)

	if err := c.ensureSpace(int64(len(content)), logger); err != nil {
		logger.Warn("ensure cache space", "error", err)
		return
	}

	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, content, 0644); err != nil {
		logger.Warn("write cache file", "error", err)
		return
	}

	if err := os.Rename(tempPath, filePath); err != nil {
		logger.Warn("rename cache file", "error", err)
		os.Remove(tempPath)
		return
	}

	logger.Debug("test file cached", "sha256", sha256Hash[:8]+"...", "size_bytes", len(content))
}

func (c *TestFileCache) ensureSpace(neededBytes int64, logger *slog.Logger) error {
	currentSize, err := c.getCacheSize()
	if err != nil {
		return err
	}

	if currentSize+neededBytes <= MaxCacheSize {
		return nil
	}

	logger.Info("cache size limit approaching, evicting old files",
		"current_size_mb", currentSize/1024/1024,
		"needed_mb", neededBytes/1024/1024,
		"limit_mb", MaxCacheSize/1024/1024)

	entries, err := c.getCacheEntries()
	if err != nil {
		return err
	}

	toEvict := currentSize + neededBytes - MaxCacheSize
	var evicted int64

	for _, entry := range entries {
		if evicted >= toEvict {
			break
		}

		if err := os.Remove(entry.Path); err != nil {
			logger.Warn("evict cache file", "path", entry.Path, "error", err)
			continue
		}

		evicted += entry.Size
		logger.Debug("evicted cache file", "path", filepath.Base(entry.Path), "size_bytes", entry.Size)
	}

	logger.Info("cache eviction completed", "evicted_bytes", evicted)
	return nil
}

func (c *TestFileCache) getCacheSize() (int64, error) {
	var total int64

	err := filepath.Walk(c.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !isTemporaryFile(path) {
			total += info.Size()
		}
		return nil
	})

	return total, err
}

func (c *TestFileCache) getCacheEntries() ([]CacheEntry, error) {
	var entries []CacheEntry

	err := filepath.Walk(c.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !isTemporaryFile(path) {
			entries = append(entries, CacheEntry{
				Path:     path,
				Size:     info.Size(),
				LastUsed: info.ModTime(),
			})
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].LastUsed.After(entries[j].LastUsed) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	return entries, nil
}

func (c *TestFileCache) getFilePath(sha256Hash string) string {
	// Shard by the first two hex chars so one directory is not huge.
	subdir := sha256Hash[:2]
	dir := filepath.Join(c.cacheDir, subdir)
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, sha256Hash+".txt")
}

func (c *TestFileCache) touchFile(filePath string) {
	now := time.Now()
	os.Chtimes(filePath, now, now)
}

func isTemporaryFile(path string) bool {
	return filepath.Ext(path) == ".tmp"
}

func (c *TestFileCache) GetCacheStats() (totalSize int64, fileCount int, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	err = filepath.Walk(c.cacheDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !info.IsDir() && !isTemporaryFile(path) {
			totalSize += info.Size()
			fileCount++
		}
		return nil
	})

	return totalSize, fileCount, err
}
