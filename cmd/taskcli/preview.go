package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/programme-lv/backend/fstask"
)

func renderTaskPreview(task *fstask.Task) string {
	wrapper := newTaskWrapper(task)

	difficultyMap := map[int]string{
		1: "very easy",
		2: "easy",
		3: "medium",
		4: "hard",
		5: "very hard",
	}

	g := func(format string, a ...any) string {
		greenText := lipgloss.NewStyle().Foreground(lipgloss.Color("#2ecc71"))
		return greenText.Render(fmt.Sprintf(format, a...))
	}

	illustrationImgPath := ""
	if task.GetTaskIllustrationImage() != nil {
		illustrationImgPath = task.GetTaskIllustrationImage().RelativePath
	}
	return fmt.Sprintf(`
	Full name: %s
	Cpu time limit: %s seconds | Memory limit: %s MB
	Difficulty: %s (%s)
	Origin notes: %s
	Test count: %s (total size: %s MB) | Example count: %s
	Test group count: %s (points: %s)
	Total score: %s points
	Visible input subtasks: %s
	Pdf statement langs: %s | Markdown statement langs: %s
	Has illustration img: %s (assets/%s)`,
		g(task.FullName),
		g("%.3f", task.CpuTimeLimInSeconds),
		g("%d", task.MemoryLimInMegabytes),
		g("%d", task.DifficultyOneToFive),
		difficultyMap[task.DifficultyOneToFive],
		g("%v", task.OriginNotes),
		g("%d", wrapper.GetTestTotalCount()),
		g("%d", wrapper.GetTestTotalSize()/1024/1024),
		g("%v", len(task.GetExamples())),
		g("%d", len(task.GetTestGroupIDs())),
		g("%v", wrapper.GetTestGroupPoints()),
		g("%d", wrapper.GetTotalScore()),
		g("%v", task.GetVisibleInputSubtasks()),
		g("%v", wrapper.GetPdfStatementLangs()),
		g("%v", wrapper.GetMdStatementLangs()),
		g("%v", task.GetTaskIllustrationImage() != nil),
		g("%v", illustrationImgPath),
	)
}
