package fstask

import (
	"fmt"

	"github.com/pelletier/go-toml/v2"
)

func (task *Task) readConstraintsFromToml(bytes []byte) error {
	vers, err := getSemVersFromToml(bytes)
	if err != nil {
		return fmt.Errorf("failed to get the specification version: %w", err)
	}

	if vers.LessThan(SemVer{major: 2}) {
		return fmt.Errorf("unsupported specification version: %s",
			vers.String())
	}

	type pTomlConstraints struct {
		CPUTimeLimitInSeconds  float64 `toml:"cpu_time_seconds"`
		MemoryLimitInMegabytes int     `toml:"memory_megabytes"`
	}

	tomlStruct := struct {
		Constraints pTomlConstraints `toml:"constraints"`
	}{}

	err = toml.Unmarshal(bytes, &tomlStruct)
	if err != nil {
		return fmt.Errorf("failed to unmarshal the constraints: %w", err)
	}

	task.CpuTimeLimInSeconds = tomlStruct.Constraints.CPUTimeLimitInSeconds
	task.MemoryLimInMegabytes = tomlStruct.Constraints.MemoryLimitInMegabytes

	return nil
}
