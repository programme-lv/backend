package srvc

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	taskzipv1 "github.com/programme-lv/backend/modules/task/taskzip"
)

func mapFromTaskZip(t taskzipv1.Task, overrideID string) (Task, error) {
	id := t.ID
	if overrideID != "" {
		id = overrideID
	}
	if err := validateTaskId(id); err != nil {
		return Task{}, fmt.Errorf("task id: %w", err)
	}
	res := Task{
		ShortId: id, FullName: t.Name, Readme: string(t.Readme),
		MemLimMegabytes: int(t.Testing.MemMiB),
		CpuTimeLimSecs:  float64(t.Testing.CPUMs) / 1000,
		IllustrImg:      &IllustrationImage{},
	}
	mapTaskZipOrigin(t, &res)
	mapTaskZipContent(t, &res)
	if err := mapTaskZipScoring(t, &res); err != nil {
		return Task{}, err
	}
	return res, nil
}

func mapTaskZipOrigin(t taskzipv1.Task, res *Task) {
	if t.Origin != nil {
		res.OriginOlympiad = t.Origin.Olymp
		res.OriginOrg = t.Origin.Org
		res.OlympStage = t.Origin.Stage
		res.Authors = t.Origin.Authors
		res.OrigLang = t.Origin.Lang
		if t.Origin.Year != nil {
			res.OriginYear = strconv.Itoa(*t.Origin.Year)
		}
	}
	if t.Metadata != nil {
		res.DifficultyRating = int(*t.Metadata.Difficulty)
		res.ProblemTags = appendAllowed(res.ProblemTags, t.Metadata.Topics, taskZipTopics)
		res.ProblemTags = appendAllowed(res.ProblemTags, t.Metadata.Techniques, taskZipTechniques)
		res.ProblemTags = appendAllowed(res.ProblemTags, t.Metadata.DataStructures, taskZipDataStructures)
	}
}

func mapTaskZipContent(t taskzipv1.Task, res *Task) {
	for lang, data := range t.Statements {
		res.MdStatements = append(res.MdStatements, parseTaskZipStatement(lang, data))
	}
	for _, example := range t.Examples {
		note := make(map[string]string, len(example.Notes))
		for lang, data := range example.Notes {
			note[lang] = string(data)
		}
		res.Examples = append(res.Examples, Example{
			Input: string(example.Input), Output: string(example.Output), MdNote: note,
		})
	}
	res.Checker = string(t.Checker)
	for _, solution := range t.Solutions {
		res.Solutions = append(res.Solutions, Solution{
			Fname: solution.Filename, Content: string(solution.Data),
			Subtasks: uint32sToInts(solution.Subtasks),
		})
	}
}

func mapTaskZipScoring(t taskzipv1.Task, res *Task) error {
	for _, group := range t.TestGroups {
		res.TestGroups = append(res.TestGroups, TestGroup{
			Points: int(group.Points), Public: group.Public,
			TestIDs: integerRange(int(group.First), int(group.Last)),
		})
	}
	for i, subtask := range t.Subtasks {
		testIDs, score, err := taskZipSubtaskTests(subtask, t.TestGroups)
		if err != nil {
			return fmt.Errorf("subtask %d: %w", i+1, err)
		}
		res.Subtasks = append(res.Subtasks, Subtask{
			Score: score, TestIDs: testIDs, Descriptions: subtask.Description,
		})
		if subtask.VisibleInput {
			res.VisInpSubtasks = append(res.VisInpSubtasks, visibleInput(i+1, testIDs, t.Tests))
		}
	}
	return nil
}

func taskZipSubtaskTests(
	subtask taskzipv1.Subtask, groups []taskzipv1.TestGroup,
) ([]int, int, error) {
	if subtask.Tests != "" {
		first, last, err := parseTaskZipRange(subtask.Tests)
		if err != nil {
			return nil, 0, err
		}
		if subtask.Points == nil {
			return nil, 0, fmt.Errorf("direct range needs points")
		}
		return integerRange(first, last), int(*subtask.Points), nil
	}
	first, last, err := parseTaskZipRange(subtask.Groups)
	if err != nil || first < 1 || last > len(groups) {
		return nil, 0, fmt.Errorf("invalid group range %q", subtask.Groups)
	}
	var tests []int
	score := 0
	for id := first; id <= last; id++ {
		group := groups[id-1]
		tests = append(tests, integerRange(int(group.First), int(group.Last))...)
		score += int(group.Points)
	}
	return tests, score, nil
}

func visibleInput(id int, testIDs []int, tests []taskzipv1.Test) VisibleInputSubtask {
	res := VisibleInputSubtask{SubtaskId: id}
	for _, testID := range testIDs {
		if testID > 0 && testID <= len(tests) {
			res.Tests = append(res.Tests, VisInpSubtaskTest{
				TestId: testID, Input: string(tests[testID-1].Input),
			})
		}
	}
	return res
}

func mapToTaskZip(t Task) (taskzipv1.Task, error) {
	if t.DifficultyRating < 0 || t.DifficultyRating > 5 {
		return taskzipv1.Task{}, fmt.Errorf("difficulty out of range")
	}
	if t.Checker != "" && t.Interactor != "" {
		return taskzipv1.Task{}, fmt.Errorf("checker and interactor both set")
	}
	testingType := "simple"
	if t.Checker != "" {
		testingType = "checker"
	} else if t.Interactor != "" {
		testingType = "interactor"
	}
	cpu, err := uint32Value(t.CpuMillis(), "CPU limit")
	if err != nil {
		return taskzipv1.Task{}, err
	}
	mem, err := uint32Value(t.MemLimMegabytes, "memory limit")
	if err != nil {
		return taskzipv1.Task{}, err
	}
	res := taskzipv1.Task{
		Version: 1, ID: t.ShortId, Name: t.FullName,
		Testing: taskzipv1.Testing{Type: testingType, CPUMs: cpu, MemMiB: mem},
		Readme:  []byte(t.Readme), Statements: map[string][]byte{},
		StatementImages: map[string][]byte{},
	}
	if t.Checker != "" {
		res.Checker = []byte(t.Checker)
	}
	mapServiceOrigin(t, &res)
	mapServiceContent(t, &res)
	if err := mapServiceScoring(t, &res); err != nil {
		return taskzipv1.Task{}, err
	}
	return res, nil
}

func mapServiceOrigin(t Task, res *taskzipv1.Task) {
	origin := &taskzipv1.Origin{
		Olymp: t.OriginOlympiad, Stage: t.OlympStage, Org: t.OriginOrg,
		Authors: t.Authors, Lang: t.OrigLang,
	}
	if year, err := strconv.Atoi(t.OriginYear); err == nil {
		origin.Year = &year
	}
	if origin.Olymp != "" || origin.Stage != "" || origin.Org != "" ||
		len(origin.Authors) != 0 || origin.Lang != "" || origin.Year != nil {
		res.Origin = origin
	}
	if t.DifficultyRating != 0 {
		difficulty := uint8(t.DifficultyRating)
		res.Metadata = &taskzipv1.Metadata{
			Topics: t.ProblemTags, Difficulty: &difficulty,
		}
	}
}

func mapServiceContent(t Task, res *taskzipv1.Task) {
	for _, statement := range t.MdStatements {
		res.Statements[statement.LangIso639] = renderTaskZipStatement(statement)
	}
	for _, example := range t.Examples {
		notes := make(map[string][]byte, len(example.MdNote))
		for lang, note := range example.MdNote {
			notes[lang] = []byte(note)
		}
		res.Examples = append(res.Examples, taskzipv1.Example{
			Input: []byte(example.Input), Output: []byte(example.Output), Notes: notes,
		})
	}
	for _, solution := range t.Solutions {
		res.Solutions = append(res.Solutions, taskzipv1.Solution{
			Filename: solution.Fname, Data: []byte(solution.Content),
			Subtasks: intsToUint32s(solution.Subtasks),
		})
	}
}

func mapServiceScoring(t Task, res *taskzipv1.Task) error {
	for i, group := range t.TestGroups {
		first, last, err := contiguousRange(group.TestIDs)
		if err != nil {
			return fmt.Errorf("test group %d: %w", i+1, err)
		}
		points, err := uint32Value(group.Points, "test group points")
		if err != nil {
			return err
		}
		res.TestGroups = append(res.TestGroups, taskzipv1.TestGroup{
			ID: uint32(i + 1), First: uint32(first), Last: uint32(last),
			Points: points, Public: group.Public,
		})
	}
	return mapServiceSubtasks(t, res)
}

func mapServiceSubtasks(t Task, res *taskzipv1.Task) error {
	visible := make(map[int]bool, len(t.VisInpSubtasks))
	for _, subtask := range t.VisInpSubtasks {
		visible[subtask.SubtaskId] = true
	}
	nextGroup := 0
	for i, subtask := range t.Subtasks {
		mapped, used, err := mapServiceSubtask(subtask, t.TestGroups[nextGroup:], nextGroup)
		if err != nil {
			return fmt.Errorf("subtask %d: %w", i+1, err)
		}
		mapped.VisibleInput = visible[i+1]
		res.Subtasks = append(res.Subtasks, mapped)
		nextGroup += used
	}
	if nextGroup != len(t.TestGroups) {
		return fmt.Errorf("test groups do not match subtasks")
	}
	return nil
}

func mapServiceSubtask(subtask Subtask, groups []TestGroup, groupOffset int) (taskzipv1.Subtask, int, error) {
	mapped := taskzipv1.Subtask{Description: subtask.Descriptions}
	if used := matchingGroups(subtask.TestIDs, groups); used != 0 {
		mapped.Groups = formatTaskZipRange(groupOffset+1, groupOffset+used, 2)
		return mapped, used, nil
	}
	first, last, err := contiguousRange(subtask.TestIDs)
	if err != nil {
		return taskzipv1.Subtask{}, 0, err
	}
	points, err := uint32Value(subtask.Score, "subtask points")
	if err != nil {
		return taskzipv1.Subtask{}, 0, err
	}
	mapped.Points = &points
	mapped.Tests = formatTaskZipRange(first, last, 3)
	return mapped, 0, nil
}

func matchingGroups(testIDs []int, groups []TestGroup) int {
	var grouped []int
	for i, group := range groups {
		grouped = append(grouped, group.TestIDs...)
		if equalInts(grouped, testIDs) {
			return i + 1
		}
		if len(grouped) >= len(testIDs) || !equalInts(grouped, testIDs[:len(grouped)]) {
			return 0
		}
	}
	return 0
}

func parseTaskZipRange(value string) (int, int, error) {
	parts := strings.Split(value, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid range %q", value)
	}
	first, err1 := strconv.Atoi(parts[0])
	last, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || first < 1 || first > last {
		return 0, 0, fmt.Errorf("invalid range %q", value)
	}
	return first, last, nil
}

func contiguousRange(ids []int) (int, int, error) {
	if len(ids) == 0 {
		return 0, 0, fmt.Errorf("empty test range")
	}
	for i := 1; i < len(ids); i++ {
		if ids[i] != ids[i-1]+1 {
			return 0, 0, fmt.Errorf("non-contiguous test range")
		}
	}
	return ids[0], ids[len(ids)-1], nil
}

func formatTaskZipRange(first, last, width int) string {
	return fmt.Sprintf("%0*d-%0*d", width, first, width, last)
}

func integerRange(first, last int) []int {
	if first < 1 || first > last {
		return nil
	}
	res := make([]int, last-first+1)
	for i := range res {
		res[i] = first + i
	}
	return res
}

func uint32Value(value int, name string) (uint32, error) {
	if value < 0 || uint64(value) > math.MaxUint32 {
		return 0, fmt.Errorf("%s out of range", name)
	}
	return uint32(value), nil
}

func uint32sToInts(values []uint32) []int {
	res := make([]int, len(values))
	for i, value := range values {
		res[i] = int(value)
	}
	return res
}

func intsToUint32s(values []int) []uint32 {
	res := make([]uint32, len(values))
	for i, value := range values {
		res[i] = uint32(value)
	}
	return res
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

var taskZipTopics = stringSet(
	"implementation", "arrays", "strings", "sorting-searching", "mathematics",
	"number-theory", "combinatorics", "graphs", "trees", "grids", "geometry",
	"data-structures", "dynamic-programming", "bitwise", "games", "construction",
	"interactive",
)

var taskZipTechniques = stringSet(
	"brute-force", "simulation", "sorting", "binary-search", "two-pointers",
	"sliding-window", "prefix-sums", "difference-array", "greedy", "recursion",
	"backtracking", "divide-and-conquer", "meet-in-the-middle",
	"coordinate-compression", "sweep-line", "bfs", "dfs", "flood-fill",
	"shortest-paths", "dijkstra", "bellman-ford", "floyd-warshall",
	"topological-sort", "strongly-connected-components", "minimum-spanning-tree",
	"euler-tour", "lca", "tree-dp", "max-flow", "matching", "dp", "knapsack-dp",
	"interval-dp", "bitmask-dp", "digit-dp", "dp-optimization",
	"modular-arithmetic", "gcd", "sieve", "primes", "combinatorics", "probability",
	"matrix-exponentiation", "game-theory", "string-matching", "hashing", "kmp",
	"z-function", "trie", "suffix-array", "convex-hull", "point-line-geometry",
	"polygon-geometry",
)

var taskZipDataStructures = stringSet(
	"array", "stack", "queue", "deque", "map-set", "priority-queue", "dsu",
	"fenwick-tree", "segment-tree", "lazy-segment-tree", "sparse-table",
	"ordered-set", "bitset", "trie",
)

var statementHeadings = map[string]string{
	"story": "story", "stāsts": "story",
	"input": "input", "ievaddati": "input",
	"output": "output", "izvaddati": "output",
	"notes": "notes", "piezīmes": "notes",
	"scoring": "scoring", "vērtēšana": "scoring",
	"communication": "talk", "komunikācija": "talk",
	"example": "example", "piemērs": "example",
}

func stringSet(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func appendAllowed(dst, values []string, allowed map[string]struct{}) []string {
	for _, value := range values {
		if _, ok := allowed[value]; ok {
			dst = append(dst, value)
		}
	}
	return dst
}

func parseTaskZipStatement(lang string, data []byte) MarkdownStatement {
	statement := MarkdownStatement{LangIso639: lang}
	lines := strings.Split(string(data), "\n")
	start, section, found := 0, "story", false
	for i := 0; i+1 < len(lines); i++ {
		next, ok := statementHeading(lines[i], lines[i+1])
		if !ok {
			continue
		}
		setStatementSection(&statement, section, strings.Join(lines[start:i], "\n"))
		section, start, found = next, i+2, true
		i++
	}
	if !found {
		statement.Story = string(data)
		return statement
	}
	setStatementSection(&statement, section, strings.Join(lines[start:], "\n"))
	return statement
}

func statementHeading(title, underline string) (string, bool) {
	key, ok := statementHeadings[strings.ToLower(strings.TrimSpace(title))]
	underline = strings.TrimSpace(underline)
	if !ok || len(underline) < 3 || strings.Trim(underline, "-") != "" {
		return "", false
	}
	return key, true
}

func setStatementSection(statement *MarkdownStatement, section, value string) {
	value = strings.Trim(value, "\n")
	switch section {
	case "story":
		statement.Story = value
	case "input":
		statement.Input = value
	case "output":
		statement.Output = value
	case "notes":
		statement.Notes = value
	case "scoring":
		statement.Scoring = value
	case "talk":
		statement.Talk = value
	case "example":
		statement.Example = value
	}
}

func renderTaskZipStatement(statement MarkdownStatement) []byte {
	labels := []string{"Story", "Input", "Output", "Notes", "Scoring", "Communication", "Example"}
	if statement.LangIso639 == "lv" || strings.HasPrefix(statement.LangIso639, "lv-") {
		labels = []string{"Stāsts", "Ievaddati", "Izvaddati", "Piezīmes", "Vērtēšana", "Komunikācija", "Piemērs"}
	}
	var parts []string
	values := []string{
		statement.Story, statement.Input, statement.Output, statement.Notes,
		statement.Scoring, statement.Talk, statement.Example,
	}
	for i, value := range values {
		if value != "" || i == 0 {
			parts = append(parts, labels[i]+"\n"+strings.Repeat("-", len([]rune(labels[i])))+"\n"+value)
		}
	}
	return []byte(strings.Trim(strings.Join(parts, "\n\n"), "\n") + "\n")
}
