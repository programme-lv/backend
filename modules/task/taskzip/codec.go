package taskzip

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

var (
	idRE      = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	langRE    = regexp.MustCompile(`^[a-z]{2,3}(?:-[a-z0-9]{2,8})*$`)
	testRE    = regexp.MustCompile(`^tests/(\d{3})([io])\.txt$`)
	exampleRE = regexp.MustCompile(`^examples/(\d{3})([io])\.txt$`)
	noteRE    = regexp.MustCompile(`^examples/(\d{3})\.md$`)
	groupRE   = regexp.MustCompile(`^(\d{2}): (\d{3})-(\d{3}) (\d+)p( \*)?$`)
)

const (
	maxImportFiles = 10_000
	maxImportSize  = uint64(512 << 20)
)

type taskTOML struct {
	TaskZip   uint32            `toml:"taskzip"`
	ID        string            `toml:"id"`
	Name      map[string]string `toml:"name"`
	Testing   Testing           `toml:"testing"`
	Solutions []solutionTOML    `toml:"solutions,omitempty"`
	Attached  []attachedTOML    `toml:"attached,omitempty"`
	Subtasks  []subtaskTOML     `toml:"subtasks,omitempty"`
	Origin    *originTOML       `toml:"origin,omitempty"`
	Metadata  *metadataTOML     `toml:"metadata,omitempty"`
	Ext       map[string]any    `toml:"ext,omitempty"`
}

type solutionTOML struct {
	Filename string   `toml:"fname"`
	Subtasks []uint32 `toml:"subtasks,omitempty"`
	Score    *uint32  `toml:"score,omitempty"`
}

type attachedTOML struct {
	Path string `toml:"path"`
}

type subtaskTOML struct {
	Points       *uint32           `toml:"points,omitempty"`
	Tests        string            `toml:"tests,omitempty"`
	Groups       string            `toml:"groups,omitempty"`
	VisibleInput bool              `toml:"vis_input,omitempty"`
	Description  map[string]string `toml:"description,omitempty"`
}

type originTOML struct {
	Olymp       string   `toml:"olymp,omitempty"`
	Year        *int     `toml:"year,omitempty"`
	Stage       string   `toml:"stage,omitempty"`
	Divisions   []string `toml:"divisions,omitempty"`
	Org         string   `toml:"org,omitempty"`
	Authors     []string `toml:"authors,omitempty"`
	Lang        string   `toml:"lang,omitempty"`
	Contestants *uint32  `toml:"contestants,omitempty"`
	Solvers     *uint32  `toml:"solvers,omitempty"`
}

type metadataTOML struct {
	Topics         []string `toml:"topics,omitempty"`
	Techniques     []string `toml:"techniques,omitempty"`
	DataStructures []string `toml:"data_structures,omitempty"`
	Difficulty     *uint8   `toml:"difficulty,omitempty"`
}

func Read(data []byte) (Task, error) {
	files, err := readZIP(data)
	if err != nil {
		return Task{}, err
	}
	raw, ok := files["task.toml"]
	if !ok {
		return Task{}, errors.New("task.toml missing")
	}
	var meta taskTOML
	dec := toml.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&meta); err != nil {
		return Task{}, fmt.Errorf("task.toml: %w", err)
	}
	if len(meta.Attached) != 0 || hasPrefix(files, "attached/") {
		return Task{}, ErrAttached
	}
	if meta.Testing.Type == "interactor" {
		return Task{}, ErrInteractive
	}
	task := fromTOML(meta)
	if err := consumeFiles(&task, files); err != nil {
		return Task{}, err
	}
	if err := validate(&task); err != nil {
		return Task{}, err
	}
	return task, nil
}

func Write(task Task) ([]byte, error) {
	if err := validate(&task); err != nil {
		return nil, err
	}
	meta := toTOML(task)
	raw, err := toml.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("task.toml: %w", err)
	}
	files, err := taskFiles(task, raw)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		h.SetModTime(time.Unix(0, 0).UTC())
		h.SetMode(0o644)
		w, err := zw.CreateHeader(h)
		if err == nil {
			_, err = w.Write(files[name])
		}
		if err != nil {
			_ = zw.Close()
			return nil, fmt.Errorf("write %s: %w", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("close zip: %w", err)
	}
	return out.Bytes(), nil
}

func readZIP(data []byte) (map[string][]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("zip: %w", err)
	}
	if len(zr.File) > maxImportFiles {
		return nil, errors.New("zip has too many entries")
	}
	flat := false
	for _, f := range zr.File {
		name, err := safePath(f.Name)
		if err != nil {
			return nil, err
		}
		flat = flat || name == "task.toml"
	}
	raw := make(map[string][]byte)
	seen := make(map[string]struct{})
	var totalSize uint64
	for _, f := range zr.File {
		name, err := safePath(f.Name)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[name]; ok {
			return nil, fmt.Errorf("duplicate path %q", name)
		}
		seen[name] = struct{}{}
		mode := f.Mode()
		if mode&(^mode.Perm()) != 0 && !mode.IsDir() {
			return nil, fmt.Errorf("unsupported zip entry %q", name)
		}
		if f.FileInfo().IsDir() {
			continue
		}
		logicalName := rootRelativePath(name, flat)
		if ignoredPath(logicalName) {
			continue
		}
		fileLimit := maxFileSize(logicalName)
		if f.UncompressedSize64 > fileLimit {
			return nil, fmt.Errorf("%s too large", name)
		}
		if f.UncompressedSize64 > maxImportSize-totalSize {
			return nil, errors.New("zip contents too large")
		}
		totalSize += f.UncompressedSize64
		b, err := readEntry(f, fileLimit)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		raw[name] = b
	}
	return resolveRoot(raw)
}

func rootRelativePath(name string, flat bool) string {
	if flat {
		return name
	}
	_, rest, ok := strings.Cut(name, "/")
	if !ok {
		return name
	}
	return rest
}

func ignoredPath(name string) bool {
	return strings.HasPrefix(name, "archive/") || strings.HasPrefix(name, "testspec/")
}

func safePath(name string) (string, error) {
	if name == "" || strings.Contains(name, `\`) || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("unsafe zip path %q", name)
	}
	clean := path.Clean(name)
	if clean == "." || clean != strings.TrimSuffix(name, "/") ||
		clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("unsafe zip path %q", name)
	}
	if len(clean) > 512 {
		return "", fmt.Errorf("zip path too long")
	}
	for _, part := range strings.Split(clean, "/") {
		if part == ".git" || part == "__MACOSX" || part == ".DS_Store" {
			return "", fmt.Errorf("forbidden path %q", name)
		}
	}
	return clean, nil
}

func resolveRoot(raw map[string][]byte) (map[string][]byte, error) {
	if _, ok := raw["task.toml"]; ok {
		return raw, nil
	}
	var wrapper string
	for name := range raw {
		first, rest, ok := strings.Cut(name, "/")
		if !ok || rest == "" {
			return nil, errors.New("zip must be flat or have one wrapper directory")
		}
		if wrapper == "" {
			wrapper = first
		} else if first != wrapper {
			return nil, errors.New("zip has multiple top-level directories")
		}
	}
	if wrapper == "" {
		return nil, errors.New("empty zip")
	}
	files := make(map[string][]byte, len(raw))
	for name, data := range raw {
		files[strings.TrimPrefix(name, wrapper+"/")] = data
	}
	metaRaw, ok := files["task.toml"]
	if !ok {
		return nil, errors.New("task.toml missing")
	}
	var id struct {
		ID string `toml:"id"`
	}
	if err := toml.Unmarshal(metaRaw, &id); err != nil {
		return nil, fmt.Errorf("task.toml: %w", err)
	}
	if wrapper != id.ID {
		return nil, fmt.Errorf("wrapper %q does not match task id %q", wrapper, id.ID)
	}
	return files, nil
}

func readEntry(f *zip.File, limit uint64) ([]byte, error) {
	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	data, err := io.ReadAll(io.LimitReader(r, int64(limit)+1))
	if err == nil && uint64(len(data)) > limit {
		return nil, errors.New("uncompressed data exceeds limit")
	}
	return data, err
}

func maxFileSize(name string) uint64 {
	switch {
	case name == "task.toml":
		return 1 << 20
	case name == "readme.md", noteRE.MatchString(name):
		return 512 << 10
	case name == "checker.cpp", name == "interactor.cpp":
		return 2 << 20
	case strings.HasPrefix(name, "tests/"):
		return 256 << 20
	case strings.HasPrefix(name, "examples/"):
		return 256 << 10
	case strings.HasPrefix(name, "statement/"):
		return 20 << 20
	default:
		return 256 << 20
	}
}

func fromTOML(m taskTOML) Task {
	t := Task{
		Version: m.TaskZip, ID: m.ID, Name: m.Name, Testing: m.Testing,
		Statements: map[string][]byte{}, StatementImages: map[string][]byte{},
		Extensions: m.Ext,
	}
	for _, s := range m.Solutions {
		t.Solutions = append(t.Solutions, Solution{
			Filename: s.Filename, Subtasks: s.Subtasks, Score: s.Score,
		})
	}
	for _, s := range m.Subtasks {
		t.Subtasks = append(t.Subtasks, Subtask{
			Points: s.Points, Tests: s.Tests, Groups: s.Groups,
			VisibleInput: s.VisibleInput, Description: s.Description,
		})
	}
	if m.Origin != nil {
		t.Origin = &Origin{
			Olymp: m.Origin.Olymp, Year: m.Origin.Year, Stage: m.Origin.Stage,
			Divisions: m.Origin.Divisions, Org: m.Origin.Org,
			Authors: m.Origin.Authors, Lang: m.Origin.Lang,
			Contestants: m.Origin.Contestants, Solvers: m.Origin.Solvers,
		}
	}
	if m.Metadata != nil {
		t.Metadata = &Metadata{
			Topics: m.Metadata.Topics, Techniques: m.Metadata.Techniques,
			DataStructures: m.Metadata.DataStructures, Difficulty: m.Metadata.Difficulty,
		}
	}
	return t
}

func toTOML(t Task) taskTOML {
	m := taskTOML{
		TaskZip: t.Version, ID: t.ID, Name: t.Name, Testing: t.Testing,
		Ext: t.Extensions,
	}
	for _, s := range t.Solutions {
		m.Solutions = append(m.Solutions, solutionTOML{
			Filename: s.Filename, Subtasks: s.Subtasks, Score: s.Score,
		})
	}
	for _, s := range t.Subtasks {
		m.Subtasks = append(m.Subtasks, subtaskTOML{
			Points: s.Points, Tests: s.Tests, Groups: s.Groups,
			VisibleInput: s.VisibleInput, Description: s.Description,
		})
	}
	if t.Origin != nil {
		m.Origin = &originTOML{
			Olymp: t.Origin.Olymp, Year: t.Origin.Year, Stage: t.Origin.Stage,
			Divisions: t.Origin.Divisions, Org: t.Origin.Org,
			Authors: t.Origin.Authors, Lang: t.Origin.Lang,
			Contestants: t.Origin.Contestants, Solvers: t.Origin.Solvers,
		}
	}
	if t.Metadata != nil {
		m.Metadata = &metadataTOML{
			Topics: t.Metadata.Topics, Techniques: t.Metadata.Techniques,
			DataStructures: t.Metadata.DataStructures, Difficulty: t.Metadata.Difficulty,
		}
	}
	return m
}

func hasPrefix(files map[string][]byte, prefix string) bool {
	for name := range files {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func number(match []string) (int, error) {
	n, err := strconv.Atoi(match[1])
	if err != nil || n == 0 {
		return 0, errors.New("index must be positive")
	}
	return n, nil
}
