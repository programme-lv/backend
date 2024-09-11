package fstask

import (
	"fmt"
	"log"

	"github.com/pelletier/go-toml/v2"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

type metadata struct {
	problemTags         []string
	difficultyOneToFive int
	problemAuthors      []string
	originOlympiad      string
}

func readProblemTags(specVers string, tomlContent string) ([]string, error) {
	log.Printf("Reading problem tags for specification version: %s\n", specVers)
	cmpres, err := largerOrEqualSemVersionThan(specVers, "2.0")
	if err != nil {
		log.Printf("Error comparing semversions: %v\n", err)
		return nil, fmt.Errorf("error comparing semversions: %w", err)
	}
	if !cmpres {
		log.Printf("Unsupported specification version: %s\n", specVers)
		return nil, fmt.Errorf("unsupported specification version: %s", specVers)
	}

	type metadataStruct struct {
		ProblemTags []string `toml:"problem_tags"`
	}

	tomlStruct := struct {
		Metadata metadataStruct `toml:"metadata"`
	}{}

	err = toml.Unmarshal([]byte(tomlContent), &tomlStruct)
	if err != nil {
		log.Printf("Failed to unmarshal the problem tags: %v\n", err)
		return nil, fmt.Errorf("failed to unmarshal the problem tags: %w", err)
	}

	log.Printf("Successfully read problem tags: %v\n", tomlStruct.Metadata.ProblemTags)
	return tomlStruct.Metadata.ProblemTags, nil
}

func readProblemAuthors(specVers string, tomlContent string) ([]string, error) {
	log.Printf("Reading problem authors for specification version: %s\n", specVers)
	cmpres, err := largerOrEqualSemVersionThan(specVers, "2.0")
	if err != nil {
		log.Printf("Error comparing semversions: %v\n", err)
		return nil, fmt.Errorf("error comparing semversions: %w", err)
	}
	if !cmpres {
		log.Printf("Unsupported specification version: %s\n", specVers)
		return nil, fmt.Errorf("unsupported specification version: %s", specVers)
	}

	type metadataStruct struct {
		ProblemAuthors []string `toml:"task_authors"`
	}

	tomlStruct := struct {
		Metadata metadataStruct `toml:"metadata"`
	}{}

	err = toml.Unmarshal([]byte(tomlContent), &tomlStruct)
	if err != nil {
		log.Printf("Failed to unmarshal the problem authors: %v\n", err)
		return nil, fmt.Errorf("failed to unmarshal the problem authors: %w", err)
	}

	log.Printf("Successfully read problem authors: %v\n", tomlStruct.Metadata.ProblemAuthors)
	return tomlStruct.Metadata.ProblemAuthors, nil
}

func readOriginOlympiad(specVers string, tomlContent string) (string, error) {
	log.Printf("Reading origin olympiad for specification version: %s\n", specVers)
	cmpres, err := largerOrEqualSemVersionThan(specVers, "2.0")
	if err != nil {
		log.Printf("Error comparing semversions: %v\n", err)
		return "", fmt.Errorf("error comparing semversions: %w", err)
	}
	if !cmpres {
		log.Printf("Unsupported specification version: %s\n", specVers)
		return "", fmt.Errorf("unsupported specification version: %s", specVers)
	}

	type metadataStruct struct {
		OriginOlympiad *string `toml:"origin_olympiad"`
	}

	tomlStruct := struct {
		Metadata metadataStruct `toml:"metadata"`
	}{}

	err = toml.Unmarshal([]byte(tomlContent), &tomlStruct)
	if err != nil {
		log.Printf("Failed to unmarshal the origin olympiad: %v\n", err)
		return "", fmt.Errorf("failed to unmarshal the origin olympiad: %w", err)
	}

	res := ""
	if tomlStruct.Metadata.OriginOlympiad != nil {
		res = *tomlStruct.Metadata.OriginOlympiad
	}

	log.Printf("Successfully read origin olympiad: %s\n", res)
	return res, nil
}

func readDifficultyOneToFive(specVers string, tomlContent string) (int, error) {
	log.Printf("Reading difficulty (1 to 5) for specification version: %s\n", specVers)
	cmpres, err := largerOrEqualSemVersionThan(specVers, "2.0")
	if err != nil {
		log.Printf("Error comparing semversions: %v\n", err)
		return 0, fmt.Errorf("error comparing semversions: %w", err)
	}
	if !cmpres {
		log.Printf("Unsupported specification version: %s\n", specVers)
		return 0, fmt.Errorf("unsupported specification version: %s", specVers)
	}
	type metadataStruct struct {
		DifficultyFrom1To5 *int `toml:"difficulty_1_to_5"`
	}

	tomlStruct := struct {
		Metadata metadataStruct `toml:"metadata"`
	}{}

	err = toml.Unmarshal([]byte(tomlContent), &tomlStruct)
	if err != nil {
		log.Printf("Failed to unmarshal the difficulty: %v\n", err)
		return 0, fmt.Errorf("failed to unmarshal the difficulty: %w", err)
	}

	res := 0
	if tomlStruct.Metadata.DifficultyFrom1To5 != nil {
		res = *tomlStruct.Metadata.DifficultyFrom1To5
	}

	log.Printf("Successfully read difficulty: %d\n", res)
	return res, nil
}
