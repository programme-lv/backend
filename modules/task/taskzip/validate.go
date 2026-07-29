package taskzip

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var upperCodeRE = regexp.MustCompile(`^[A-Z0-9]{1,10}$`)

func validate(task *Task) error {
	if task.Testing.Type == "interactor" {
		return ErrInteractive
	}
	if task.Version != 1 {
		return errors.New("taskzip must be 1")
	}
	if !idRE.MatchString(task.ID) {
		return fmt.Errorf("invalid task id %q", task.ID)
	}
	if err := validateMetadata(task); err != nil {
		return err
	}
	if err := validateContent(task); err != nil {
		return err
	}
	if err := validateScoring(task); err != nil {
		return err
	}
	return validateSolutions(task)
}

func validateMetadata(task *Task) error {
	if len(task.Name) == 0 {
		return errors.New("name needs at least one language")
	}
	for lang, name := range task.Name {
		if !langRE.MatchString(lang) || name == "" {
			return fmt.Errorf("invalid name entry %q", lang)
		}
	}
	switch task.Testing.Type {
	case "simple":
		if task.Checker != nil {
			return errors.New("checker.cpp present for simple task")
		}
	case "checker":
		if task.Checker == nil {
			return errors.New("checker.cpp missing")
		}
	case "interactor":
		return ErrInteractive
	default:
		return fmt.Errorf("unknown testing type %q", task.Testing.Type)
	}
	if task.Testing.CPUMs < 100 || task.Testing.CPUMs > 15000 {
		return errors.New("testing.cpu_ms out of range")
	}
	if task.Testing.MemMiB < 40 || task.Testing.MemMiB > 4096 {
		return errors.New("testing.mem_mib out of range")
	}
	if task.Origin != nil {
		if err := validateOrigin(task.Origin); err != nil {
			return err
		}
	}
	if task.Metadata != nil {
		if task.Metadata.Difficulty == nil {
			return errors.New("metadata.difficulty missing")
		}
		if *task.Metadata.Difficulty < 1 || *task.Metadata.Difficulty > 5 {
			return errors.New("metadata.difficulty out of range")
		}
	}
	return nil
}

func validateOrigin(origin *Origin) error {
	if origin.Olymp != "" && !upperCodeRE.MatchString(origin.Olymp) {
		return errors.New("invalid origin.olymp")
	}
	if origin.Org != "" && !upperCodeRE.MatchString(origin.Org) {
		return errors.New("invalid origin.org")
	}
	if origin.Stage != "" {
		valid := map[string]bool{
			"online": true, "school": true, "municipal": true, "national": true,
			"selection": true, "regional": true, "international": true,
		}
		if origin.Olymp == "" || !valid[origin.Stage] {
			return errors.New("invalid origin.stage")
		}
	}
	if origin.Year != nil && *origin.Year < 1980 {
		return errors.New("origin.year out of range")
	}
	if origin.Lang != "" && !langRE.MatchString(origin.Lang) {
		return errors.New("invalid origin.lang")
	}
	if origin.Solvers != nil && origin.Contestants != nil &&
		*origin.Solvers > *origin.Contestants {
		return errors.New("origin.solvers exceeds contestants")
	}
	return nil
}

func validateContent(task *Task) error {
	if len(task.Tests) == 0 {
		return errors.New("no official tests")
	}
	if len(task.Tests) > 999 {
		return errors.New("too many official tests")
	}
	for i, test := range task.Tests {
		if test.Input == nil || len(test.Input) == 0 || test.Output == nil {
			return fmt.Errorf("test %03d needs input and output", i+1)
		}
		if err := checkText(fmt.Sprintf("tests/%03di.txt", i+1), test.Input, true); err != nil {
			return err
		}
		if err := checkText(fmt.Sprintf("tests/%03do.txt", i+1), test.Output, false); err != nil {
			return err
		}
	}
	if len(task.Statements) == 0 {
		return errors.New("statement missing")
	}
	for lang, data := range task.Statements {
		if !langRE.MatchString(lang) {
			return fmt.Errorf("invalid statement language %q", lang)
		}
		if err := checkText("statement/"+lang+".md", data, false); err != nil {
			return err
		}
	}
	for name, data := range task.StatementImages {
		if strings.Contains(name, "/") || !allowedImage(name) {
			return fmt.Errorf("invalid statement image %q", name)
		}
		if strings.EqualFold(path.Ext(name), ".svg") && unsafeSVG(data) {
			return fmt.Errorf("unsanitized svg %s", name)
		}
	}
	if task.Readme != nil {
		if err := checkText("readme.md", task.Readme, false); err != nil {
			return err
		}
	}
	return validateExamples(task.Examples)
}

func validateExamples(examples []Example) error {
	if len(examples) > 20 {
		return errors.New("too many examples")
	}
	for i, example := range examples {
		if len(example.Input) == 0 || len(example.Output) == 0 {
			return fmt.Errorf("example %03d needs non-empty input and output", i+1)
		}
		if err := checkText(fmt.Sprintf("examples/%03di.txt", i+1), example.Input, true); err != nil {
			return err
		}
		if err := checkText(fmt.Sprintf("examples/%03do.txt", i+1), example.Output, false); err != nil {
			return err
		}
		for lang, note := range example.Notes {
			if lang != "" && !langRE.MatchString(lang) {
				return fmt.Errorf("invalid example note language %q", lang)
			}
			if err := checkText(fmt.Sprintf("examples/%03d.md", i+1), note, false); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateScoring(task *Task) error {
	if len(task.Subtasks) == 0 {
		if len(task.TestGroups) != 0 {
			return errors.New("tgroups.txt needs subtasks")
		}
		return nil
	}
	nextTest, nextGroup := uint32(1), uint32(1)
	for i, subtask := range task.Subtasks {
		for lang := range subtask.Description {
			if !langRE.MatchString(lang) {
				return fmt.Errorf("subtask %d: invalid language %q", i+1, lang)
			}
		}
		hasTests, hasGroups := subtask.Tests != "", subtask.Groups != ""
		if hasTests == hasGroups {
			return fmt.Errorf("subtask %d needs exactly one range", i+1)
		}
		if hasTests {
			if subtask.Points == nil || *subtask.Points == 0 {
				return fmt.Errorf("subtask %d needs positive points", i+1)
			}
			first, last, err := parseRange(subtask.Tests, 3)
			if err != nil || first != nextTest || last > uint32(len(task.Tests)) {
				return fmt.Errorf("subtask %d has invalid test range", i+1)
			}
			nextTest = last + 1
			continue
		}
		if subtask.Points != nil {
			return fmt.Errorf("subtask %d with groups must not declare points", i+1)
		}
		first, last, err := parseRange(subtask.Groups, 2)
		if err != nil || first != nextGroup || last > uint32(len(task.TestGroups)) {
			return fmt.Errorf("subtask %d has invalid group range", i+1)
		}
		for id := first; id <= last; id++ {
			group := task.TestGroups[id-1]
			if group.ID != id || group.First != nextTest || group.Last < group.First ||
				group.Last > uint32(len(task.Tests)) || group.Points == 0 {
				return fmt.Errorf("test group %02d is invalid", id)
			}
			nextTest = group.Last + 1
		}
		nextGroup = last + 1
	}
	if nextTest != uint32(len(task.Tests))+1 || nextGroup != uint32(len(task.TestGroups))+1 {
		return errors.New("subtasks do not partition tests and groups")
	}
	return nil
}

func validateSolutions(task *Task) error {
	seen := map[string]bool{}
	total := taskTotal(task)
	for _, solution := range task.Solutions {
		if solution.Filename == "" || path.Base(solution.Filename) != solution.Filename ||
			seen[solution.Filename] || solution.Data == nil {
			return fmt.Errorf("invalid solution %q", solution.Filename)
		}
		seen[solution.Filename] = true
		if solution.Score != nil && *solution.Score > total {
			return fmt.Errorf("solution %s score out of range", solution.Filename)
		}
		if len(task.Subtasks) == 0 && len(solution.Subtasks) != 0 {
			return fmt.Errorf("solution %s has subtasks for ungrouped task", solution.Filename)
		}
		for _, subtask := range solution.Subtasks {
			if subtask == 0 || subtask > uint32(len(task.Subtasks)) {
				return fmt.Errorf("solution %s has invalid subtask", solution.Filename)
			}
		}
	}
	return nil
}

func taskTotal(task *Task) uint32 {
	if len(task.Subtasks) == 0 {
		return uint32(len(task.Tests))
	}
	var total uint32
	for _, subtask := range task.Subtasks {
		if subtask.Points != nil {
			total += *subtask.Points
			continue
		}
		first, last, _ := parseRange(subtask.Groups, 2)
		for id := first; id <= last && id != 0; id++ {
			total += task.TestGroups[id-1].Points
		}
	}
	return total
}

func parseRange(value string, width int) (uint32, uint32, error) {
	parts := strings.Split(value, "-")
	if len(parts) != 2 || len(parts[0]) != width || len(parts[1]) != width {
		return 0, 0, errors.New("bad range")
	}
	first, err1 := strconv.ParseUint(parts[0], 10, 32)
	last, err2 := strconv.ParseUint(parts[1], 10, 32)
	if err1 != nil || err2 != nil || first == 0 || first > last {
		return 0, 0, errors.New("bad range")
	}
	return uint32(first), uint32(last), nil
}

func checkText(name string, data []byte, input bool) error {
	if !utf8.Valid(data) {
		return fmt.Errorf("%s must be UTF-8", name)
	}
	if input && len(data) == 0 {
		return fmt.Errorf("%s is empty", name)
	}
	if bytes.Contains(data, []byte{'\r'}) {
		return fmt.Errorf("%s must use LF endings", name)
	}
	for len(data) != 0 {
		r, size := utf8.DecodeRune(data)
		if (unicode.IsControl(r) && r != '\n' && r != '\t') ||
			r == '\u200b' || r == '\u200c' || r == '\u200d' || r == '\u2060' ||
			(r >= '\u202a' && r <= '\u202e') || (r >= '\u2066' && r <= '\u2069') {
			return fmt.Errorf("%s has forbidden character", name)
		}
		data = data[size:]
	}
	return nil
}

func unsafeSVG(data []byte) bool {
	lower := strings.ToLower(string(data))
	if strings.Contains(lower, "<script") || strings.Contains(lower, "<foreignobject") ||
		strings.Contains(lower, "javascript:") {
		return true
	}
	attr := regexp.MustCompile(`\son[a-z0-9_-]*\s*=`)
	if attr.MatchString(lower) {
		return true
	}
	ref := regexp.MustCompile(`\s(?:xlink:href|href|src)\s*=\s*["']([^"']*)["']`)
	for _, match := range ref.FindAllStringSubmatch(lower, -1) {
		value := match[1]
		if strings.HasPrefix(value, "/") || strings.HasPrefix(value, "../") ||
			strings.Contains(value, ":") {
			return true
		}
	}
	return false
}
