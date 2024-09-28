package fstask

import (
	"fmt"

	"github.com/pelletier/go-toml/v2"
)

type Metadata struct {
	ProblemTags         []string
	DifficultyOneToFive int
	TaskAuthors         []string
	OriginOlympiad      string
	OriginNotes         map[string]string
}

func ReadMetadataFromTaskDir(dir TaskDir) (Metadata, error) {
	requiredSpec := SemVer{major: 2}
	if dir.Spec.LessThan(requiredSpec) {
		format := "specification version %s is not supported, required at least %s"
		return Metadata{}, fmt.Errorf(format, dir.Spec.String(), requiredSpec.String())
	}

	x := struct {
		Metadata struct {
			ProblemTags         []string          `toml:"problem_tags"`
			DifficultyOneToFive int               `toml:"difficulty_1_to_5"`
			TaskAuthors         []string          `toml:"task_authors"`
			OriginOlympiad      string            `toml:"origin_olympiad"`
			OriginNotes         map[string]string `toml:"origin_notes,omitempty"`
		} `toml:"metadata"`
	}{}

	err := toml.Unmarshal(dir.Info, &x)
	if err != nil {
		format := "failed to unmarshal the metadata: %w"
		return Metadata{}, fmt.Errorf(format, err)
	}

	return Metadata{
		ProblemTags:         x.Metadata.ProblemTags,
		DifficultyOneToFive: x.Metadata.DifficultyOneToFive,
		TaskAuthors:         x.Metadata.TaskAuthors,
		OriginOlympiad:      x.Metadata.OriginOlympiad,
		OriginNotes:         x.Metadata.OriginNotes,
	}, nil
}

func (task *Task) LoadMetadataFromDir(dir TaskDir) error {
	metadata, err := ReadMetadataFromTaskDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}
	task.ProblemTags = metadata.ProblemTags
	task.DifficultyOneToFive = metadata.DifficultyOneToFive
	task.TaskAuthors = metadata.TaskAuthors
	task.OriginOlympiad = metadata.OriginOlympiad
	task.OriginNotes = metadata.OriginNotes
	return nil
}
