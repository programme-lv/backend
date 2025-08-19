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
	// CacheDir is the standard Linux cache directory for proglv
	CacheDir = "/var/cache/proglv/testfiles"

	// MaxCacheSize is the maximum cache size in bytes (10GB)
	MaxCacheSize = 10 * 1024 * 1024 * 1024

	// MaxConcurrentDownloads limits parallel downloads
	MaxConcurrentDownloads = 10
)

// TestFileCache manages cached test files with size limits and LRU eviction
type TestFileCache struct {
	cacheDir string
	mu       sync.RWMutex
	sem      chan struct{} // semaphore for download concurrency
}

// CacheEntry represents a cached file with metadata
type CacheEntry struct {
	Path     string
	Size     int64
	LastUsed time.Time
}

// NewTestFileCache creates a new test file cache
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

// GetTestFile returns decompressed test file content from cache or downloads it
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

// getFromCache retrieves file from cache and updates access time
func (c *TestFileCache) getFromCache(sha256Hash string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	filePath := c.getFilePath(sha256Hash)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, false
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false
	}

	// Update access time
	go c.touchFile(filePath)

	return content, true
}

// downloadAndDecompress downloads and decompresses a test file
func (c *TestFileCache) downloadAndDecompress(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Read compressed data
	compressed, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Decompress
	content, err := DecompressWithZstd(compressed)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress: %w", err)
	}

	return content, nil
}

// storeInCache stores content in cache with size management
func (c *TestFileCache) storeInCache(sha256Hash string, content []byte, logger *slog.Logger) {
	c.mu.Lock()
	defer c.mu.Unlock()

	filePath := c.getFilePath(sha256Hash)

	// Ensure we have space
	if err := c.ensureSpace(int64(len(content)), logger); err != nil {
		logger.Warn("failed to ensure cache space", "error", err)
		return
	}

	// Write to temp file first
	tempPath := filePath + ".tmp"
	if err := os.WriteFile(tempPath, content, 0644); err != nil {
		logger.Warn("failed to write cache file", "error", err)
		return
	}

	// Atomic rename
	if err := os.Rename(tempPath, filePath); err != nil {
		logger.Warn("failed to rename cache file", "error", err)
		os.Remove(tempPath)
		return
	}

	logger.Debug("test file cached", "sha256", sha256Hash[:8]+"...", "size_bytes", len(content))
}

// ensureSpace makes sure we have enough space by evicting old files if needed
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

	// Get all cache entries sorted by last used time
	entries, err := c.getCacheEntries()
	if err != nil {
		return err
	}

	// Evict oldest files until we have enough space
	toEvict := currentSize + neededBytes - MaxCacheSize
	var evicted int64

	for _, entry := range entries {
		if evicted >= toEvict {
			break
		}

		if err := os.Remove(entry.Path); err != nil {
			logger.Warn("failed to evict cache file", "path", entry.Path, "error", err)
			continue
		}

		evicted += entry.Size
		logger.Debug("evicted cache file", "path", filepath.Base(entry.Path), "size_bytes", entry.Size)
	}

	logger.Info("cache eviction completed", "evicted_bytes", evicted)
	return nil
}

// getCacheSize returns the total size of cached files
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

// getCacheEntries returns all cache entries sorted by last used time (oldest first)
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

	// Sort by last used time (oldest first)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].LastUsed.After(entries[j].LastUsed) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	return entries, nil
}

// getFilePath returns the cache file path for a given SHA256 hash
func (c *TestFileCache) getFilePath(sha256Hash string) string {
	// Create subdirectories to avoid too many files in one directory
	subdir := sha256Hash[:2]
	dir := filepath.Join(c.cacheDir, subdir)
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, sha256Hash+".txt")
}

// touchFile updates the modification time of a file
func (c *TestFileCache) touchFile(filePath string) {
	now := time.Now()
	os.Chtimes(filePath, now, now)
}

// isTemporaryFile checks if a file is a temporary file
func isTemporaryFile(path string) bool {
	return filepath.Ext(path) == ".tmp"
}

// GetCacheStats returns cache statistics
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
