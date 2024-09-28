package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/programme-lv/backend/fstask"
)

type TaskPreview struct {
	TaskDirectory      string
	FullName           string
	Difficulty         int
	DiffTranslation    string
	CpuTimeLim         float64
	MemoryLim          int
	OriginOlymp        string
	OlympTranslation   string
	OriginNotes        string
	TestCount          int
	TestTotalSize      float64
	ExampleCount       int
	TestGroupCount     int
	TGrPointsRunLenEnc string
	VisInpStasks       []int
	PdfSttmntLangs     []string
	MdSttmntLangs      []string
	HasIllstrImg       string
	IllstrImgRelPath   string
}

func getPreview(dir string) (TaskPreview, error) {
	res := TaskPreview{}

	task, err := fstask.Read(dir)
	if err != nil {
		return res, fmt.Errorf("failed to read task: %w", err)
	}

	res.TaskDirectory, err = filepath.Abs(dir)
	if err != nil {
		return res, fmt.Errorf("failed to get absolute path: %w", err)
	}

	res.FullName = strings.TrimSpace(task.FullName)
	res.Difficulty = task.DifficultyOneToFive

	diffTranslations := map[int]string{
		1: "ļoti viegls",
		2: "viegls",
		3: "vidējs",
		4: "grūts",
		5: "ļoti grūts",
	}
	res.DiffTranslation = diffTranslations[res.Difficulty]

	res.CpuTimeLim = task.CpuTimeLimInSeconds
	res.MemoryLim = task.MemoryLimInMegabytes

	res.OriginOlymp = task.OriginOlympiad

	olympiadTranslations := map[string]string{
		"LIO": "Latvijas informātikas olimpiāde",
		// Add more translations as needed
	}
	res.OlympTranslation = olympiadTranslations[res.OriginOlymp]

	res.OriginNotes = fmt.Sprintf("%v", task.OriginNotes)

	tests := task.GetTestsSortedByID()

	res.TestCount = len(tests)

	testTotalSize := 0
	for _, test := range tests {
		testTotalSize += len(test.Input) + len(test.Answer)
	}
	res.TestTotalSize = float64(testTotalSize) / (1024.0 * 1024.0) // Convert bytes to megabytes

	examples := task.GetExamples()

	res.ExampleCount = len(examples)

	testgroups := task.GetTestGroups()
	res.TestGroupCount = len(testgroups)

	// 1. Populate TGrPointsRunLenEnc (Run-Length Encoding of Test Group Points)
	type rleElement struct {
		count int
		ele   int
	}

	testGroupPoints := make([]int, len(testgroups))
	for i, group := range testgroups {
		testGroupPoints[i] = group.Points
	}

	var rle []rleElement
	if len(testGroupPoints) > 0 {
		rle = append(rle, rleElement{count: 1, ele: testGroupPoints[0]})
		for i := 1; i < len(testGroupPoints); i++ {
			if testGroupPoints[i] == rle[len(rle)-1].ele {
				rle[len(rle)-1].count++
			} else {
				rle = append(rle, rleElement{count: 1, ele: testGroupPoints[i]})
			}
		}
	}

	parts := make([]string, len(rle))
	for i, elem := range rle {
		parts[i] = fmt.Sprintf("%dx%d", elem.count, elem.ele)
	}
	res.TGrPointsRunLenEnc = "[" + strings.Join(parts, " ") + "]"

	// 2. Populate VisInpStasks (Visible Input Subtasks)
	visibleSubtasks := task.GetVisibleInputSubtaskIds()
	res.VisInpStasks = make([]int, len(visibleSubtasks))
	copy(res.VisInpStasks, visibleSubtasks)

	// 3. Populate PdfSttmntLangs (PDF Statement Languages)
	pdfStmts := task.PdfStatements
	res.PdfSttmntLangs = make([]string, len(pdfStmts))
	for i, stmt := range pdfStmts {
		res.PdfSttmntLangs[i] = stmt.Language
	}

	// 4. Populate MdSttmntLangs (Markdown Statement Languages)
	mdStmts := task.MarkdownStatements
	res.MdSttmntLangs = make([]string, len(mdStmts))
	for i, stmt := range mdStmts {
		res.MdSttmntLangs[i] = stmt.Language
	}

	// 5. Check for Illustration Image
	if task.GetTaskIllustrationImage() != nil {
		res.IllstrImgRelPath = task.GetTaskIllustrationImage().RelativePath
	}

	if res.IllstrImgRelPath != "" {
		res.HasIllstrImg = "Yes"
	} else {
		res.HasIllstrImg = "No"
	}

	return res, nil
}
func (p TaskPreview) View() string {
	labelStyle := lipgloss.NewStyle()
	// valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#3498db"))
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#3498db"))

	lines := []string{
		fmt.Sprintf("%s %s",
			labelStyle.Render("Task directory:"), valueStyle.Render(p.TaskDirectory)),
		fmt.Sprintf("%s %s | %s %s",
			labelStyle.Render("Full name:"), valueStyle.Render(p.FullName),
			labelStyle.Render("Difficulty:"), fmt.Sprintf("%s (%s)", valueStyle.Render(fmt.Sprintf("%d", p.Difficulty)), p.DiffTranslation)),
		fmt.Sprintf("%s %s | %s %s",
			labelStyle.Render("CPU time limit:"), fmt.Sprintf("%s s", valueStyle.Render(fmt.Sprintf("%.3f", p.CpuTimeLim))),
			labelStyle.Render("Memory limit:"), fmt.Sprintf("%s MB", valueStyle.Render(fmt.Sprintf("%d", p.MemoryLim)))),
		fmt.Sprintf("%s %s (%s)",
			labelStyle.Render("Origin olympiad:"), valueStyle.Render(p.OriginOlymp), p.OlympTranslation),
		fmt.Sprintf("%s %s",
			labelStyle.Render("Origin notes:"), valueStyle.Render(p.OriginNotes)),
		fmt.Sprintf("%s %s (total size: %s MB) | %s %s",
			labelStyle.Render("Test count:"), valueStyle.Render(fmt.Sprintf("%d", p.TestCount)), valueStyle.Render(fmt.Sprintf("%.3f", p.TestTotalSize)),
			labelStyle.Render("Example count:"), valueStyle.Render(fmt.Sprintf("%d", p.ExampleCount))),
		fmt.Sprintf("%s %s (points r.l.e.: %s)",
			labelStyle.Render("Test groups:"), valueStyle.Render(fmt.Sprintf("%d", p.TestGroupCount)), p.TGrPointsRunLenEnc),
		fmt.Sprintf("%s %s",
			labelStyle.Render("Visible input subtasks:"), valueStyle.Render(fmt.Sprintf("%v", p.VisInpStasks))),
		fmt.Sprintf("%s %s | %s %s",
			labelStyle.Render("Pdf statement langs:"), valueStyle.Render(fmt.Sprintf("%v", p.PdfSttmntLangs)),
			labelStyle.Render("Markdown statement langs:"), valueStyle.Render(fmt.Sprintf("%v", p.MdSttmntLangs))),
		fmt.Sprintf("%s %s (assets/%s)",
			labelStyle.Render("Illustration image:"), valueStyle.Render(p.HasIllstrImg), valueStyle.Render(p.IllstrImgRelPath)),
	}
	for i := range len(lines) {
		lines[i] = lipgloss.NewStyle().Render(fmt.Sprintf("\t%s", lines[i]))
	}
	return strings.Join(lines, "\n")
}
