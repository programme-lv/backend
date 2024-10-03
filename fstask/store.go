package fstask

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

const directorySpecificationVersion = "v3.0.0"

type ProblemTOML struct {
	Specification string           `toml:"specification"`
	TaskName      string           `toml:"task_name"`
	Origin        PTomlOrigin      `toml:"origin"`
	Constraints   PTomlConstraints `toml:"constraints"`
	IllustrImage  string           `toml:"illustration_image"`
	VisInpSTs     []int            `toml:"visible_input_subtasks"`
	Solutions     []PTomlSolution  `toml:"solutions"`
	ProblemTags   []string         `toml:"problem_tags"`
	Difficulty    int              `toml:"difficulty_1_to_5"`
	Subtasks      []PTomlSubtask   `toml:"subtasks"`
	TestGroups    []PTomlTestGroup `toml:"test_groups"`
}

type PTomlSubtask struct {
	ID           int               `toml:"id"`
	Points       int               `toml:"points"`
	Descriptions map[string]string `toml:"descriptions"`
	Tests        []int             `toml:"tests"`
}

type PTomlSolution struct {
	Filename string   `toml:"filename"`
	ScoreEq  *int     `toml:"score_eq"`
	ScoreLt  *int     `toml:"score_lt"`
	ScoreLte *int     `toml:"score_lte"`
	ScoreGt  *int     `toml:"score_gt"`
	ScoreGte *int     `toml:"score_gte"`
	Author   *string  `toml:"author"`
	ExecTime *float64 `toml:"exec_time"`
}

type PTomlOrigin struct {
	Olympiad     string   `toml:"olympiad"`
	AcademicYear string   `toml:"academic_year"`
	Stage        string   `toml:"stage"`
	Institution  string   `toml:"institution"`
	Authors      []string `toml:"authors"`

	Notes map[string]string `toml:"notes,omitempty"`
}

type PTomlConstraints struct {
	MemoryMegabytes int     `toml:"memory_megabytes"`
	CPUTimeSeconds  float64 `toml:"cpu_time_seconds"`
}

// PTomlTestGroup is a structure to store groups used in LIO test format
type PTomlTestGroup struct {
	GroupID int   `toml:"id"`
	Points  int   `toml:"points"`
	Public  bool  `toml:"public"`
	Tests   []int `toml:"tests"`
}

func (task *Task) encodeProblemTOML() ([]byte, error) {
	t := ProblemTOML{
		Specification: directorySpecificationVersion,
		TaskName:      task.FullName,
		Origin: PTomlOrigin{
			Olympiad:     task.OriginOlympiad,
			AcademicYear: task.AcademicYear,
			Stage:        task.OlympiadStage,
			Institution:  task.OriginInstitution,
			Authors:      task.TaskAuthors,
			Notes:        task.OriginNotes,
		},
		Constraints: PTomlConstraints{
			MemoryMegabytes: task.MemoryLimitMegabytes,
			CPUTimeSeconds:  task.CPUTimeLimitSeconds,
		},
		TestGroups:   []PTomlTestGroup{},
		IllustrImage: task.IllustrAssetFilename,
		VisInpSTs:    task.VisibleInputSubtasks,
		Solutions:    []PTomlSolution{},
		ProblemTags:  task.ProblemTags,
		Difficulty:   task.DifficultyOneToFive,
		Subtasks:     []PTomlSubtask{},
	}
	t.Specification = directorySpecificationVersion

	for i, st := range task.Subtasks {
		ptomlSubtask := PTomlSubtask{
			ID:           i + 1,
			Points:       st.Points,
			Descriptions: st.Descriptions,
			Tests:        st.TestIDs,
		}
		t.Subtasks = append(t.Subtasks, ptomlSubtask)
	}

	for i, tg := range task.TestGroups {
		ptomlTestGroup := PTomlTestGroup{
			GroupID: i + 1,
			Public:  tg.Public,
			Points:  tg.Points,
			Tests:   tg.TestIDs,
		}

		t.TestGroups = append(t.TestGroups, ptomlTestGroup)
	}

	for _, sol := range task.Solutions {
		ptomlSol := PTomlSolution{
			Filename: sol.Filename,
			ScoreEq:  sol.ScoreEq,
			ScoreLt:  sol.ScoreLt,
			ScoreLte: sol.ScoreLte,
			ScoreGt:  sol.ScoreGt,
			ScoreGte: sol.ScoreGte,
			Author:   sol.Author,
			ExecTime: sol.ExecTime,
		}
		t.Solutions = append(t.Solutions, ptomlSol)
	}

	buf := bytes.NewBuffer(make([]byte, 0))
	err := toml.NewEncoder(buf).
		SetTablesInline(false).
		// SetArraysMultiline(true).
		SetIndentTables(true).Encode(t)

	if err != nil {
		return nil, fmt.Errorf("failed to encode the problem.toml: %w", err)
	}

	return buf.Bytes(), nil
}

func (task *Task) Store(dirPath string) error {
	if _, err := os.Stat(dirPath); !os.IsNotExist(err) {
		return fmt.Errorf("directory already exists: %s", dirPath)
	}

	err := os.Mkdir(dirPath, 0755)
	if err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	err = task.storeProblemToml(filepath.Join(dirPath, "problem.toml"))
	if err != nil {
		return fmt.Errorf("error storing problem.toml: %w", err)
	}

	err = task.storeTests(filepath.Join(dirPath, "tests"))
	if err != nil {
		return fmt.Errorf("error storing tests: %w", err)
	}

	err = task.storeExamples(filepath.Join(dirPath, "examples"))
	if err != nil {
		return fmt.Errorf("error storing examples: %w", err)
	}

	err = task.storePDFStatementsShallow(filepath.Join(dirPath, "statements"))
	if err != nil {
		return fmt.Errorf("error storing PDF statements: %w", err)
	}

	err = task.storeMdStatementsShallow(filepath.Join(dirPath, "statements"))
	if err != nil {
		return fmt.Errorf("error storing Markdown statements: %w", err)
	}

	err = task.storeAssets(filepath.Join(dirPath, "assets"))
	if err != nil {
		return fmt.Errorf("error storing assets: %w", err)
	}

	err = task.storeSolutions(filepath.Join(dirPath, "solutions"))
	if err != nil {
		return fmt.Errorf("error storing solutions: %w", err)
	}

	err = task.storeArchiveFiles(filepath.Join(dirPath, "archive"))
	if err != nil {
		return fmt.Errorf("error storing archive files: %w", err)
	}

	err = task.storeCheckerAndInteractor(filepath.Join(dirPath, "evaluation"))
	if err != nil {
		return fmt.Errorf("error storing evaluation checker and interactor: %w", err)
	}

	return nil
}

func (task *Task) storeCheckerAndInteractor(dirPath string) error {
	if task.TestlibChecker == "" && task.TestlibInteractor == "" {
		return nil
	}

	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		return fmt.Errorf("error creating evaluation directory: %w", err)
	}

	if task.TestlibChecker != "" {
		path := filepath.Join(dirPath, "checker.cpp")
		err = os.WriteFile(path, []byte(task.TestlibChecker), 0644)
		if err != nil {
			return fmt.Errorf("error writing evaluation checker: %w", err)
		}
	}
	if task.TestlibInteractor != "" {
		path := filepath.Join(dirPath, "interactor.cpp")
		err = os.WriteFile(path, []byte(task.TestlibInteractor), 0644)
		if err != nil {
			return fmt.Errorf("error writing evaluation interactor: %w", err)
		}
	}

	return nil
}

func (task *Task) storeArchiveFiles(archiveDir string) error {
	if len(task.ArchiveFiles) == 0 {
		return nil
	}
	err := os.MkdirAll(archiveDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating archive directory: %w", err)
	}
	for _, v := range task.ArchiveFiles {
		path := filepath.Join(archiveDir, v.RelativePath)
		dir := filepath.Dir(path)
		// check if dir is subdir of archiveDir
		if !strings.HasPrefix(dir, archiveDir) {
			return fmt.Errorf("invalid archive file relative path: %s", v.RelativePath)
		}
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("error creating archive file directory: %w", err)
		}
		err = os.WriteFile(path, v.Content, 0644)
		if err != nil {
			return fmt.Errorf("error writing archive file: %w", err)
		}
	}
	return nil
}

func (task *Task) storeSolutions(solutionsDir string) error {
	if len(task.Solutions) == 0 {
		return nil
	}
	err := os.MkdirAll(solutionsDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating solutions directory: %w", err)
	}
	for _, v := range task.Solutions {
		err = os.WriteFile(filepath.Join(solutionsDir, v.Filename), []byte(v.Content), 0644)
		if err != nil {
			return fmt.Errorf("error writing solution: %w", err)
		}
	}
	return nil
}

func (task *Task) storeAssets(assetDir string) error {
	if len(task.Assets) == 0 {
		return nil
	}
	err := os.MkdirAll(assetDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating assets directory: %w", err)
	}

	for _, v := range task.Assets {
		path := filepath.Join(assetDir, v.RelativePath)
		err = os.WriteFile(path, v.Content, 0644)
		if err != nil {
			return fmt.Errorf("error writing asset: %w", err)
		}
	}
	return nil
}

func (task *Task) storeMdStatementsShallow(statementsDir string) error {
	if len(task.MarkdownStatements) == 0 {
		return nil
	}
	err := os.MkdirAll(statementsDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating Markdown statements directory: %w", err)
	}

	for _, v := range task.MarkdownStatements {
		filename := fmt.Sprintf("%s.md", v.Language)
		filePath := filepath.Join(statementsDir, filename)
		sections := []string{v.Story, v.Input, v.Output}
		contentStr := strings.Join(sections, "\n\n---\n\n")
		err = os.WriteFile(filePath, []byte(contentStr), 0644)
		if err != nil {
			return fmt.Errorf("error writing Markdown statement: %w", err)
		}
	}

	return nil
}

// func (task *Task) storeMdStatements(statementsDir string) error {
// 	if len(task.MarkdownStatements) == 0 {
// 		return nil
// 	}
// 	mdStatementDir := filepath.Join(statementsDir, "md")
// 	err := os.MkdirAll(mdStatementDir, 0755)
// 	if err != nil {
// 		return fmt.Errorf("error creating Markdown statements directory: %w", err)
// 	}

// 	for _, v := range task.MarkdownStatements {
// 		dirPath := filepath.Join(mdStatementDir, v.Language)
// 		err = os.MkdirAll(dirPath, 0755)
// 		if err != nil {
// 			return fmt.Errorf("error creating Markdown statement directory: %w", err)
// 		}

// 		inputPath := filepath.Join(dirPath, "input.md")
// 		outputPath := filepath.Join(dirPath, "output.md")
// 		storyPath := filepath.Join(dirPath, "story.md")
// 		scoringPath := filepath.Join(dirPath, "scoring.md")
// 		notesPath := filepath.Join(dirPath, "notes.md")

// 		if v.Input != "" {
// 			err = os.WriteFile(inputPath, []byte(v.Input), 0644)
// 			if err != nil {
// 				return fmt.Errorf("error writing Markdown statement: %w", err)
// 			}
// 		}

// 		if v.Output != "" {
// 			err = os.WriteFile(outputPath, []byte(v.Output), 0644)
// 			if err != nil {
// 				return fmt.Errorf("error writing Markdown statement: %w", err)
// 			}
// 		}

// 		if v.Story != "" {
// 			err = os.WriteFile(storyPath, []byte(v.Story), 0644)
// 			if err != nil {
// 				return fmt.Errorf("error writing Markdown statement: %w", err)
// 			}
// 		}

// 		if v.Scoring != "" {
// 			err = os.WriteFile(scoringPath, []byte(v.Scoring), 0644)
// 			if err != nil {
// 				return fmt.Errorf("error writing Markdown statement: %w", err)
// 			}
// 		}

// 		if v.Notes != "" {
// 			err = os.WriteFile(notesPath, []byte(v.Notes), 0644)
// 			if err != nil {
// 				return fmt.Errorf("error writing Markdown statement: %w", err)
// 			}
// 		}
// 	}

// 	return nil
// }

func (task *Task) storePDFStatementsShallow(statementsDir string) error {
	if len(task.PdfStatements) == 0 {
		return nil
	}
	err := os.MkdirAll(statementsDir, 0755)
	if err != nil {
		return fmt.Errorf("error creating PDF statements directory: %w", err)
	}
	for _, v := range task.PdfStatements {
		fname := fmt.Sprintf("%s.pdf", v.Language)
		fpath := filepath.Join(statementsDir, fname)
		err = os.WriteFile(fpath, []byte(v.Content), 0644)
		if err != nil {
			return fmt.Errorf("error writing PDF statement: %w", err)
		}
	}
	return nil
}

// func (task *Task) storePDFStatements(pdfStatementsDir string) error {
// 	if len(task.PdfStatements) == 0 {
// 		return nil
// 	}
// 	err := os.MkdirAll(pdfStatementsDir, 0755)
// 	if err != nil {
// 		return fmt.Errorf("error creating PDF statements directory: %w", err)
// 	}

// 	for _, v := range task.PdfStatements {
// 		fname := fmt.Sprintf("%s.pdf", v.Language)
// 		fpath := filepath.Join(pdfStatementsDir, fname)
// 		err = os.WriteFile(fpath, []byte(v.Content), 0644)
// 		if err != nil {
// 			return fmt.Errorf("error writing PDF statement: %w", err)
// 		}
// 	}

// 	return nil
// }

func (task *Task) storeProblemToml(problemTomlPath string) error {
	pToml, err := task.encodeProblemTOML()
	if err != nil {
		return fmt.Errorf("error encoding problem.toml: %w", err)
	}
	err = os.WriteFile(problemTomlPath, pToml, 0644)
	if err != nil {
		return fmt.Errorf("error writing problem.toml: %w", err)
	}
	return nil
}

func (task *Task) storeTests(testsDirPath string) error {
	var err error
	err = os.Mkdir(testsDirPath, 0755)
	if err != nil {
		return fmt.Errorf("error creating tests directory: %w", err)
	}

	for i, t := range task.Tests {
		fname := fmt.Sprintf("%03d", i+1)
		inPath := filepath.Join(testsDirPath, fname+".in")
		ansPath := filepath.Join(testsDirPath, fname+".out")

		err = os.WriteFile(inPath, t.Input, 0644)
		if err != nil {
			return fmt.Errorf("error writing input file: %w", err)
		}

		err = os.WriteFile(ansPath, t.Answer, 0644)
		if err != nil {
			return fmt.Errorf("error writing answer file: %w", err)
		}

	}

	return nil
}

func (task *Task) storeExamples(examplesDirPath string) error {
	var err error
	err = os.Mkdir(examplesDirPath, 0755)
	if err != nil {
		return fmt.Errorf("error creating examples directory: %w", err)
	}
	for i, e := range task.Examples {
		var inPath string
		var ansPath string
		var mdPath string

		inName := fmt.Sprintf("%03d.in", i+1)
		ansName := fmt.Sprintf("%03d.out", i+1)
		mdName := fmt.Sprintf("%03d.md", i+1)
		inPath = filepath.Join(examplesDirPath, inName)
		ansPath = filepath.Join(examplesDirPath, ansName)
		mdPath = filepath.Join(examplesDirPath, mdName)

		if _, err := os.Stat(inPath); err == nil {
			return fmt.Errorf("input file already exists: %s", inPath)
		}
		err = os.WriteFile(inPath, e.Input, 0644)
		if err != nil {
			return fmt.Errorf("error writing input file: %w", err)
		}

		if _, err := os.Stat(ansPath); err == nil {
			return fmt.Errorf("answer file already exists: %s", ansPath)
		}
		err = os.WriteFile(ansPath, e.Output, 0644)
		if err != nil {
			return fmt.Errorf("error writing answer file: %w", err)
		}

		if len(e.MdNote) > 0 {
			if _, err := os.Stat(mdPath); err == nil {
				return fmt.Errorf("markdown note file already exists: %s", mdPath)
			}

			err = os.WriteFile(mdPath, e.MdNote, 0644)
			if err != nil {
				return fmt.Errorf("error writing Markdown note file: %w", err)
			}
		}
	}
	return nil
}
