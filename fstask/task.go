package fstask

type Task struct {
	FullName             string
	VisibleInputSubtasks []int
	ProblemTags          []string
	DifficultyOneToFive  int
	OriginOlympiad       string
	OlympiadStage        string
	TaskAuthors          []string
	AcademicYear         string
	OriginInstitution    string
	MemoryLimitMegabytes int
	CPUTimeLimitSeconds  float64
	Solutions            []Solution
	Subtasks             []Subtask
	Examples             []Example
	MarkdownStatements   []MarkdownStatement
	PdfStatements        []PdfStatement
	Tests                []Test
	TestGroups           []TestGroup
	OriginNotes          map[string]string
	IllustrAssetFilename string
	Assets               []AssetFile
	ArchiveFiles         []ArchiveFile
	TestlibChecker       string
	TestlibInteractor    string
}

type Subtask struct {
	Points       int
	Tests        []string
	Descriptions map[string]string
}

type TestGroup struct {
	GroupID int
	Points  int
	Public  bool
	TestIDs []int
}

type Test struct {
	Input  []byte
	Answer []byte
	MdNote []byte
}

type Example struct {
	Input  []byte
	Output []byte
	MdNote []byte
}

func (t *Task) GetIllustrationImage() *AssetFile {
	if t.IllustrAssetFilename == "" {
		return nil
	}
	for _, asset := range t.Assets {
		if asset.RelativePath == t.IllustrAssetFilename {
			return &asset
		}
	}
	return nil
}
