package fstask

import (
	"fmt"

	"github.com/pelletier/go-toml/v2"
)

// GeneralInformation holds the general details about a task.
type GeneralInformation struct {
	TaskName             string
	VisibleInputSubtasks []int
	IllustrationImage    string
	ProblemTags          []string
	DifficultyOneToFive  int
}

// ReadGeneralInformationFromTaskDir reads and parses the general information
// from the provided TaskDir. It unmarshals the TOML data into the GeneralInformation struct.
func ReadGeneralInformationFromTaskDir(dir TaskDir) (*GeneralInformation, error) {
	// Define a temporary struct to map the TOML fields.
	tomlStruct := struct {
		TaskName             string   `toml:"task_name"`
		VisibleInputSubtasks []int    `toml:"visible_input_subtasks"`
		IllustrationImage    string   `toml:"illustration_image"`
		ProblemTags          []string `toml:"problem_tags"`
		DifficultyOneToFive  int      `toml:"difficulty_1_to_5"`
	}{}

	// Unmarshal the TOML data into the temporary struct.
	err := toml.Unmarshal(dir.ProblemToml, &tomlStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal general information: %w", err)
	}

	// Optional: Add validation for required fields.
	if tomlStruct.TaskName == "" {
		return nil, fmt.Errorf("task_name is required in general information")
	}
	if tomlStruct.DifficultyOneToFive < 1 || tomlStruct.DifficultyOneToFive > 5 {
		return nil, fmt.Errorf("difficulty_1_to_5 must be between 1 and 5")
	}

	// Assign the parsed values to the GeneralInformation struct.
	gi := &GeneralInformation{
		TaskName:             tomlStruct.TaskName,
		VisibleInputSubtasks: tomlStruct.VisibleInputSubtasks,
		IllustrationImage:    tomlStruct.IllustrationImage,
		ProblemTags:          tomlStruct.ProblemTags,
		DifficultyOneToFive:  tomlStruct.DifficultyOneToFive,
	}

	return gi, nil
}

// LoadGeneralInformation loads the general information into the Task struct.
// It utilizes the ReadGeneralInformationFromTaskDir function to parse the data.
func (task *Task) LoadGeneralInformation(dir TaskDir) error {
	gi, err := ReadGeneralInformationFromTaskDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read general information: %w", err)
	}
	task.FullName = gi.TaskName
	task.VisibleInputSubtasks = gi.VisibleInputSubtasks
	task.IllustrAssetFilename = gi.IllustrationImage
	task.ProblemTags = gi.ProblemTags
	task.DifficultyOneToFive = gi.DifficultyOneToFive
	return nil
}
