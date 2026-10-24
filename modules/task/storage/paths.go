// Package storage defines the task asset layout and offline maintenance operations.
package storage

import (
	"crypto/sha256"
	"fmt"
	"path"
	"regexp"
	"strings"
)

var taskIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// ValidID reports whether id is a task identifier, never a filesystem path.
func ValidID(id string) bool { return taskIDPattern.MatchString(id) }

// Digest returns the full SHA-256 of an asset's original bytes.
func Digest(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

const (
	AssetsRoot  = "assets"
	ArchiveRoot = "archive"
	CacheRoot   = "cache"
)

// Image derives a public statement image path from task identity and metadata.
func Image(id, digest, filename string) string {
	return path.Join(AssetsRoot, "tasks", id, "images", digest, path.Base(filename))
}

// Illustration derives a public illustration path. An empty digest means none.
func Illustration(id, digest, ext string) string {
	if digest == "" {
		return ""
	}
	return path.Join(AssetsRoot, "tasks", id, "illustration", digest+ext)
}

// Archive derives the path of a task's admin-only archived source artifact.
func Archive(id, filename string) string {
	return path.Join(ArchiveRoot, "tasks", id, filename)
}

// PublicImage allows only public task image directories, never archives/cache.
func PublicImage(key string) bool {
	parts := strings.Split(key, "/")
	if len(parts) < 5 || parts[0] != AssetsRoot || parts[1] != "tasks" ||
		!ValidID(parts[2]) || path.Clean(key) != key {
		return false
	}
	return parts[3] == "images" || parts[3] == "illustration"
}
