package fstask

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

func findTestIDs(dir TaskDir, tests []any) ([]int, error) {
	testPath := filepath.Join(dir.AbsPath, "tests")

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

	ids := make([]int, 0, len(tests))

	for _, test := range tests {
		switch v := test.(type) {
		case int:
			ids = append(ids, v)
		case int64:
			ids = append(ids, int(v))
		case string:
			base := strings.TrimSuffix(v, filepath.Ext(v))
			id, ok := testOrderIds[base]
			if !ok {
				return nil, fmt.Errorf("test %s not found in tests", base)
			}
			ids = append(ids, id)
		}
	}

	return ids, nil
}

func ReadTestGroupsFromDir(dir TaskDir) ([]TestGroup, error) {
	obj := struct {
		TestGroups []struct {
			GroupID int           `toml:"id"`
			Points  int           `toml:"points"`
			Public  bool          `toml:"public"`
			Tests   []interface{} `toml:"tests"`
		} `toml:"test_groups"`
	}{}

	err := toml.Unmarshal(dir.ProblemToml, &obj)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal test groups: %w", err)
	}

	if len(obj.TestGroups) == 0 {
		return nil, nil
	}

	// sort obj.TestGroups by GroupID
	sort.Slice(obj.TestGroups, func(i, j int) bool {
		return obj.TestGroups[i].GroupID < obj.TestGroups[j].GroupID
	})

	if obj.TestGroups[0].GroupID != 1 {
		return nil, fmt.Errorf("consecutive test group IDs must start with 1")
	}
	if obj.TestGroups[len(obj.TestGroups)-1].GroupID != len(obj.TestGroups) {
		return nil, fmt.Errorf("consecutive test group IDs must end with %d", len(obj.TestGroups))
	}

	testGroups := make([]TestGroup, 0, len(obj.TestGroups))
	for _, group := range obj.TestGroups {
		testGroup := TestGroup{
			Points:  group.Points,
			TestIDs: make([]int, 0, len(group.Tests)),
			Public:  group.Public,
		}
		testIDs, err := findTestIDs(dir, group.Tests)
		if err != nil {
			return nil, fmt.Errorf("failed to translate test references to IDs: %w", err)
		}
		testGroup.TestIDs = testIDs
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
