package taskzip

import "errors"

var (
	ErrInteractive = errors.New("interactive TaskZip is unsupported")
	ErrAttached    = errors.New("attached TaskZip files are unsupported")
)

type Task struct {
	Version         uint32
	ID              string
	Name            map[string]string
	Testing         Testing
	Readme          []byte
	Statements      map[string][]byte
	StatementImages map[string][]byte
	Examples        []Example
	Tests           []Test
	Checker         []byte
	Solutions       []Solution
	Subtasks        []Subtask
	TestGroups      []TestGroup
	Origin          *Origin
	Metadata        *Metadata
	Extensions      map[string]any
}

type Testing struct {
	Type   string `toml:"type"`
	CPUMs  uint32 `toml:"cpu_ms"`
	MemMiB uint32 `toml:"mem_mib"`
}

type Test struct {
	Input  []byte
	Output []byte
}

type Example struct {
	Input  []byte
	Output []byte
	Notes  map[string][]byte
}

type Solution struct {
	Filename string
	Subtasks []uint32
	Score    *uint32
	Data     []byte
}

type Subtask struct {
	Points       *uint32
	Tests        string
	Groups       string
	VisibleInput bool
	Description  map[string]string
}

type TestGroup struct {
	ID     uint32
	First  uint32
	Last   uint32
	Points uint32
	Public bool
}

type Origin struct {
	Olymp       string
	Year        *int
	Stage       string
	Org         string
	Authors     []string
	Lang        string
	Contestants *uint32
	Solvers     *uint32
}

type Metadata struct {
	Topics         []string
	Techniques     []string
	DataStructures []string
	Difficulty     *uint8
}
