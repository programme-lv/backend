package fstask

import (
	"fmt"

	"github.com/pelletier/go-toml/v2"
)

type Constraints struct {
	CPUTimeLimitInSeconds  float64
	MemoryLimitInMegabytes int
}

func (dir TaskDir) ReadConstraintsFromTaskDir() (res Constraints, err error) {
	requiredSpec := Version{major: 2}
	if dir.Specification.LessThan(requiredSpec) {
		format := "specification version %s is not supported, required at least %s"
		err = fmt.Errorf(format, dir.Specification.String(), requiredSpec.String())
		return
	}

	x := struct {
		Constraints struct {
			CPUTimeLimitInSeconds  float64 `toml:"cpu_time_seconds"`
			MemoryLimitInMegabytes int     `toml:"memory_megabytes"`
		} `toml:"constraints"`
	}{}

	err = toml.Unmarshal(dir.ProblemToml, &x)
	if err != nil {
		format := "failed to unmarshal the constraints: %w"
		err = fmt.Errorf(format, err)
		return
	}

	res.CPUTimeLimitInSeconds = x.Constraints.CPUTimeLimitInSeconds
	res.MemoryLimitInMegabytes = x.Constraints.MemoryLimitInMegabytes

	return
}

func (task *Task) LoadConstraintsFromDir(dir TaskDir) error {
	constraints, err := dir.ReadConstraintsFromTaskDir()
	if err != nil {
		return fmt.Errorf("failed to read constraints: %w", err)
	}
	task.CPUTimeLimitSeconds = constraints.CPUTimeLimitInSeconds
	task.MemoryLimitMegabytes = constraints.MemoryLimitInMegabytes
	return nil
}
