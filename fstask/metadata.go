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
		ProblemTags         []string `toml:"problem_tags"`
		DifficultyOneToFive int      `toml:"difficulty_one_to_five"`
		ProblemAuthors      []string `toml:"problem_authors"`
		OriginOlympiad      string   `toml:"origin_olympiad"`
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
	task.ProblemAuthors = tomlStruct.Metadata.ProblemAuthors
	task.OriginOlympiad = tomlStruct.Metadata.OriginOlympiad

	return nil
}
