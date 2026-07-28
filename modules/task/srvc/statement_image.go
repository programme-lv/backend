package srvc

import (
	"path"
	"strings"
)

const taskStatementImageDir = "md-images"

func taskStatementImageObjectKey(storedKey string) string {
	if storedKey == "" {
		return ""
	}
	if strings.HasPrefix(storedKey, taskStatementImageDir+"/") ||
		strings.HasPrefix(storedKey, "task/") ||
		strings.HasPrefix(storedKey, "task-md-images/") {
		return storedKey
	}
	return path.Join(taskStatementImageDir, storedKey)
}

func taskStatementImageStoredKey(objectKey string) string {
	return strings.TrimPrefix(objectKey, taskStatementImageDir+"/")
}
