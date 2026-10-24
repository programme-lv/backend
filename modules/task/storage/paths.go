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

// Image derives a statement image path from task identity and content metadata.
func Image(id, digest, filename string) string {
	return path.Join("tasks", id, "images", digest, path.Base(filename))
}

// Illustration derives an illustration path. An empty digest means no illustration.
func Illustration(id, digest, ext string) string {
	if digest == "" {
		return ""
	}
	return path.Join("tasks", id, "illustration", digest+ext)
}

// Archive derives the path of a task's archived source artifact.
func Archive(id, filename string) string { return path.Join("tasks", id, "archive", filename) }

// PublicImage allows only task image directories, never archived source files.
func PublicImage(key string) bool {
	parts := strings.Split(key, "/")
	if len(parts) < 4 || parts[0] != "tasks" || !ValidID(parts[1]) || path.Clean(key) != key {
		return false
	}
	return parts[2] == "images" || parts[2] == "illustration"
}
