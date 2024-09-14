package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/programme-lv/backend/fstask"
)

// GeneratePreview generates a preview string of the task.
func GeneratePreview(wrapper *TaskWrapper) string {
	task := wrapper.Task
	difficultyMap := map[int]string{
		1: "Very Easy",
		2: "Easy",
		3: "Medium",
		4: "Hard",
		5: "Very Hard",
	}

	var preview strings.Builder

	addLine := func(label, value string) {
		preview.WriteString(fmt.Sprintf("\t%s: %s\n", label, value))
	}

	addLine("Full Name", task.FullName)
	addLine("CPU Time Limit", fmt.Sprintf("%.3f seconds", task.CpuTimeLimInSeconds))
	addLine("Memory Limit", fmt.Sprintf("%d MB", task.MemoryLimInMegabytes))
	addLine("Difficulty", fmt.Sprintf("%d (%s)", task.DifficultyOneToFive, difficultyMap[task.DifficultyOneToFive]))
	addLine("Origin Olympiad", task.OriginOlympiad)
	addLine("Origin Notes (LV)", task.OriginNotes["lv"])
	addLine("Test Count", fmt.Sprintf("%d (Total Size: %d MB)", wrapper.GetTestTotalCount(), wrapper.GetTestTotalSize()/1024/1024))
	addLine("Example Count", fmt.Sprintf("%d", 5)) // Placeholder
	addLine("Test Group Count", fmt.Sprintf("%d (Points RLE: %v)", len(wrapper.GetTestGroupPoints()), wrapper.GetTestGroupPoints()))
	addLine("Total Score", fmt.Sprintf("%d points", wrapper.GetTotalScore()))
	addLine("Visible Input Subtasks", fmt.Sprintf("%v", wrapper.GetVisibleInputSubtasks()))
	addLine("PDF Statement Languages", fmt.Sprintf("%v", wrapper.GetPdfStatementLangs()))
	addLine("Markdown Statement Languages", fmt.Sprintf("%v", wrapper.GetMdStatementLangs()))
	addLine("Has Illustration Image", fmt.Sprintf("%t", false)) // Placeholder

	return preview.String()
}

// Model for the Bubble Tea TUI.
type model struct {
	messages   []string
	spinner    spinner.Model
	uploading  bool
	uploadDone bool
	success    bool
	err        error
}

// NewModel initializes the TUI model.
func NewModel(preview string) model {
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	return model{
		messages: []string{
			"Task Preview:",
			preview,
			"",
			"Upload this task? (y/n)",
		},
		spinner: s,
	}
}

// Init starts the initial command.
func (m model) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		if m.uploading {
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "y", "Y":
			m.uploading = true
			m.messages = append(m.messages, "\nUploading...")
			return m, tea.Batch(m.spinner.Tick, simulateUpload())
		case "n", "N":
			return m, tea.Quit
		}

	case spinner.TickMsg:
		if m.uploading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case uploadResultMsg:
		m.uploading = false
		m.uploadDone = true
		m.success = msg.success
		m.err = msg.err
		if m.success {
			m.messages = append(m.messages, "\nUpload successful!")
		} else {
			m.messages = append(m.messages, fmt.Sprintf("\nUpload failed: %v", m.err))
		}
		return m, nil
	}
	return m, nil
}

// View renders the TUI.
func (m model) View() string {
	var b strings.Builder
	for _, msg := range m.messages {
		b.WriteString(msg)
		b.WriteString("\n")
	}
	if m.uploading {
		b.WriteString(m.spinner.View())
	}
	return b.String()
}

type uploadResultMsg struct {
	success bool
	err     error
}

// Simulate the upload process.
func simulateUpload() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(3 * time.Second)
		if rand.Intn(2) == 0 {
			return uploadResultMsg{success: true}
		}
		return uploadResultMsg{success: false, err: fmt.Errorf("upload failed")}
	}
}

func main() {
	var dir string
	flag.StringVar(&dir, "dir", "", "Path to task directory")
	flag.Parse()

	if dir == "" {
		fmt.Println("Please provide a task directory with -dir flag")
		os.Exit(1)
	}

	// Load task using fstask package
	task, err := fstask.Read(dir)
	if err != nil {
		fmt.Printf("Failed to load task: %v\n", err)
		os.Exit(1)
	}

	// Create TaskWrapper
	wrapper := NewTaskWrapper(task)

	// Generate preview
	preview := GeneratePreview(wrapper)

	// Start Bubble Tea program without the alternate screen
	if _, err := tea.NewProgram(NewModel(preview), tea.WithoutRenderer()).Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
