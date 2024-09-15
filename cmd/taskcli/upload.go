package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput" // Import textinput
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/programme-lv/backend/fstask"
	"github.com/programme-lv/backend/tasksrvc"
)

// Define upload states
type uploadState int

const (
	uploadStatePreview uploadState = iota
	uploadStateEnterID
	uploadStateConfirm
	uploadStateUploading
	uploadStateDone
)

// Define the upload model
type uploadModel struct {
	state       uploadState
	uplSpinner  spinner.Model
	previewObj  TaskPreview
	taskDir     string
	success     bool
	finished    bool
	err         error
	taskIDInput textinput.Model // Add text input field
}

// Initialize a new upload model
func newUploadModel(dir string) uploadModel {
	res := uploadModel{}
	res.taskDir = dir

	dirAbs, err := filepath.Abs(dir)
	if err != nil {
		res.err = fmt.Errorf("failed to get absolute path: %w", err)
		return res
	}
	res.taskDir = dirAbs

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#3498db"))
	res.uplSpinner = s

	// Get task preview
	preview, err := getPreview(dir)
	res.previewObj = preview
	if err != nil {
		res.err = err
		return res
	}

	// Initialize text input for Task ID
	ti := textinput.New()
	ti.SetValue(filepath.Base(dir))
	ti.CharLimit = 32
	ti.Width = 30
	ti.Focus() // Set focus to the input field when entering this state
	res.taskIDInput = ti

	// Set initial state to Preview
	res.state = uploadStatePreview

	return res
}

// Initialize the model (no initial command)
func (u uploadModel) Init() tea.Cmd {
	return nil
}

// Update function to handle messages and state transitions
func (u uploadModel) Update(msg tea.Msg) (uploadModel, tea.Cmd) {
	var cmd tea.Cmd

	switch u.state {
	case uploadStatePreview:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				u.finished = true
				return u, tea.Quit
			default:
				// Transition to Enter ID state on any key press
				u.state = uploadStateEnterID
				return u, nil
			}
		}

	case uploadStateEnterID:
		// Update the text input model
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyCtrlC:
				return u, tea.Quit
			case tea.KeyEnter:
				// Ensure Task ID is not empty
				taskID := strings.TrimSpace(u.taskIDInput.Value())
				if taskID == "" {
					// Optionally, you can add feedback for empty input
					return u, nil
				}
				// Transition to Confirm state
				u.state = uploadStateConfirm
				return u, nil
			case tea.KeyEsc:
				// Exit on Esc
				u.finished = true
				return u, tea.Quit
			}
		}

	case uploadStateConfirm:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "y", "Y":
				// Start uploading
				u.state = uploadStateUploading
				return u, tea.Batch(u.uplSpinner.Tick, u.uploadTask())
			case "n", "N", "q":
				// Cancel upload
				u.finished = true
				return u, tea.Quit
			case "ctrl+c":
				// Quit on Ctrl+C
				u.finished = true
				return u, tea.Quit
			}
		}

	case uploadStateUploading:
		// Update the spinner
		u.uplSpinner, cmd = u.uplSpinner.Update(msg)
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "ctrl+c" {
				u.finished = true
				return u, tea.Quit
			}
		case uploadResultMsg:
			// Handle upload result
			u.err = msg.err
			u.success = msg.err == nil
			u.state = uploadStateDone
			return u, nil
		}
		return u, cmd

	case uploadStateDone:
		switch msg.(type) {
		case tea.KeyMsg:
			// Exit after completion
			u.finished = true
			return u, tea.Quit
		}
	}

	u.taskIDInput, cmd = u.taskIDInput.Update(msg)
	return u, cmd
}

// View function to render the UI based on the current state
func (u uploadModel) View() string {
	s := "Selected action: Upload\n\n"
	s += "Task Preview:\n"
	s += u.previewObj.View()

	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9b59b6"))
	switch u.state {
	case uploadStatePreview:
		s += "\n\nPress any key to enter Task ID..."
	case uploadStateEnterID:
		s += fmt.Sprintf("\n\nEnter Task ID: %s\n", u.taskIDInput.Value())
	case uploadStateConfirm:
		s += fmt.Sprintf("\n\nProceed with uploading task %s? (y/n)\n", valueStyle.Render(u.taskIDInput.Value()))
	case uploadStateUploading:
		s += fmt.Sprintf("\n\n%s Uploading...\n", u.uplSpinner.View())
	case uploadStateDone:
		if u.success {
			s += "\n\nUpload successful! Press any key to continue...\n"
		} else {
			s += "\n\nUpload failed! Error message: " + u.err.Error() + "\nPress any key to continue...\n"
		}
	}
	return s
}

// Define a message type for upload results
type uploadResultMsg struct {
	err error
}

// Command to handle the upload process
func (u uploadModel) uploadTask() tea.Cmd {
	return func() tea.Msg {
		taskSrvc := tasksrvc.NewTaskSrvc()
		task, err := fstask.Read(u.taskDir)
		if err != nil {
			return uploadResultMsg{err: fmt.Errorf("failed to read task: %w", err)}
		}

		// Retrieve the entered Task ID
		taskID := strings.TrimSpace(u.taskIDInput.Value())

		// Build the input for creating a task
		createTaskInput := buildCreateTaskInput(taskID, task, nil)
		err = taskSrvc.CreateTask(createTaskInput)

		if err != nil {
			return uploadResultMsg{err: fmt.Errorf("failed to create task: %w", err)}
		}

		return uploadResultMsg{err: nil}
	}
}

// Build the input for creating a public task, including the Task ID
func buildCreateTaskInput(taskId string, task *fstask.Task, illstrS3Key *string) *tasksrvc.CreatePublicTaskInput {
	visInpStasks := make([]tasksrvc.VisInpSt, len(task.GetVisibleInputSubtasks()))
	// for i, st := range task.GetVisibleInputSubtasks() {
	// 	visInpStasks[i] = tasksrvc.VisInpSt{
	// 		Subtask: st,
	// 		Inputs:  st.Inputs, // Assuming this is correctly populated
	// 	}
	// }

	return &tasksrvc.CreatePublicTaskInput{
		TaskCode:    taskId, // Assign the entered Task ID
		FullName:    task.FullName,
		MemMBytes:   task.MemoryLimInMegabytes,
		CpuSecs:     task.CpuTimeLimInSeconds,
		Difficulty:  &task.DifficultyOneToFive,
		OriginOlymp: task.OriginOlympiad,
		IllustrKey:  illstrS3Key,
		VisInpSts:   visInpStasks,
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
