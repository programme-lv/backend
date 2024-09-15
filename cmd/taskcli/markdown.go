package main

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// mapMarkdownImageURLs takes a string of markdown content and returns a tuple of
// a new string with all image URLs replaced with a UUID, and a map of UUIDs
// to their original URLs. The replacement is done by finding all instances of
// the pattern "![.*?](.*?)" and replacing the URL (in the second capture group)
// with a new UUID. The UUID and original URL are stored in the returned map.
func mapMarkdownImageURLs(content string) (string, map[string]string) {
	// Regular expression to find image URLs in markdown
	re := regexp.MustCompile(`!\[.*?\]\((.*?)\)`)
	// Map to store UUIDs and original URLs
	urlMap := make(map[string]string)

	// Function to generate a UUID and update the URL map
	replacer := func(match string) string {
		// Extract the URL from the match
		url := re.FindStringSubmatch(match)[1]
		// Generate a new UUID
		id := uuid.New().String()
		// Store the UUID and original URL in the map
		urlMap[id] = url
		// Replace the URL with the UUID
		return strings.Replace(match, url, id, 1)
	}

	// Replace all image URLs in the content
	updatedContent := re.ReplaceAllStringFunc(content, replacer)

	return updatedContent, urlMap
}
