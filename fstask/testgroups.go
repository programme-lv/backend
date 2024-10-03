package fstask

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

func ReadTestGroupsFromDir(dir TaskDir) ([]TestGroup, error) {
	obj := struct {
		TestGroups []struct {
			GroupID int           `toml:"id"`
			Points  int           `toml:"points"`
			Tests   []interface{} `toml:"tests"`
		} `toml:"test_groups"`
	}{}

	err := toml.Unmarshal(dir.ProblemToml, &obj)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal test groups: %w", err)
	}

	testPath := filepath.Join(dir.AbsPath, "tests")
	// read test files in lex order. create a map from basename to id. id starts from 1

	entries, err := os.ReadDir(testPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tests directory: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	testOrderIds := make(map[string]int)
	testID := 1
	for _, entry := range entries {
		// if ends with ".in", assign an id and increment
		if strings.HasSuffix(entry.Name(), ".in") {
			base := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
			testOrderIds[base] = testID
			testID++
		}
	}

	testGroups := make([]TestGroup, 0, len(obj.TestGroups))
	for _, group := range obj.TestGroups {
		testGroup := TestGroup{
			GroupID: group.GroupID,
			Points:  group.Points,
			TestIDs: make([]int, 0, len(group.Tests)),
		}
		for _, test := range group.Tests {
			switch v := test.(type) {
			case int:
				testGroup.TestIDs = append(testGroup.TestIDs, v)
			case string:
				base := strings.TrimSuffix(v, filepath.Ext(v))
				id, ok := testOrderIds[base]
				if !ok {
					return nil, fmt.Errorf("test %s not found in test order", base)
				}
				testGroup.TestIDs = append(testGroup.TestIDs, id)
			}
		}
		testGroups = append(testGroups, testGroup)
	}

	return testGroups, nil
}

func (task *Task) LoadTestGroups(dir TaskDir) error {
	testGroups, err := ReadTestGroupsFromDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read test groups: %w", err)
	}
	task.TestGroups = testGroups
	return nil
}
