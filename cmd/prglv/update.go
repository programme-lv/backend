package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

type updatePlan struct {
	TaskID          string   `json:"task_id"`
	TestsVersion    int      `json:"tests_version"`
	RemoteTests     int      `json:"remote_tests"`
	LocalTests      int      `json:"local_tests"`
	TestsChanged    bool     `json:"tests_changed"`
	TaskChanged     bool     `json:"task_changed"`
	SkippedTestSize uint64   `json:"skipped_tests_bytes"`
	Sections        []string `json:"sections"`
	AddedFiles      []string `json:"added_files"`
	ModifiedFiles   []string `json:"modified_files"`
	DeletedFiles    []string `json:"deleted_files"`
}

func runUpdate(args []string) error {
	c := spec("update")
	flags, positional, err := parseFlags(c, args)
	if errors.Is(err, flag.ErrHelp) {
		printCommandHelp(os.Stdout, c)
		return nil
	}
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return fmt.Errorf("%w: update needs exactly one <task> path", errUsage)
	}

	apiKey := resolveAPIKey(flags.strs["api-key"])
	if apiKey == "" {
		return errors.New("missing admin API key; set PRGLV_API_KEY or ADMIN_API_KEY, or pass --api-key")
	}
	_, task, err := readTaskPackageWithTask(positional[0])
	if err != nil {
		return err
	}
	taskID := task.ID
	apiURL := resolveAPIURL(flags.strs["api-url"])
	manifest, err := getTaskManifest(context.Background(), apiURL, apiKey, taskID)
	if err != nil {
		return err
	}

	localSections, localFiles, localCount, testBytes, err := taskSectionDigests(task)
	if err != nil {
		return err
	}
	added, modified, deleted := diffFiles(manifest.Files, localFiles)
	var sections []string
	for _, section := range []string{
		"statement", "examples", "solutions", "tests", "readme", "checker", "interactor",
	} {
		if len(manifest.Sections) == 0 || manifest.Sections[section] != localSections[section] {
			sections = append(sections, section)
		}
	}
	testsChanged := len(manifest.Sections) == 0 ||
		localSections["tests"] != manifest.Sections["tests"]
	plan := updatePlan{
		TaskID: taskID, TestsVersion: manifest.TestsVersion,
		RemoteTests: manifest.TestsCount, LocalTests: localCount,
		TestsChanged: testsChanged, Sections: sections,
		TaskChanged: len(manifest.Sections) == 0 ||
			localSections["task"] != manifest.Sections["task"],
		AddedFiles: added, ModifiedFiles: modified, DeletedFiles: deleted,
	}
	if !testsChanged {
		plan.SkippedTestSize = testBytes
	}

	jsonOutput := flags.bools["json"]
	if flags.bools["dry-run"] {
		if jsonOutput {
			return json.NewEncoder(os.Stdout).Encode(plan)
		}
		printUpdatePlan(plan)
		return nil
	}
	if !jsonOutput {
		printUpdatePlan(plan)
	}
	if testsChanged && !flags.bools["allow-test-changes"] {
		return errors.New("official tests changed; pass --allow-test-changes to replace them")
	}
	if !jsonOutput && !flags.bools["yes"] {
		var ok bool
		var err error
		if plan.TestsChanged || (plan.TaskChanged && flags.bools["allow-test-changes"]) {
			ok, err = confirmDestructive(plan.TaskID)
		} else {
			ok, err = confirmUpdate()
		}
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("update cancelled")
		}
	}

	sectionSet := make(map[string]bool, len(sections))
	for _, section := range sections {
		sectionSet[section] = true
	}
	patchBytes, err := buildPatchZip(task, sectionSet)
	if err != nil {
		return err
	}
	meta := taskPatchMeta{
		BaseContentETag:  manifest.ContentETag,
		BaseTestsVersion: &manifest.TestsVersion,
		BaseRevision:     manifest.Revision,
		Sections:         sections,
		AllowTests:       flags.bools["allow-test-changes"],
	}
	result, err := updateTask(context.Background(), apiURL, apiKey, taskID, patchBytes, meta)
	if err != nil {
		return err
	}
	if jsonOutput {
		return json.NewEncoder(os.Stdout).Encode(result)
	}
	fmt.Printf("ok: updated %s (revision %d, tests version %d, tests changed %t)\n",
		result.TaskID, result.Revision, result.TestsVersion, result.TestsChanged)
	return nil
}

func printUpdatePlan(plan updatePlan) {
	testState := "unchanged"
	if plan.TestsChanged {
		testState = "changed"
	}
	fmt.Printf("task %s: tests version %d, remote %d tests\n",
		plan.TaskID, plan.TestsVersion, plan.RemoteTests)
	if plan.TestsChanged {
		fmt.Printf("official tests: %s; sending %d local tests\n", testState, plan.LocalTests)
	} else {
		fmt.Printf("official tests: identical; skipping %d files (%s)\n",
			plan.LocalTests, formatBytes(plan.SkippedTestSize))
	}
	if plan.TaskChanged {
		fmt.Println("task.toml: changed (may include limits or scoring)")
	} else {
		fmt.Println("task.toml: unchanged (sent as metadata)")
	}
	if len(plan.Sections) == 0 {
		fmt.Println("sending sections: none")
	} else {
		fmt.Printf("sending sections: %s\n", strings.Join(plan.Sections, ", "))
	}
	printFileChanges(plan)
}

func diffFiles(remote, local map[string]string) (added, modified, deleted []string) {
	if remote == nil {
		return nil, nil, nil
	}
	for name, sha := range local {
		old, ok := remote[name]
		if !ok {
			added = append(added, name)
		} else if old != sha {
			modified = append(modified, name)
		}
	}
	for name := range remote {
		if _, ok := local[name]; !ok {
			deleted = append(deleted, name)
		}
	}
	sort.Strings(added)
	sort.Strings(modified)
	sort.Strings(deleted)
	return added, modified, deleted
}

func printFileChanges(plan updatePlan) {
	if plan.AddedFiles == nil && plan.ModifiedFiles == nil && plan.DeletedFiles == nil {
		fmt.Println("file preview: unavailable (remote manifest has no file list)")
		return
	}
	const limit = 20
	printed := false
	for _, item := range []struct {
		label string
		files []string
	}{
		{"added", plan.AddedFiles},
		{"modified", plan.ModifiedFiles},
		{"deleted", plan.DeletedFiles},
	} {
		for i, name := range item.files {
			if i == limit {
				fmt.Printf("  ... %d more %s files\n", len(item.files)-limit, item.label)
				break
			}
			fmt.Printf("  %s: %s\n", item.label, name)
			printed = true
		}
	}
	if !printed {
		fmt.Println("file changes: none")
	}
}

func confirmUpdate() (bool, error) {
	info, err := os.Stdin.Stat()
	if err == nil && info.Mode()&os.ModeCharDevice == 0 {
		return false, errors.New("confirmation required; pass --yes")
	}
	fmt.Print("Apply? [y/N] ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func confirmDestructive(taskID string) (bool, error) {
	info, err := os.Stdin.Stat()
	if err == nil && info.Mode()&os.ModeCharDevice == 0 {
		return false, errors.New("confirmation required; pass --yes")
	}
	fmt.Printf("This replaces tests or scoring. Type %s to confirm: ", taskID)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	return strings.TrimSpace(line) == taskID, nil
}

func formatBytes(bytes uint64) string {
	const mib = 1 << 20
	if bytes >= mib {
		return fmt.Sprintf("%.1f MiB", float64(bytes)/mib)
	}
	return fmt.Sprintf("%.0f KiB", float64(bytes)/1024)
}
