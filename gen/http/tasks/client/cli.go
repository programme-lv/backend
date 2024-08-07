// Code generated by goa v3.18.2, DO NOT EDIT.
//
// tasks HTTP client CLI support package
//
// Command:
// $ goa gen github.com/programme-lv/backend/design

package client

import (
	tasks "github.com/programme-lv/backend/gen/tasks"
)

// BuildGetTaskPayload builds the payload for the tasks getTask endpoint from
// CLI flags.
func BuildGetTaskPayload(tasksGetTaskTaskID string) (*tasks.GetTaskPayload, error) {
	var taskID string
	{
		taskID = tasksGetTaskTaskID
	}
	v := &tasks.GetTaskPayload{}
	v.TaskID = taskID

	return v, nil
}
