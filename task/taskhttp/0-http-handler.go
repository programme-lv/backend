package taskhttp

import (
	"time"

	"github.com/patrickmn/go-cache"
	"github.com/programme-lv/backend/task/tasksrvc"
)

type TaskHttpHandler struct {
	taskSrvc tasksrvc.TaskSrvcClient
	cache    *cache.Cache
}

func NewTaskHttpHandler(taskSrvc tasksrvc.TaskSrvcClient) *TaskHttpHandler {
	// Create a cache with 3 second default expiration and 10 second cleanup interval
	c := cache.New(5*time.Second, 10*time.Second)
	return &TaskHttpHandler{
		taskSrvc: taskSrvc,
		cache:    c,
	}
}
