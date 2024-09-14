// upload.go
package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type uploadState int

const (
	uploadStatePreview uploadState = iota
	uploadStateUploading
	uploadStateDone
)

type uploadModel struct {
	state    uploadState
	spinner  spinner.Model
	preview  TaskPreview
	dir      string
	success  bool
	finished bool
}

func newUploadModel(dir string) uploadModel {
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	return uploadModel{
		state:   uploadStatePreview,
		spinner: s,
		preview: getMockPreview(dir),
		dir:     dir,
	}
}

func (u uploadModel) Init() tea.Cmd {
	return nil
}

func (u uploadModel) Update(msg tea.Msg) (uploadModel, tea.Cmd) {
	switch u.state {
	case uploadStatePreview:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "y", "Y":
				u.state = uploadStateUploading
				return u, tea.Batch(u.spinner.Tick, u.uploadTask())
			case "n", "N", "q", "ctrl+c":
				u.finished = true
				return u, tea.Quit
			}
		}
	case uploadStateUploading:
		var cmd tea.Cmd
		u.spinner, cmd = u.spinner.Update(msg)
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.String() == "ctrl+c" {
				u.finished = true
				return u, tea.Quit
			}
		case uploadResultMsg:
			u.success = msg.success
			u.state = uploadStateDone
			return u, nil
		}
		return u, cmd
	case uploadStateDone:
		// Allow user to press any key to return to the main menu
		switch msg.(type) {
		case tea.KeyMsg:
			u.finished = true
			return u, nil
		}
	}
	return u, nil
}

func (u uploadModel) View() string {
	s := "Task Preview:\n"
	s += u.preview.View()

	switch u.state {
	case uploadStatePreview:
		s += "\nProceed with upload? (y/n)\n"
	case uploadStateUploading:
		s += fmt.Sprintf("\n\n%s Uploading...\n\n", u.spinner.View())
	case uploadStateDone:
		if u.success {
			s += "\n\nUpload successful! Press any key to continue...\n"
		} else {
			s += "\n\nUpload failed! Press any key to continue...\n"
		}
	}
	return s
}

type uploadResultMsg struct {
	success bool
}

func (u uploadModel) uploadTask() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(3 * time.Second)
		success := rand.Intn(2) == 0
		return uploadResultMsg{success: success}
	}
}
