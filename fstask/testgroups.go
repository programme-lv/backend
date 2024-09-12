package fstask

import (
	"fmt"
	"log"

	"github.com/pelletier/go-toml/v2"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func readTGroupFnames(specVers string, tomlContent []byte, tGroupIDs []int) (map[int][]string, error) {
	res := make(map[int][]string, len(tGroupIDs))
	for i := 0; i < len(tGroupIDs); i++ {
		res[tGroupIDs[i]] = []string{}
	}

	semVerCmpRes, err := getCmpSemVersionsResult(specVers, "v2.2.0")
	if err != nil {
		return nil, fmt.Errorf("error comparing sem versions: %w", err)
	}

	if semVerCmpRes < 0 {
		return res, nil
	}

	type testGroupInfo struct {
		GroupID int      `toml:"group_id"`
		Fnames  []string `toml:"test_filenames"`
	}

	tomlStruct := struct {
		Groups []testGroupInfo `toml:"test_groups"`
	}{}

	err = toml.Unmarshal(tomlContent, &tomlStruct)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling test groups: %w", err)
	}

	for _, group := range tomlStruct.Groups {
		if _, ok := res[group.GroupID]; ok {
			res[group.GroupID] = group.Fnames
		}
	}

	return res, nil
}

func readTGroupTestIDs(specVers string, tomlContent []byte, tGroupIDs []int) (map[int][]int, error) {
	res := make(map[int][]int, len(tGroupIDs))
	for i := 0; i < len(tGroupIDs); i++ {
		res[tGroupIDs[i]] = []int{}
	}

	semVerCmpRes, err := getCmpSemVersionsResult(specVers, "v2.2.0")
	if err != nil {
		return nil, fmt.Errorf("error comparing sem versions: %w", err)
	}

	if semVerCmpRes < 0 {
		return res, nil
	}

	type testGroupInfo struct {
		GroupID int   `toml:"group_id"`
		TestIDs []int `toml:"test_ids"`
	}

	tomlStruct := struct {
		Groups []testGroupInfo `toml:"test_groups"`
	}{}

	err = toml.Unmarshal(tomlContent, &tomlStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal the test group IDs: %w", err)
	}

	for i := 0; i < len(tomlStruct.Groups); i++ {
		res[tomlStruct.Groups[i].GroupID] = tomlStruct.Groups[i].TestIDs
	}

	return res, nil
}

func readTGroupToStMap(specVers string, tomlContent []byte) (map[int]int, error) {
	semVerCmpRes, err := getCmpSemVersionsResult(specVers, "v2.2.0")
	if err != nil {
		return nil, fmt.Errorf("error comparing sem versions: %w", err)
	}

	if semVerCmpRes < 0 {
		return nil, nil
	}

	type testGroupInfo struct {
		GroupID int `toml:"group_id"`
		Subtask int `toml:"subtask"`
	}

	tomlStruct := struct {
		Groups []testGroupInfo `toml:"test_groups"`
	}{}

	err = toml.Unmarshal(tomlContent, &tomlStruct)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling test groups: %w", err)
	}

	res := make(map[int]int, len(tomlStruct.Groups))

	for _, group := range tomlStruct.Groups {
		if _, ok := res[group.GroupID]; ok {
			return nil, fmt.Errorf("duplicate group ID: %d", group.GroupID)
		}
		res[group.GroupID] = group.Subtask
	}

	return res, nil
}

func readTGroupPoints(specVers string, tomlContent []byte, tGroupIDs []int) (map[int]int, error) {
	res := make(map[int]int, len(tGroupIDs))

	for _, id := range tGroupIDs {
		res[id] = 0
	}

	semVerCmpRes, err := getCmpSemVersionsResult(specVers, "v2.2.0")
	if err != nil {
		return nil, fmt.Errorf("error comparing sem versions: %w", err)
	}

	if semVerCmpRes < 0 {
		return res, nil
	}

	type testGroupInfo struct {
		GroupID int `toml:"group_id"`
		Points  int `toml:"points"`
	}

	tomlStruct := struct {
		Groups []testGroupInfo `toml:"test_groups"`
	}{}

	err = toml.Unmarshal(tomlContent, &tomlStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal the test group points: %w", err)
	}

	for _, group := range tomlStruct.Groups {
		res[group.GroupID] = group.Points
	}

	return res, nil
}

func readTestGroupIDs(specVers string, tomlContent []byte) ([]int, error) {
	semVerCmpRes, err := getCmpSemVersionsResult(specVers, "v2.2.0")
	if err != nil {
		return nil, fmt.Errorf("error comparing sem versions: %w", err)
	}

	if semVerCmpRes < 0 {
		return nil, nil
	}

	type TestGroupID struct {
		TestGroupID int `toml:"group_id"`
	}

	tomlStruct := struct {
		Groups []TestGroupID `toml:"test_groups"`
	}{}

	err = toml.Unmarshal(tomlContent, &tomlStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal the test group IDs: %w", err)
	}

	res := make([]int, len(tomlStruct.Groups))

	for i, group := range tomlStruct.Groups {
		res[i] = group.TestGroupID
	}

	return res, nil
}

func readIsTGroupPublic(specVers string, tomlContent []byte, tGroupIDs []int) (map[int]bool, error) {
	res := make(map[int]bool, len(tGroupIDs))

	for _, id := range tGroupIDs {
		res[id] = true
	}

	semVerCmpRes, err := getCmpSemVersionsResult(specVers, "v2.2.0")
	if err != nil {
		return nil, fmt.Errorf("error comparing sem versions: %w", err)
	}

	if semVerCmpRes < 0 {
		return res, nil
	}

	type testGroupInfo struct {
		GroupID int  `toml:"group_id"`
		Public  bool `toml:"public"`
	}

	tomlStruct := struct {
		Groups []testGroupInfo `toml:"test_groups"`
	}{}

	err = toml.Unmarshal(tomlContent, &tomlStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal the test group public status: %w", err)
	}

	for _, group := range tomlStruct.Groups {
		_, ok := res[group.GroupID]
		if !ok {
			log.Printf("Warning: unknown test group ID: %d\n", group.GroupID)
			continue
		}
		res[group.GroupID] = group.Public
	}

	return res, nil
}
