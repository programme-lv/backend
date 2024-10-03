package fstask

import (
	"fmt"
	"sort"

	"github.com/pelletier/go-toml/v2"
)

type Subtask struct {
	Points       int
	TestIDs      []int
	Descriptions map[string]string
}

func ReadSubtasksFromDir(dir TaskDir) ([]Subtask, error) {
	x := struct {
		Subtasks []struct {
			ID           int               `toml:"id"`
			Points       int               `toml:"points"`
			Descriptions map[string]string `toml:"descriptions"`
			Tests        []interface{}     `toml:"tests"`
		}
	}{}

	err := toml.Unmarshal(dir.ProblemToml, &x)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal subtasks: %w", err)
	}

	if len(x.Subtasks) == 0 {
		return nil, nil
	}

	// sort x.Subtask by ID
	sort.Slice(x.Subtasks, func(i, j int) bool {
		return x.Subtasks[i].ID < x.Subtasks[j].ID
	})

	if x.Subtasks[0].ID != 1 {
		return nil, fmt.Errorf("consecutive subtask IDs must start with 1")
	}
	if x.Subtasks[len(x.Subtasks)-1].ID != len(x.Subtasks) {
		return nil, fmt.Errorf("consecutive subtask IDs must end with %d", len(x.Subtasks))
	}

	subtasks := make([]Subtask, 0)
	for _, v := range x.Subtasks {
		testIDs, err := findTestIDs(dir, v.Tests)
		if err != nil {
			return nil, fmt.Errorf("failed to translate test references to IDs: %w", err)
		}
		subtasks = append(subtasks, Subtask{
			Points:       v.Points,
			TestIDs:      testIDs,
			Descriptions: v.Descriptions,
		})
	}

	return subtasks, nil
}

func (task *Task) LoadSubtasks(dir TaskDir) error {
	subtasks, err := ReadSubtasksFromDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read subtasks: %w", err)
	}

	task.Subtasks = subtasks
	return nil
}
