package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/programme-lv/backend/fstask"
)

func main() {
	dir := flag.String("dir", "", "directory path")
	flag.Parse()

	if *dir == "" {
		fmt.Println("Please provide a directory path using the -dir flag.")
		os.Exit(1)
	}

	if err := validateDirectory(*dir); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	absPath, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Printf("Error retrieving absolute path: %v\n", err)
		os.Exit(1)
	}

	task, err := fstask.Read(absPath)
	if err != nil {
		log.Fatal(err)
	}

	p := tea.NewProgram(initialModel(absPath, task))
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func validateDirectory(dir string) error {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist")
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory")
	}
	return nil
}

type (
	errMsg error
)

type model struct {
	err             error
	dirPath         string
	task            *fstask.Task
	testTotalCount  int
	testTotalSize   int
	testGroupPoints []int
	totalScore      int
	pdfSttmntLangs  []string
	mdSttmntLangs   []string
}

func initialModel(dirPath string, task *fstask.Task) model {
	tests := task.GetTestsSortedByID()
	testTotalCount := 0
	testTotalSize := 0
	for _, test := range tests {
		testTotalCount++
		testTotalSize += len(test.Answer)
		testTotalSize += len(test.Input)
	}

	groups := task.GetTestGroupIDs()
	testGroupPoints := make([]int, len(groups))
	for _, groupID := range groups {
		info := task.GetInfoOnTestGroup(groupID)
		testGroupPoints[groupID-1] = info.Points
	}

	totalScore := 0
	if len(groups) == 0 {
		totalScore = len(tests)
	} else {
		totalScore = 0
		for _, groupID := range groups {
			totalScore += testGroupPoints[groupID-1]
		}
	}

	pdfSttments := task.GetAllPDFStatements()
	pdfSttmntLangs := make([]string, len(pdfSttments))
	for i, pdfSttmnt := range pdfSttments {
		pdfSttmntLangs[i] = pdfSttmnt.Language
	}

	mdSttments := task.GetMarkdownStatements()
	mdSttmntLangs := make([]string, len(mdSttments))
	for i, mdSttmnt := range mdSttments {
		mdSttmntLangs[i] = mdSttmnt.Language
	}

	return model{
		err:             nil,
		dirPath:         dirPath,
		task:            task,
		testTotalCount:  testTotalCount,
		testTotalSize:   testTotalSize,
		testGroupPoints: testGroupPoints,
		totalScore:      totalScore,
		pdfSttmntLangs:  pdfSttmntLangs,
		mdSttmntLangs:   mdSttmntLangs,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}

	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, cmd
}

func (m model) View() string {
	difficultyMap := map[int]string{
		1: "very easy",
		2: "easy",
		3: "medium",
		4: "hard",
		5: "very hard",
	}

	blueText := lipgloss.NewStyle().Foreground(lipgloss.Color("#0000ff"))
	greenText := lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00"))

	illustrationImgPath := ""
	if m.task.GetTaskIllustrationImage() != nil {
		illustrationImgPath = m.task.GetTaskIllustrationImage().RelativePath
	}
	return fmt.Sprintf(`Directory: %s
Task preview:
	Full name: %s
	Cpu time limit: %s seconds
	Memory limit: %s MB
	Difficulty: %s (%s)
	Origin notes: %s
	Test count: %s (total size: %s MB)
	Test group count: %s (points: %s)
	Total score: %s points
	Visible input subtasks: %s
	Pdf statement langs: %s
	Markdown statement langs: %s
	Has illustration img: %s (%s)
	Example count: %s

Press Ctrl+C to cancel & exit
`,
		blueText.Render(m.dirPath),
		greenText.Render(m.task.FullName),
		greenText.Render(fmt.Sprintf("%.3f", m.task.CpuTimeLimInSeconds)),
		greenText.Render(fmt.Sprintf("%d", m.task.MemoryLimInMegabytes)),
		greenText.Render(fmt.Sprintf("%d", m.task.DifficultyOneToFive)),
		difficultyMap[m.task.DifficultyOneToFive],
		greenText.Render(fmt.Sprintf("%v", m.task.OriginNotes)),
		greenText.Render(fmt.Sprintf("%d", m.testTotalCount)),
		greenText.Render(fmt.Sprintf("%d", m.testTotalSize/1024/1024)),
		greenText.Render(fmt.Sprintf("%d", len(m.task.GetTestGroupIDs()))),
		greenText.Render(fmt.Sprintf("%v", m.testGroupPoints)),
		greenText.Render(fmt.Sprintf("%d", m.totalScore)),
		greenText.Render(fmt.Sprintf("%v", m.task.GetVisibleInputSubtasks())),
		greenText.Render(fmt.Sprintf("%v", m.pdfSttmntLangs)),
		greenText.Render(fmt.Sprintf("%v", m.mdSttmntLangs)),
		greenText.Render(fmt.Sprintf("%v", m.task.GetTaskIllustrationImage() != nil)),
		greenText.Render(fmt.Sprintf("%v", illustrationImgPath)),
		greenText.Render(fmt.Sprintf("%v", len(m.task.GetExamples()))),
	)
}
