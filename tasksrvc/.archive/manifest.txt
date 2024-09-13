package task

import (
	"fmt"

	"github.com/pelletier/go-toml/v2"
	"golang.org/x/text/language"
)

type TaskManifest struct {
	FullName   string
	Contraints ManifestConstraints
	Statement  ManifestStatement
	Metadata   ManifestMetadata

	Tests      []ManifestTest
	TestGroups []TomlTestGroup
}

type ManifestMetadata struct {
	ProblemTags       []string
	Difficulty        int
	TaskAuthors       []string
	OriginOlympiad    string
	OriginNotes       map[string]string
	OriginInstitution string
}

type ManifestConstraints struct {
	MemoryLimMB   int
	CpuTimeInSecs float64
}

type ManifestTest struct {
	InputSHA256  string
	AnswerSHA256 string
}

type ManifestStatement struct {
	PDFs      []ManifestPDF
	MDs       []ManifestMdStatement
	Examples  []ManifestExample
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

type ManifestPDF struct {
	Language language.Tag
	SHA256   string
}

type ManifestMdStatement struct {
	Language language.Tag
	Story    ManifestMdSection
	Input    ManifestMdSection
	Output   ManifestMdSection
	Notes    ManifestMdSection
	Scoring  ManifestMdSection

	ImgUuidToS3Key map[string]string
}

type ManifestMdSection struct {
	Content string
}

type ManifestMdImage struct {
	S3ObjKey string
}

type ManifestExample struct {
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

	var mdStatements []ManifestMdStatement
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

		mdStatement := ManifestMdStatement{
			Language:       lang,
			Story:          ManifestMdSection{md.Story},
			Input:          ManifestMdSection{md.Input},
			Output:         ManifestMdSection{md.Output},
			Notes:          ManifestMdSection{notes},
			Scoring:        ManifestMdSection{scoring},
			ImgUuidToS3Key: taskTomlManifest.ImgUuidToObjKey,
		}

		mdStatements = append(mdStatements, mdStatement)
	}

	var tests []ManifestTest
	for _, test := range taskTomlManifest.TestSHA256s {
		tests = append(tests, ManifestTest{
			InputSHA256:  test.InputSHA256,
			AnswerSHA256: test.AnswerSHA256,
		})
	}

	var examples []ManifestExample = make([]ManifestExample, 0)
	for _, ex := range taskTomlManifest.Examples {
		var mdNote *string = nil
		if mdNoteStr := ex.MdNote; mdNoteStr != "" {
			mdNote = &mdNoteStr
		}

		examples = append(examples, ManifestExample{
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

	var testGroups []TomlTestGroup = make([]TomlTestGroup, 0)
	for _, tg := range taskTomlManifest.TestGroups {
		testGroups = append(testGroups, TomlTestGroup(tg))
	}

	var pdfs []ManifestPDF = make([]ManifestPDF, 0)
	for _, pdf := range taskTomlManifest.PDFSHA256s {
		lang, err := language.Parse(pdf.Language)
		if err != nil {
			return nil, fmt.Errorf("could not parse language %s", pdf.Language)
		}

		pdfs = append(pdfs, ManifestPDF{
			Language: lang,
			SHA256:   pdf.SHA256,
		})
	}

	return &TaskManifest{
		FullName: taskTomlManifest.TaskFullName,
		Contraints: ManifestConstraints{
			MemoryLimMB:   taskTomlManifest.MemoryLimMB,
			CpuTimeInSecs: taskTomlManifest.CpuTimeInSecs,
		},
		Statement: ManifestStatement{
			PDFs:      pdfs,
			MDs:       mdStatements,
			Examples:  examples,
			VisInpSTs: visInpSTs,
			IllustrationImg: IllustrationImg{
				S3ObjKey: taskTomlManifest.IllustrationImg,
			},
		},
		Metadata: ManifestMetadata{
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

	TaskFullName    string          `toml:"task_full_name"`
	MemoryLimMB     int             `toml:"memory_lim_megabytes"`
	CpuTimeInSecs   float64         `toml:"cpu_time_in_seconds"`
	ProblemTags     []string        `toml:"problem_tags"`
	Difficulty      int             `toml:"difficulty_1_to_5"`
	TaskAuthors     []string        `toml:"task_authors"`
	OriginOlympiad  string          `toml:"origin_olympiad"`
	VisibleInputSTs []int           `toml:"visible_input_subtasks"`
	VisInpStInputs  []TomlStInputs  `toml:"vis_inp_subtask_inputs"`
	TestGroups      []TomlTestGroup `toml:"test_groups"`

	IllustrationImg string `toml:"illustration_img_s3objkey, omitempty"`

	OriginNotes       map[string]string `toml:"origin_notes,omitempty"`
	OriginInstitution string            `toml:"origin_institution,omitempty"`

	Examples []TomlExample `toml:"examples,omitempty"`
}

type TomlStInputs struct {
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

type TomlTestGroup struct {
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
