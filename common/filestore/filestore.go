package filestore

import (
	"errors"
	"fmt"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type Store struct {
	root string
}

func NewStore(root string) (*Store, error) {
	if root == "" {
		return nil, errors.New("storage root is empty")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve storage root: %w", err)
	}
	if err := os.MkdirAll(absRoot, 0755); err != nil {
		return nil, fmt.Errorf("create storage root: %w", err)
	}
	return &Store{root: absRoot}, nil
}

func (s *Store) Root() string {
	return s.root
}

func (s *Store) Upload(content []byte, key string, mediaType string) (string, error) {
	fullPath, err := s.Path(key)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("create object directory: %w", err)
	}
	if err := os.WriteFile(fullPath, content, 0644); err != nil {
		return "", fmt.Errorf("write object: %w", err)
	}
	return "file://" + key, nil
}

func (s *Store) Download(key string) ([]byte, error) {
	fullPath, err := s.Path(key)
	if err != nil {
		return nil, err
	}
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("read object: %w", err)
	}
	return content, nil
}

func (s *Store) Exists(key string) (bool, error) {
	fullPath, err := s.Path(key)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("stat object: %w", err)
}

func (s *Store) Delete(key string) error {
	fullPath, err := s.Path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(fullPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

func (s *Store) ServeHTTP(w http.ResponseWriter, r *http.Request, key string) error {
	fullPath, err := s.Path(key)
	if err != nil {
		return err
	}
	file, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("open object: %w", err)
	}
	defer file.Close()
	contentType := mime.TypeByExtension(filepath.Ext(fullPath))
	if contentType == "" {
		var buf [512]byte
		n, _ := file.Read(buf[:])
		contentType = http.DetectContentType(buf[:n])
		if _, err := file.Seek(0, 0); err != nil {
			return fmt.Errorf("seek object: %w", err)
		}
	}
	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, path.Base(key), fileModTime(fullPath), file)
	return nil
}

func (s *Store) Path(key string) (string, error) {
	cleanKey, err := CleanKey(key)
	if err != nil {
		return "", err
	}
	fullPath := filepath.Join(s.root, filepath.FromSlash(cleanKey))
	rel, err := filepath.Rel(s.root, fullPath)
	if err != nil {
		return "", fmt.Errorf("resolve object path: %w", err)
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", fmt.Errorf("object key escapes storage root: %q", key)
	}
	return fullPath, nil
}

func CleanKey(key string) (string, error) {
	if key == "" {
		return "", errors.New("object key is empty")
	}
	if strings.Contains(key, "\\") {
		return "", fmt.Errorf("object key contains backslash: %q", key)
	}
	if strings.HasPrefix(key, "/") {
		return "", fmt.Errorf("object key is absolute: %q", key)
	}
	for _, part := range strings.Split(key, "/") {
		if part == ".." {
			return "", fmt.Errorf("object key contains parent segment: %q", key)
		}
	}
	cleanKey := path.Clean(key)
	if cleanKey == "." || cleanKey == ".." || strings.HasPrefix(cleanKey, "../") {
		return "", fmt.Errorf("object key escapes storage root: %q", key)
	}
	return cleanKey, nil
}

func fileModTime(fullPath string) time.Time {
	info, err := os.Stat(fullPath)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
