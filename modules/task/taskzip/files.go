package taskzip

import (
	"bytes"
	"errors"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
)

func consumeFiles(task *Task, files map[string][]byte) error {
	tests := map[int]*Test{}
	examples := map[int]*Example{}
	solutions := map[string]*Solution{}
	for i := range task.Solutions {
		solutions[task.Solutions[i].Filename] = &task.Solutions[i]
	}
	for name, data := range files {
		switch {
		case name == "task.toml":
		case name == "readme.md":
			task.Readme = data
		case name == "checker.cpp":
			task.Checker = data
		case name == "interactor.cpp":
			return ErrInteractive
		case name == "tgroups.txt":
			groups, err := parseGroups(data)
			if err != nil {
				return err
			}
			task.TestGroups = groups
		case strings.HasPrefix(name, "archive/"), strings.HasPrefix(name, "testspec/"):
		case strings.HasPrefix(name, "attached/"):
			return ErrAttached
		case testRE.MatchString(name):
			m := testRE.FindStringSubmatch(name)
			n, err := number(m)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			if tests[n] == nil {
				tests[n] = &Test{}
			}
			if m[2] == "i" {
				tests[n].Input = data
			} else {
				tests[n].Output = data
			}
		case exampleRE.MatchString(name):
			m := exampleRE.FindStringSubmatch(name)
			n, err := number(m)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			if examples[n] == nil {
				examples[n] = &Example{}
			}
			if m[2] == "i" {
				examples[n].Input = data
			} else {
				examples[n].Output = data
			}
		case noteRE.MatchString(name):
			m := noteRE.FindStringSubmatch(name)
			n, err := number(m)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			if examples[n] == nil {
				examples[n] = &Example{}
			}
			notes, err := parseNotes(data)
			if err != nil {
				return fmt.Errorf("%s: %w", name, err)
			}
			examples[n].Notes = notes
		case strings.HasPrefix(name, "statement/"):
			base := strings.TrimPrefix(name, "statement/")
			if strings.Contains(base, "/") {
				return fmt.Errorf("unrecognized path %s", name)
			}
			if strings.HasSuffix(base, ".md") {
				task.Statements[strings.TrimSuffix(base, ".md")] = data
			} else if allowedImage(base) {
				task.StatementImages[base] = data
			} else {
				return fmt.Errorf("unsupported statement file %s", name)
			}
		case strings.HasPrefix(name, "solutions/"):
			fname := strings.TrimPrefix(name, "solutions/")
			s, ok := solutions[fname]
			if !ok || strings.Contains(fname, "/") {
				return fmt.Errorf("unlisted solution %s", name)
			}
			s.Data = data
		default:
			return fmt.Errorf("unrecognized path %s", name)
		}
	}
	task.Tests = orderedTests(tests)
	task.Examples = orderedExamples(examples)
	for _, s := range task.Solutions {
		if s.Data == nil {
			return fmt.Errorf("missing solutions/%s", s.Filename)
		}
	}
	return nil
}

func orderedTests(indexed map[int]*Test) []Test {
	out := make([]Test, len(indexed))
	for i, test := range indexed {
		if i >= 1 && i <= len(out) {
			out[i-1] = *test
		}
	}
	return out
}

func orderedExamples(indexed map[int]*Example) []Example {
	out := make([]Example, len(indexed))
	for i, example := range indexed {
		if i >= 1 && i <= len(out) {
			out[i-1] = *example
		}
	}
	return out
}

func parseNotes(data []byte) (map[string][]byte, error) {
	text := string(data)
	lines := strings.SplitAfter(text, "\n")
	if len(lines) < 2 || !langRE.MatchString(strings.TrimSuffix(lines[0], "\n")) ||
		strings.TrimSuffix(lines[1], "\n") != "---" {
		return map[string][]byte{"": data}, nil
	}
	notes := map[string][]byte{}
	for len(lines) > 0 {
		if len(lines) < 2 {
			return nil, errors.New("malformed language block")
		}
		lang := strings.TrimSuffix(lines[0], "\n")
		if !langRE.MatchString(lang) || strings.TrimSuffix(lines[1], "\n") != "---" {
			return nil, errors.New("malformed language block")
		}
		if _, exists := notes[lang]; exists {
			return nil, fmt.Errorf("duplicate language %s", lang)
		}
		lines = lines[2:]
		end := len(lines)
		for i := 0; i+1 < len(lines); i++ {
			tag := strings.TrimSuffix(lines[i], "\n")
			if langRE.MatchString(tag) && strings.TrimSuffix(lines[i+1], "\n") == "---" {
				end = i
				break
			}
		}
		notes[lang] = []byte(strings.Join(lines[:end], ""))
		lines = lines[end:]
	}
	return notes, nil
}

func renderNotes(notes map[string][]byte) ([]byte, error) {
	if len(notes) == 0 {
		return nil, nil
	}
	if raw, ok := notes[""]; ok {
		if len(notes) != 1 {
			return nil, errors.New("unmarked example note cannot be multilingual")
		}
		return raw, nil
	}
	langs := make([]string, 0, len(notes))
	for lang := range notes {
		if !langRE.MatchString(lang) {
			return nil, fmt.Errorf("invalid example note language %q", lang)
		}
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	var out bytes.Buffer
	for i, lang := range langs {
		if i != 0 && out.Len() != 0 && !bytes.HasSuffix(out.Bytes(), []byte("\n")) {
			out.WriteByte('\n')
		}
		fmt.Fprintf(&out, "%s\n---\n", lang)
		out.Write(notes[lang])
	}
	return out.Bytes(), nil
}

func parseGroups(data []byte) ([]TestGroup, error) {
	if err := checkText("tgroups.txt", data, false); err != nil {
		return nil, err
	}
	var groups []TestGroup
	for lineNo, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		m := groupRE.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("tgroups.txt line %d: bad format", lineNo+1)
		}
		id, idErr := strconv.ParseUint(m[1], 10, 32)
		first, firstErr := strconv.ParseUint(m[2], 10, 32)
		last, lastErr := strconv.ParseUint(m[3], 10, 32)
		points, pointsErr := strconv.ParseUint(m[4], 10, 32)
		if idErr != nil || firstErr != nil || lastErr != nil || pointsErr != nil ||
			id != uint64(len(groups)+1) || first > last || points == 0 {
			return nil, fmt.Errorf("tgroups.txt line %d: invalid group", lineNo+1)
		}
		groups = append(groups, TestGroup{
			ID: uint32(id), First: uint32(first), Last: uint32(last),
			Points: uint32(points), Public: m[5] != "",
		})
	}
	if len(groups) == 0 {
		return nil, errors.New("tgroups.txt has no groups")
	}
	return groups, nil
}

func renderGroups(groups []TestGroup) []byte {
	var out strings.Builder
	for _, group := range groups {
		fmt.Fprintf(&out, "%02d: %03d-%03d %dp", group.ID, group.First, group.Last, group.Points)
		if group.Public {
			out.WriteString(" *")
		}
		out.WriteByte('\n')
	}
	return []byte(out.String())
}

func taskFiles(task Task, meta []byte) (map[string][]byte, error) {
	files := map[string][]byte{"task.toml": meta}
	if task.Readme != nil {
		files["readme.md"] = task.Readme
	}
	for lang, statement := range task.Statements {
		files["statement/"+lang+".md"] = statement
	}
	for name, data := range task.StatementImages {
		files["statement/"+name] = data
	}
	for i, test := range task.Tests {
		files[fmt.Sprintf("tests/%03di.txt", i+1)] = test.Input
		files[fmt.Sprintf("tests/%03do.txt", i+1)] = test.Output
	}
	for i, example := range task.Examples {
		files[fmt.Sprintf("examples/%03di.txt", i+1)] = example.Input
		files[fmt.Sprintf("examples/%03do.txt", i+1)] = example.Output
		note, err := renderNotes(example.Notes)
		if err != nil {
			return nil, fmt.Errorf("example %03d: %w", i+1, err)
		}
		if note != nil {
			files[fmt.Sprintf("examples/%03d.md", i+1)] = note
		}
	}
	if task.Checker != nil {
		files["checker.cpp"] = task.Checker
	}
	for _, solution := range task.Solutions {
		files[path.Join("solutions", solution.Filename)] = solution.Data
	}
	if len(task.TestGroups) != 0 {
		files["tgroups.txt"] = renderGroups(task.TestGroups)
	}
	return files, nil
}

func allowedImage(name string) bool {
	ext := path.Ext(name)
	return ext == ".png" || ext == ".jpg" || ext == ".jpeg" ||
		ext == ".webp" || ext == ".svg"
}
