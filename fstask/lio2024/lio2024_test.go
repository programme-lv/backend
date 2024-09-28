package lio2024_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/programme-lv/backend/importers/lio2024"
)

func TestLio2024Yaml(t *testing.T) {
	yamlContent := `name: 'Kp'
title: 'Kvadrātveida putekļsūcējs'
statements:
  - ['./teksts/kp.typ', 'lv']

time_limit: 0.5
memory_limit: 256

subtask_points: [0, 3, 48, 28, 21]

tests_archive: './testi/tests.zip'
checker: './riki/checker.cpp'

tests_groups:
  - groups: 0
    points: 0
    public: true
    subtask: 0
    comment: Piemēri

  - groups: 1
    points: 3
    public: true
    subtask: 1

  - groups: [2, 2]
    points: 8
    public: [2]
    subtask: 2

  - groups: [3, 4]
    points: 10
    public: [4]
    subtask: 2
`

	expected := lio2024.ParsedLio2024Yaml{
		CpuTimeLimitInSeconds:  0.5,
		MemoryLimitInMegabytes: 256,
		FullTaskName:           "Kvadrātveida putekļsūcējs",
		TestZipPathRelToYaml:   "./testi/tests.zip",
		CheckerPathRelToYaml:   &([]string{"./riki/checker.cpp"}[0]),
		// InteractorPathRelToYaml: &([]string{"./riki/interactor.cpp"}[0]),
		SubtaskPoints: []int{0, 3, 48, 28, 21},
		TestGroups: []lio2024.ParsedLio2024YamlTestGroup{
			{
				GroupID: 0,
				Points:  0,
				Public:  true,
				Subtask: 0,
				Comment: &([]string{"Piemēri"}[0]),
			},
			{
				GroupID: 1,
				Points:  3,
				Public:  true,
				Subtask: 1,
			},
			{
				GroupID: 2,
				Points:  8,
				Public:  true,
				Subtask: 2,
			},
			{
				GroupID: 3,
				Points:  10,
				Public:  false,
				Subtask: 2,
			},
			{
				GroupID: 4,
				Points:  10,
				Public:  true,
				Subtask: 2,
			},
		},
	}

	actual, err := lio2024.ParseLio2024Yaml([]byte(yamlContent))
	require.NoErrorf(t, err, "Failed to parse Lio2024 YAML: %v", err)

	assert.Equal(t, expected, actual)
}
