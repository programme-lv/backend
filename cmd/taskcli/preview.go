// preview.go
package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type TaskPreview struct {
	FullName           string
	Difficulty         int
	DiffTranslation    string
	CpuTimeLim         int
	MemoryLim          int
	OriginOlymp        string
	OlympTranslation   string
	OriginNotes        string
	TestCount          int
	TestTotalSize      int
	ExampleCount       int
	TestGroupCount     int
	TGrPointsRunLenEnc string
	VisInpStasks       []int
	PdfSttmntLangs     []string
	MdSttmntLangs      []string
	HasIllstrImg       string
	IllstrImgRelPath   string
}

func getMockPreview(dir string) TaskPreview {
	// Here, you would read from the directory using fstask.Read(dir)
	// For now, we return mock data
	return TaskPreview{
		FullName:           "Kvadrātveida putekļsūcējs",
		Difficulty:         3,
		DiffTranslation:    "vidējs",
		CpuTimeLim:         2,
		MemoryLim:          256,
		OriginOlymp:        "LIO",
		OlympTranslation:   "Latvijas informātikas olimpiāde",
		OriginNotes:        "Uzdevums tika izmantots Latvijas 37. informātikas olimpiādē",
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
	}
}

func (p TaskPreview) View() string {
	labelStyle := lipgloss.NewStyle()
	valueStyle := lipgloss.NewStyle().Bold(true)

	lines := []string{
		fmt.Sprintf("%s %s | %s %d (%s)",
			labelStyle.Render("Full name:"), valueStyle.Render(p.FullName),
			labelStyle.Render("Difficulty:"), p.Difficulty, p.DiffTranslation),
		fmt.Sprintf("%s %d s | %s %d MB",
			labelStyle.Render("CPU time limit:"), p.CpuTimeLim,
			labelStyle.Render("Memory limit:"), p.MemoryLim),
		fmt.Sprintf("%s %s (%s)",
			labelStyle.Render("Origin olympiad:"), p.OriginOlymp, p.OlympTranslation),
		fmt.Sprintf("%s %s",
			labelStyle.Render("Origin notes:"), p.OriginNotes),
		fmt.Sprintf("%s %d (total size: %d MB) | %s %d",
			labelStyle.Render("Test count:"), p.TestCount, p.TestTotalSize,
			labelStyle.Render("Example count:"), p.ExampleCount),
		fmt.Sprintf("%s %d (points r.l.e.: %s)",
			labelStyle.Render("Test groups:"), p.TestGroupCount, p.TGrPointsRunLenEnc),
		fmt.Sprintf("%s %v",
			labelStyle.Render("Visible input subtasks:"), p.VisInpStasks),
		fmt.Sprintf("%s %v | %s %v",
			labelStyle.Render("Pdf statement langs:"), p.PdfSttmntLangs,
			labelStyle.Render("Markdown statement langs:"), p.MdSttmntLangs),
		fmt.Sprintf("%s %s (assets/%s)",
			labelStyle.Render("Illustration image:"), p.HasIllstrImg, p.IllstrImgRelPath),
	}

	return strings.Join(lines, "\n")
}
