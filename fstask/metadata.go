package fstask

import (
	"fmt"

	"github.com/pelletier/go-toml/v2"
)

func (task *Task) readMetadataFromToml(bytes []byte) error {
	vers, err := getSemVersFromToml(bytes)
	if err != nil {
		return fmt.Errorf("failed to get the specification version: %w", err)
	}

	if vers.LessThan(SemVer{major: 2}) {
		return fmt.Errorf("unsupported specification version: %s",
			vers.String())
	}

	type pTomlMetadata struct {
		ProblemTags         []string          `toml:"problem_tags"`
		DifficultyOneToFive int               `toml:"difficulty_1_to_5"`
		TaskAuthors         []string          `toml:"task_authors"`
		OriginOlympiad      string            `toml:"origin_olympiad"`
		OriginNotes         map[string]string `toml:"origin_notes,omitempty"`
	}

	tomlStruct := struct {
		Metadata pTomlMetadata `toml:"metadata"`
	}{}

	err = toml.Unmarshal(bytes, &tomlStruct)
	if err != nil {
		return fmt.Errorf("failed to unmarshal the metadata: %w", err)
	}

	task.ProblemTags = tomlStruct.Metadata.ProblemTags
	task.DifficultyOneToFive = tomlStruct.Metadata.DifficultyOneToFive
	task.TaskAuthors = tomlStruct.Metadata.TaskAuthors
	task.OriginOlympiad = tomlStruct.Metadata.OriginOlympiad
	task.OriginNotes = tomlStruct.Metadata.OriginNotes

	return nil
}

/*

	t.OriginNotes, err = readOriginNotes(problemTomlContent)
	if err != nil {
		log.Printf("Error reading origin notes: %v\n", err)
	}
*/

/*

func readOriginNotes(pToml []byte) (map[string]string, error) {
	type Metadata struct {
		OriginNotes map[string]string `toml:"origin_notes,omitempty"`
	}
	metadata := struct {
		Metadata Metadata `toml:"metadata"`
	}{}

	err := toml.Unmarshal(pToml, &metadata)
	if err != nil {
		log.Printf("Failed to unmarshal the origin notes: %v\n", err)
		return nil, fmt.Errorf("failed to unmarshal the origin notes: %w", err)
	}

	return metadata.Metadata.OriginNotes, nil
}
*/
