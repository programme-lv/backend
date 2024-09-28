package fstask

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Solution struct {
	Filename string
	ScoreEq  *int // equal
	ScoreLt  *int // less than
	ScoreLte *int // less than or equal
	ScoreGt  *int // greater than
	ScoreGte *int // greater than or equal
	Author   *string
	ExecTime *float64 // original maximum execution time
	Content  []byte
}

func ReadSolutionsFromTaskDir(dir TaskDir) (res []Solution, err error) {
	requiredSpec := SemVer{major: 2, minor: 5}
	if dir.Spec.LessThan(requiredSpec) {
		format := "specification version %s is not supported, required at least %s"
		return nil, fmt.Errorf(format, dir.Spec.String(), requiredSpec.String())
	}

	tomlStruct := struct {
		Solutions []struct {
			Filename string   `toml:"filename"`
			ScoreEq  *int     `toml:"score_eq"`
			ScoreLt  *int     `toml:"score_lt"`
			ScoreLte *int     `toml:"score_lte"`
			ScoreGt  *int     `toml:"score_gt"`
			ScoreGte *int     `toml:"score_gte"`
			Author   *string  `toml:"author"`
			ExecTime *float64 `toml:"og_max_exec_time"`
		} `toml:"solutions"`
	}{}

	err = toml.Unmarshal(dir.Info, &tomlStruct)
	if err != nil {
		err = fmt.Errorf("failed to unmarshal the solutions: %w", err)
		return
	}

	// If no solutions are defined, return an empty slice.
	if len(tomlStruct.Solutions) == 0 {
		res = []Solution{}
		return res, nil
	}

	// Verify that each solution file exists in the solutions directory.
	solutionsDirPath := filepath.Join(dir.Path, "solutions")
	for _, sol := range tomlStruct.Solutions {
		solPath := filepath.Join(solutionsDirPath, sol.Filename)
		if _, err := os.Stat(solPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("solution file does not exist: %s", solPath)
		} else if err != nil {
			return nil, fmt.Errorf("error accessing solution file '%s': %w", solPath, err)
		}
	}

	res = make([]Solution, 0, len(tomlStruct.Solutions))

	// Read each solution file.
	for _, sol := range tomlStruct.Solutions {
		solutionContent, err := os.ReadFile(filepath.Join(solutionsDirPath, sol.Filename))
		if err != nil {
			return nil, fmt.Errorf("failed to read solution file: %w", err)
		}
		res = append(res, Solution{
			Filename: sol.Filename,
			ScoreEq:  sol.ScoreEq,
			ScoreLt:  sol.ScoreLt,
			ScoreLte: sol.ScoreLte,
			ScoreGt:  sol.ScoreGt,
			ScoreGte: sol.ScoreGte,
			Author:   sol.Author,
			ExecTime: sol.ExecTime,
			Content:  solutionContent,
		})
	}

	return res, nil
}

// LoadSolutionsFromDir loads solutions into the task from the specified directory.
func (task *Task) LoadSolutionsFromDir(dir TaskDir) error {
	solutions, err := ReadSolutionsFromTaskDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read solutions: %w", err)
	}
	task.Solutions = solutions
	return nil
}
