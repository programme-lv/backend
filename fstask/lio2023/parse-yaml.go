package lio2023

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

/*
name: 'iedalas'
title: 'Iedaļas'
statements:
  - ['./../Uzdevumi/TODO', 'lv']
public_groups: [0, 1, 6, 11]
time_limit: 1.5
memory_limit: 256
subtask_points: [0, 20, 20, 60]
validator: './riki/validator.cpp'
test_archive: './testi.zip'
*/

type Lio2023Yaml struct {
	Name          string  `yaml:"name"`
	Title         string  `yaml:"title"`
	PublicGroups  []int   `yaml:"public_groups"`
	TimeLimit     float64 `yaml:"time_limit"`
	MemoryLimit   int     `yaml:"memory_limit"`
	SubtaskPoints []int   `yaml:"subtask_points"`
	Validator     string  `yaml:"validator"`
	TestArchive   string  `yaml:"test_archive"`
}

func ParseLio2023Yaml(yamlContent []byte) (*Lio2023Yaml, error) {
	var lio2023Yaml Lio2023Yaml
	if err := yaml.Unmarshal(yamlContent, &lio2023Yaml); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml: %v", err)
	}

	return &lio2023Yaml, nil
}
