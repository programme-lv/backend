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
	"github.com/joho/godotenv"
	"github.com/programme-lv/backend/fstask"
	"github.com/programme-lv/backend/task"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("Error loading .env file")
	}

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

type phase int

const (
	phaseEnterTaskId phase = iota
	phaseConfirmUpload
	phaseWaitingUploadRes
)

type model struct {
	errMsg  string
	phase   phase
	err     error
	dirPath string

	task *fstask.Task

	shortCodeInput textinput.Model
	inputingTaskId bool
}

func initialModel(dirPath string, task *fstask.Task) model {
	ti := textinput.New()
	ti.SetValue(filepath.Base(dirPath))
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return model{
		phase:          phaseEnterTaskId,
		err:            nil,
		dirPath:        dirPath,
		task:           task,
		shortCodeInput: ti,
		inputingTaskId: true,
	}
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

type uploadResult struct {
	err error
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case uploadResult:
		log.Printf("upload finished %+v", msg.err)
		return m, tea.Quit
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.phase == phaseEnterTaskId {
				m.inputingTaskId = false
				m.shortCodeInput.Blur()
				m.shortCodeInput.CursorEnd()
				m.phase = phaseConfirmUpload
				return m, cmd
			}
		case tea.KeyRunes:
			switch msg.Runes[0] {
			case 'y', 'Y':
				return m, func() tea.Msg {
					err := task.NewTaskSrvc().CreateTask(&task.CreatePublicTaskInput{
						TaskCode:    m.shortCodeInput.Value(),
						FullName:    m.task.FullName,
						MemMBytes:   m.task.MemoryLimInMegabytes,
						CpuSecs:     m.task.CpuTimeLimInSeconds,
						Difficulty:  &m.task.DifficultyOneToFive,
						OriginOlymp: m.task.OriginOlympiad,
						IllustrKey:  new(string),
						VisInpSts: []struct {
							Subtask int
							Inputs  []string
						}{},
						TestGroups: []struct {
							GroupID int
							Points  int
							Public  bool
							Subtask int
							TestIDs []int
						}{},
						TestChsums: []struct {
							TestID  int
							InSha2  string
							AnsSha2 string
						}{},
						PdfSttments: []struct {
							LangIso639 string
							PdfSha2    string
						}{},
						MdSttments: []struct {
							LangIso639 string
							Story      string
							Input      string
							Output     string
							Score      string
						}{},
						ImgUuidMap: []struct {
							Uuid  string
							S3Key string
						}{},
						Examples: []struct {
							ExampleID int
							Input     string
							Output    string
						}{},
						OriginNotes: []struct {
							LangIso639 string
							OgInfo     string
						}{},
					})

					return uploadResult{err: err}
				}
			case 'n', 'N':
				return m, tea.Quit
			}
		}
	}

	if m.inputingTaskId {
		m.shortCodeInput, cmd = m.shortCodeInput.Update(msg)
	}
	return m, cmd
}

func (m model) View() string {
	b := func(format string, a ...any) string {
		blueText := lipgloss.NewStyle().Foreground(lipgloss.Color("#3498db"))
		return blueText.Render(fmt.Sprintf(format, a...))
	}

	v := func(format string, a ...any) string {
		violetText := lipgloss.NewStyle().Foreground(lipgloss.Color("#e056fd"))
		return violetText.Render(fmt.Sprintf(format, a...))
	}

	res := fmt.Sprintf(`Select action:
	[X] Upload task
Directory: %s
Task preview: %s
Please enter task's short code (id) %s
`,
		b(m.dirPath),
		renderTaskPreview(m.task),
		v(m.shortCodeInput.View()),
	)

	if m.phase == phaseConfirmUpload {
		res = fmt.Sprintf("%sPress %s to confirm & upload, %s to cancel & exit", res, v("Y"), v("N"))
	}

	if m.phase == phaseWaitingUploadRes {
		res = fmt.Sprintf("%sWaiting for upload result", res)
	}

	res += "\n"
	return res
}
