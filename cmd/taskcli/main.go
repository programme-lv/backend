package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/joho/godotenv"
	"github.com/programme-lv/backend/fstask"
	"github.com/programme-lv/backend/tasksrvc"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	// Parse command-line flags
	dir := flag.String("dir", "", "directory path")
	flag.Parse()

	if *dir == "" {
		fmt.Println("Please provide a directory path using the -dir flag.")
		os.Exit(1)
	}

	// Validate the provided directory
	if err := validateDirectory(*dir); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	absPath, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Printf("Error retrieving absolute path: %v\n", err)
		os.Exit(1)
	}

	// Read the task from the specified directory
	task, err := fstask.Read(absPath)
	if err != nil {
		log.Fatal("Failed to read task:", err)
	}

	// Initialize and run the Bubble Tea program
	p := tea.NewProgram(initialModel(absPath, task))
	if _, err := p.Run(); err != nil {
		log.Fatal("Failed to run program:", err)
	}
}

// validateDirectory checks if the provided path exists and is a directory.
func validateDirectory(dir string) error {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist")
	}
	if err != nil {
		return fmt.Errorf("unable to access directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory")
	}
	return nil
}

// Phase represents the current phase of the application.
type Phase int

const (
	PhaseEnterTaskID Phase = iota
	PhaseConfirmUpload
	PhaseUploadingIllustrationImg
	PhaseCreatingTask
	PhaseFinished
)

// Model defines the state of the application.
type Model struct {
	Phase   Phase
	Err     error
	DirPath string

	Task    *fstask.Task
	Wrapper *TaskWrapper

	TaskService *tasksrvc.TaskService

	TaskShortCodeIDInput   textinput.Model
	IllstrImgUploadSpinner spinner.Model

	IllstrS3Key *string
}

// initialModel initializes the application's model.
func initialModel(dirPath string, task *fstask.Task) Model {
	taskIDInput := textinput.New()
	taskIDInput.SetValue(filepath.Base(dirPath))
	taskIDInput.Focus()
	taskIDInput.CharLimit = 156
	taskIDInput.Width = 20

	illstrSpin := spinner.New()
	illstrSpin.Spinner = spinner.Dot
	illstrSpin.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		Phase:                  PhaseEnterTaskID,
		DirPath:                dirPath,
		Task:                   task,
		Wrapper:                NewTaskWrapper(task),
		TaskService:            tasksrvc.NewTaskSrvc(),
		TaskShortCodeIDInput:   taskIDInput,
		IllstrImgUploadSpinner: illstrSpin,
		IllstrS3Key:            nil,
	}
}

// Init initializes the Bubble Tea program.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// UploadIllustrationImgResult represents the result of uploading an illustration image.
type UploadIllustrationImgResult struct {
	Err error
}

// CreateTaskResult represents the result of creating a task.
type CreateTaskResult struct {
	Err error
}

// Define styles globally
var (
	labelStyle = lipgloss.NewStyle().
		// Foreground(lipgloss.Color("#7f8c8d")).
		Bold(false)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#2ecc71"))
		// Bold(true)

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9b59b6")).Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#e74c3c")).
			Bold(true)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3498db")).
			Bold(true)

	greenCheck = lipgloss.NewStyle().
			SetString("✓").
			Foreground(lipgloss.Color("#2ecc71")).
			Bold(true)
)

// Update handles incoming messages and updates the model accordingly.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case UploadIllustrationImgResult:
		log.Print("Received UploadIllustrationImgResult")
		if msg.Err != nil {
			m.Err = msg.Err
			log.Println("Error uploading illustration image:", msg.Err)
			return m, tea.Quit
		}
		m.Phase = PhaseCreatingTask
		return m, func() tea.Msg {
			createTaskInput := buildCreateTaskInput(m)
			err := m.TaskService.CreateTask(createTaskInput)
			return CreateTaskResult{Err: err}
		}

	case CreateTaskResult:
		if msg.Err != nil {
			m.Err = msg.Err
			log.Println("Error creating task:", msg.Err)
			return m, tea.Quit
		}
		m.Phase = PhaseFinished
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.Phase == PhaseEnterTaskID {
				m.TaskShortCodeIDInput.Blur()
				m.Phase = PhaseConfirmUpload
				return m, nil
			}
		case tea.KeyRunes:
			switch msg.Runes[0] {
			case 'y', 'Y':
				if m.Phase == PhaseConfirmUpload {
					m.Phase = PhaseUploadingIllustrationImg
					return m, tea.Batch(
						m.IllstrImgUploadSpinner.Tick,
						uploadIllustrationImageCmd(m),
					)
				}
			case 'n', 'N':
				return m, tea.Quit
			}
		}

	case spinner.TickMsg:
		m.IllstrImgUploadSpinner, cmd = m.IllstrImgUploadSpinner.Update(msg)
		return m, cmd
	}

	// Update the text input field
	m.TaskShortCodeIDInput, cmd = m.TaskShortCodeIDInput.Update(msg)
	return m, cmd
}

// buildCreateTaskInput constructs the input for creating a task.
func buildCreateTaskInput(m Model) *tasksrvc.CreatePublicTaskInput {
	return &tasksrvc.CreatePublicTaskInput{
		TaskCode:    m.TaskShortCodeIDInput.Value(),
		FullName:    m.Task.FullName,
		MemMBytes:   m.Task.MemoryLimInMegabytes,
		CpuSecs:     m.Task.CpuTimeLimInSeconds,
		Difficulty:  &m.Task.DifficultyOneToFive,
		OriginOlymp: m.Task.OriginOlympiad,
		IllustrKey:  m.IllstrS3Key,
		VisInpSts:   m.Wrapper.GetVisibleInputSubtasks(),
		// Initialize other fields as empty slices
		TestGroups:  []tasksrvc.TestGroup{},
		TestChsums:  []tasksrvc.TestChecksum{},
		PdfSttments: []tasksrvc.PdfStatement{},
		MdSttments:  []tasksrvc.MarkdownStatement{},
		ImgUuidMap:  []tasksrvc.ImageUUIDMap{},
		Examples:    []tasksrvc.Example{},
		OriginNotes: []tasksrvc.OriginNote{},
	}
}

// uploadIllustrationImageCmd creates a command to upload the illustration image.
func uploadIllustrationImageCmd(m Model) tea.Cmd {
	return func() tea.Msg {
		illstrImg := m.Task.GetTaskIllustrationImage()
		if illstrImg != nil {
			s3Key, err := UploadIllustrationImage(illstrImg, m.TaskService)
			if err != nil {
				return UploadIllustrationImgResult{Err: err}
			}
			m.IllstrS3Key = &s3Key
		}
		return UploadIllustrationImgResult{Err: nil}
	}
}

// View renders the UI based on the current model state.
func (m Model) View() string {
	var res strings.Builder

	res.WriteString(fmt.Sprintf("%s [X] Upload task\n", labelStyle.Render("Select action:")))

	// Directory and Task Preview
	res.WriteString(fmt.Sprintf("%s: %s\n", labelStyle.Render("Directory"), valueStyle.Render(m.DirPath)))
	res.WriteString("Task preview:\n")
	res.WriteString(renderTaskPreview(m.Wrapper))
	res.WriteString(fmt.Sprintf("\n%s: %s\n\n", labelStyle.Render("Task Short Code (ID)"), inputStyle.Render(m.TaskShortCodeIDInput.View())))

	// Phase-specific Views
	switch m.Phase {
	case PhaseConfirmUpload:
		res.WriteString("Press 'Y' to confirm & upload, 'N' to cancel & exit\n")
	case PhaseUploadingIllustrationImg:
		res.WriteString(fmt.Sprintf("Uploading illustration image to S3: %s\n", m.IllstrImgUploadSpinner.View()))
	case PhaseCreatingTask:
		// Replace spinner with checkmark for upload
		res.WriteString(fmt.Sprintf("%s Illustration image uploaded successfully.\n", greenCheck.Render()))
		res.WriteString(fmt.Sprintf("Creating task in DynamoDB: %s\n", m.IllstrImgUploadSpinner.View()))
	case PhaseFinished:
		// Replace spinner with checkmark for task creation
		res.WriteString(fmt.Sprintf("%s Task rows inserted into DynamoDB.\n", greenCheck.Render()))
		res.WriteString(fmt.Sprintf("\n%s Task \"%s\" created successfully!\n", greenCheck.Render(), m.TaskShortCodeIDInput.Value()))
	}

	// Error Message
	if m.Err != nil {
		res.WriteString(fmt.Sprintf("\n%s %s\n", errorStyle.Render("Error:"), m.Err.Error()))
	}

	return res.String()
}

// renderTaskPreview generates a formatted string that previews the task details.
func renderTaskPreview(wrapper *TaskWrapper) string {
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
		preview.WriteString(fmt.Sprintf("  %s: %s\n", labelStyle.Render(label), valueStyle.Render(value)))
	}

	addLine("Full Name", task.FullName)
	addLine("CPU Time Limit", fmt.Sprintf("%.3f seconds", task.CpuTimeLimInSeconds))
	addLine("Memory Limit", fmt.Sprintf("%d MB", task.MemoryLimInMegabytes))
	addLine("Difficulty", fmt.Sprintf("%d (%s)", task.DifficultyOneToFive, difficultyMap[task.DifficultyOneToFive]))
	addLine("Origin Olympiad", fmt.Sprintf("%v", task.OriginOlympiad))
	addLine("Origin Notes (LV)", fmt.Sprintf("%v", task.OriginNotes["lv"]))
	addLine("Test Count", fmt.Sprintf("%d (Total Size: %d MB)", wrapper.GetTestTotalCount(), wrapper.GetTestTotalSize()/1024/1024))
	addLine("Example Count", fmt.Sprintf("%d", len(task.GetExamples())))
	addLine("Test Group Count", fmt.Sprintf("%d (Points RLE: %v)", len(task.GetTestGroupIDs()), wrapper.GetTestGroupPointsRLE()))
	addLine("Total Score", fmt.Sprintf("%d points", wrapper.GetTotalScore()))
	addLine("Visible Input Subtasks", fmt.Sprintf("%v", task.GetVisibleInputSubtasks()))
	addLine("PDF Statement Languages", fmt.Sprintf("%v", wrapper.GetPdfStatementLangs()))
	addLine("Markdown Statement Languages", fmt.Sprintf("%v", wrapper.GetMdStatementLangs()))
	addLine("Has Illustration Image", fmt.Sprintf("%t", task.GetTaskIllustrationImage() != nil))
	if img := task.GetTaskIllustrationImage(); img != nil {
		addLine("Illustration Image Path", fmt.Sprintf("assets/%s", img.RelativePath))
	}

	return preview.String()
}
