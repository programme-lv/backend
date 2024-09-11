package fstask

import (
	"fmt"
	"log"

	"github.com/pelletier/go-toml/v2"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

type constraints struct {
	CPUTimeLimitInSeconds  float64
	MemoryLimitInMegabytes int
}

func readConstraintsFromToml(version SemVer, tomlContent string) (*constraints, error) {
	if version.LessThan(SemVer{major: 2}) {
		return nil, fmt.Errorf("unsupported specification version: %s", version)
	}

	type pTomlConstraints struct {
		CPUTimeLimitInSeconds  float64 `toml:"cpu_time_seconds"`
		MemoryLimitInMegabytes int     `toml:"memory_megabytes"`
	}

	tomlStruct := struct {
		Constraints pTomlConstraints `toml:"constraints"`
	}{}

	err := toml.Unmarshal([]byte(tomlContent), &tomlStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal the constraints: %w", err)
	}

	return &constraints{
		CPUTimeLimitInSeconds:  tomlStruct.Constraints.CPUTimeLimitInSeconds,
		MemoryLimitInMegabytes: tomlStruct.Constraints.MemoryLimitInMegabytes,
	}, nil
}
