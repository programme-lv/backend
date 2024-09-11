package fstask

import (
	"bytes"
	"log"

	"github.com/pelletier/go-toml/v2"
)

type ProblemTOML struct {
	Specification        string           `toml:"specification"`
	TaskName             string           `toml:"task_name"`
	Metadata             PTomlMetadata    `toml:"metadata"`
	Constraints          PTomlConstraints `toml:"constraints"`
	TestGroups           []PTomlTestGroup `toml:"test_groups"`
	IllustrationImgFname string           `toml:"illustration_image,omitempty"`
	VisInpSTs            []int            `toml:"visible_input_subtasks"`
	TestIDOverwrite      map[string]int   `toml:"test_id_overwrite,omitempty"`
}

type PTomlMetadata struct {
	ProblemTags        []string          `toml:"problem_tags"`
	DifficultyFrom1To5 int               `toml:"difficulty_1_to_5"`
	TaskAuthors        []string          `toml:"task_authors"`
	OriginOlympiad     string            `toml:"origin_olympiad"`
	OriginNotes        map[string]string `toml:"origin_notes,omitempty"`
	OriginInstitution  string            `toml:"origin_institution,omitempty"`
}

type PTomlConstraints struct {
	MemoryMegabytes int     `toml:"memory_megabytes"`
	CPUTimeSeconds  float64 `toml:"cpu_time_seconds"`
}

// PTomlTestGroup is a structure to store groups used in LIO test format
type PTomlTestGroup struct {
	GroupID    int      `toml:"group_id"`
	Points     int      `toml:"points"`
	Public     bool     `toml:"public"`
	Subtask    int      `toml:"subtask,omitempty"`
	TestIDs    []int    `toml:"test_ids,omitempty"`
	TestFnames []string `toml:"test_filenames,omitempty"`
}

func (task *Task) encodeProblemTOML() ([]byte, error) {
	testIDOverwrite := task.getTestIDByFilenameOverwriteMap()

	t := ProblemTOML{
		Specification: proglvFSTaskFormatSpecVersOfScript,
		TaskName:      task.taskName,
		Metadata: PTomlMetadata{
			ProblemTags:        task.problemTags,
			DifficultyFrom1To5: task.difficultyOneToFive,
			TaskAuthors:        task.problemAuthors,
			OriginOlympiad:     task.originOlympiad,
			OriginNotes:        task.OriginNotes,
			OriginInstitution:  task.OriginInstitution,
		},
		Constraints:          PTomlConstraints{MemoryMegabytes: task.memoryMegabytes, CPUTimeSeconds: task.cpuTimeSeconds},
		TestGroups:           []PTomlTestGroup{},
		IllustrationImgFname: task.illstrImgFname,
		VisInpSTs:            task.visibleInputSubtasks,
		TestIDOverwrite:      testIDOverwrite,
	}
	t.Specification = proglvFSTaskFormatSpecVersOfScript

	// fill test groups
	for _, tg := range task.testGroupIDs {
		testFnames := make([]string, 0)
		for _, testID := range task.tGroupTestIDs[tg] {
			testFnames = append(testFnames, task.getTestToBeWrittenFname(testID))
		}
		ptomlTestGroup := PTomlTestGroup{
			GroupID: tg,
			Points:  task.tGroupPoints[tg],
			Public:  task.isTGroupPublic[tg],
			Subtask: task.tGroupToStMap[tg],
			// TestIDs: task.tGroupTestIDs[tg],
			TestFnames: testFnames,
		}

		t.TestGroups = append(t.TestGroups, ptomlTestGroup)
	}

	buf := bytes.NewBuffer(make([]byte, 0))
	err := toml.NewEncoder(buf).
		SetTablesInline(false).
		// SetArraysMultiline(true).
		SetIndentTables(true).Encode(t)

	if err != nil {
		log.Fatalf("Failed to marshal the problem.toml: %v\n", err)
	}

	return buf.Bytes(), nil
}
