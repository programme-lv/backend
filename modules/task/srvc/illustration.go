package srvc

import (
	"path"
	"strings"
)

const taskIllustrationDir = "illustrations"

func taskIllustrationObjectKey(storedKey string) string {
	if storedKey == "" {
		return ""
	}
	if strings.Contains(storedKey, "/") {
		return storedKey
	}
	return path.Join(taskIllustrationDir, storedKey)
}

func taskIllustrationStoredKey(objectKey string) string {
	return strings.TrimPrefix(objectKey, taskIllustrationDir+"/")
}
