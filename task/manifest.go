package task

import (
	"fmt"

	"github.com/pelletier/go-toml/v2"
	"golang.org/x/text/language"
)

type TaskManifest struct {
	FullName   string
	Contraints Constraints
	Statement  Statement
	Metadata   Metadata

	Tests      []Test
	TestGroups []TestGroup
}

type Metadata struct {
	ProblemTags       []string
	Difficulty        int
	TaskAuthors       []string
	OriginOlympiad    string
	OriginNotes       map[string]string
	OriginInstitution string
}

type Constraints struct {
	MemoryLimMB   int
	CpuTimeInSecs float64
}

type Test struct {
	InputSHA256  string
	AnswerSHA256 string
}

type Statement struct {
	PDFs      []PDF
	MDs       []MdStatement
	Examples  []Example
	VisInpSTs []VisibleInputSubtask

	IllustrationImg IllustrationImg
}

type IllustrationImg struct {
	S3ObjKey string
}

type VisibleInputSubtask struct {
	Subtask int
	Inputs  []string
}

type PDF struct {
	Language language.Tag
	SHA256   string
}

type MdStatement struct {
	Language language.Tag
	Story    MdSection
	Input    MdSection
	Output   MdSection
	Notes    MdSection
	Scoring  MdSection

	ImgUuidToS3Key map[string]string
}

type MdSection struct {
	Content string
}

type MdImage struct {
	S3ObjKey string
}

type Example struct {
	Input  string
	Output string
	MdNote *string
}

func ParseTaskTomlManifest(manifest string) (*TaskManifest, error) {
	taskTomlManifest := TaskTomlManifest{}
	err := toml.Unmarshal([]byte(manifest), &taskTomlManifest)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %v", err)
	}

	var mdStatements []MdStatement
	for _, md := range taskTomlManifest.MDStatements {
		lang := language.English
		if md.Language != nil {
			lang, err = language.Parse(*md.Language)
			if err != nil {
				return nil, fmt.Errorf("could not parse language %s", *md.Language)
			}
		}

		notes := ""
		if md.Notes != nil {
			notes = *md.Notes
		}

		scoring := ""
		if md.Scoring != nil {
			scoring = *md.Scoring
		}

		mdStatement := MdStatement{
			Language:       lang,
			Story:          MdSection{md.Story},
			Input:          MdSection{md.Input},
			Output:         MdSection{md.Output},
			Notes:          MdSection{notes},
			Scoring:        MdSection{scoring},
			ImgUuidToS3Key: taskTomlManifest.ImgUuidToObjKey,
		}

		mdStatements = append(mdStatements, mdStatement)
	}

	var tests []Test
	for _, test := range taskTomlManifest.TestSHA256s {
		tests = append(tests, Test{
			InputSHA256:  test.InputSHA256,
			AnswerSHA256: test.AnswerSHA256,
		})
	}

	var examples []Example = make([]Example, 0)
	for _, ex := range taskTomlManifest.Examples {
		var mdNote *string = nil
		if mdNoteStr := ex.MdNote; mdNoteStr != "" {
			mdNote = &mdNoteStr
		}

		examples = append(examples, Example{
			Input:  ex.Input,
			Output: ex.Output,
			MdNote: mdNote,
		})
	}

	var visInpSTs []VisibleInputSubtask = make([]VisibleInputSubtask, 0)
	for i, visInpST := range taskTomlManifest.VisibleInputSTs {
		visInpSTs = append(visInpSTs, VisibleInputSubtask{
			Subtask: visInpST,
			Inputs:  taskTomlManifest.VisInpStInputs[i].Inputs,
		})
	}

	var testGroups []TestGroup = make([]TestGroup, 0)
	for _, tg := range taskTomlManifest.TestGroups {
		testGroups = append(testGroups, TestGroup(tg))
	}

	var pdfs []PDF = make([]PDF, 0)
	for _, pdf := range taskTomlManifest.PDFSHA256s {
		lang, err := language.Parse(pdf.Language)
		if err != nil {
			return nil, fmt.Errorf("could not parse language %s", pdf.Language)
		}

		pdfs = append(pdfs, PDF{
			Language: lang,
			SHA256:   pdf.SHA256,
		})
	}

	return &TaskManifest{
		FullName: taskTomlManifest.TaskFullName,
		Contraints: Constraints{
			MemoryLimMB:   taskTomlManifest.MemoryLimMB,
			CpuTimeInSecs: taskTomlManifest.CpuTimeInSecs,
		},
		Statement: Statement{
			PDFs:      pdfs,
			MDs:       mdStatements,
			Examples:  examples,
			VisInpSTs: visInpSTs,
			IllustrationImg: IllustrationImg{
				S3ObjKey: taskTomlManifest.IllustrationImg,
			},
		},
		Metadata: Metadata{
			ProblemTags:       taskTomlManifest.ProblemTags,
			Difficulty:        taskTomlManifest.Difficulty,
			TaskAuthors:       taskTomlManifest.TaskAuthors,
			OriginOlympiad:    taskTomlManifest.OriginOlympiad,
			OriginNotes:       taskTomlManifest.OriginNotes,
			OriginInstitution: taskTomlManifest.OriginInstitution,
		},
		Tests:      tests,
		TestGroups: testGroups,
	}, nil
}

type TaskTomlManifest struct {
	TestSHA256s     []TestfileSHA256Ref     `toml:"tests_sha256s"`
	PDFSHA256s      []PDFStatemenSHA256tRef `toml:"pdf_statements_sha256s"`
	MDStatements    []TomlMDStatement       `toml:"md_statements"`
	ImgUuidToObjKey map[string]string       `toml:"img_uuid_to_obj_key"`

	TaskFullName    string      `toml:"task_full_name"`
	MemoryLimMB     int         `toml:"memory_lim_megabytes"`
	CpuTimeInSecs   float64     `toml:"cpu_time_in_seconds"`
	ProblemTags     []string    `toml:"problem_tags"`
	Difficulty      int         `toml:"difficulty_1_to_5"`
	TaskAuthors     []string    `toml:"task_authors"`
	OriginOlympiad  string      `toml:"origin_olympiad"`
	VisibleInputSTs []int       `toml:"visible_input_subtasks"`
	VisInpStInputs  []StInputs  `toml:"vis_inp_subtask_inputs"`
	TestGroups      []TestGroup `toml:"test_groups"`

	IllustrationImg string `toml:"illustration_img_s3objkey, omitempty"`

	OriginNotes       map[string]string `toml:"origin_notes,omitempty"`
	OriginInstitution string            `toml:"origin_institution,omitempty"`

	Examples []TomlExample `toml:"examples,omitempty"`
}

type StInputs struct {
	Subtask int      `toml:"subtask"`
	Inputs  []string `toml:"inputs,multiline"`
}

type TomlExample struct {
	Input  string `toml:"input"`
	Output string `toml:"output"`
	MdNote string `toml:"md_note,omitempty"`
}

type TestfileSHA256Ref struct {
	TestID       int    `toml:"test_id"`
	InputSHA256  string `toml:"input_sha256"`
	AnswerSHA256 string `toml:"answer_sha256"`
}

type PDFStatemenSHA256tRef struct {
	Language string `toml:"language"`
	SHA256   string `toml:"sha256"`
}

type TestGroup struct {
	GroupID int   `toml:"group_id"`
	Points  int   `toml:"points"`
	Public  bool  `toml:"public"`
	Subtask int   `toml:"subtask"`
	TestIDs []int `toml:"test_ids"`
}

type TomlMDStatement struct {
	Language *string `toml:"language"`
	Story    string  `toml:"story"`
	Input    string  `toml:"input"`
	Output   string  `toml:"output"`
	Notes    *string `toml:"notes"`
	Scoring  *string `toml:"scoring"`
}
