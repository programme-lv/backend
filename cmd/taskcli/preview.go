// preview.go
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

func getMockPreview(dir string) (TaskPreview, error) {
	_ = dir
	// Here, you would read from the directory using fstask.Read(dir)
	// For now, we return mock data
	return TaskPreview{
		TaskDirectory:      "/home/kp/Programming/_PROGLV/task-workspace/proglv/kvadrputekl",
		FullName:           "Kvadrātveida putekļsūcējs",
		Difficulty:         3,
		DiffTranslation:    "vidējs",
		CpuTimeLim:         2,
		MemoryLim:          256,
		OriginOlymp:        "LIO",
		OlympTranslation:   "Latvijas informātikas olimpiāde",
		OriginNotes:        "Uzdevums tika izmantots Latvijas 37. informātikas olimpiādē. Uzdevums tika izmantots Latvijas 37. informātikas olimpiādē",
		TestCount:          98,
		TestTotalSize:      88,
		ExampleCount:       3,
		TestGroupCount:     7,
		TGrPointsRunLenEnc: "[3x3 4x2 3x7]",
		VisInpStasks:       []int{1},
		PdfSttmntLangs:     []string{"lv"},
		MdSttmntLangs:      []string{"lv"},
		HasIllstrImg:       "Yes",
		IllstrImgRelPath:   "illustration.png",
	}, nil
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
	}
	res.OlympTranslation = olympiadTranslations[res.OriginOlymp]

	res.OriginNotes = fmt.Sprintf("%v", task.OriginNotes)

	tests := task.GetTestsSortedByID()

	res.TestCount = len(tests)

	testTotalSize := 0
	for _, test := range tests {
		testTotalSize += len(test.Input) + len(test.Answer)
	}
	res.TestTotalSize = float64(testTotalSize) / (1024.0 * 1024.0)

	exapmles := task.GetExamples()

	res.ExampleCount = len(exapmles)

	testgroups := task.GetTestGroups()
	res.TestGroupCount = len(testgroups)

	// TODO help me finish here

	return res, nil
}

func (p TaskPreview) View() string {
	labelStyle := lipgloss.NewStyle()
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
			labelStyle.Render("Illustration image:"), valueStyle.Render(p.HasIllstrImg), p.IllstrImgRelPath),
	}

	for i := range len(lines) {
		lines[i] = lipgloss.NewStyle().Render(fmt.Sprintf("\t%s", lines[i]))
	}
	return strings.Join(lines, "\n")
}
