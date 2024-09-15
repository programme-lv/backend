// tui.go
package main

import (
	tea "github.com/charmbracelet/bubbletea"
)

type state int

const (
	stateMenu state = iota
	stateUpload
	stateDelete
	stateTransform
)

type model struct {
	state          state
	uploadTaskDir  string
	uploadModel    uploadModel
	transformModel transformModel
}

func initialModel(dir string) model {
	return model{
		state:         stateMenu,
		uploadTaskDir: dir,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateMenu:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "1":
				m.state = stateUpload
				m.uploadModel = newUploadModel(m.uploadTaskDir)
				return m, m.uploadModel.Init()
			case "2":
				m.state = stateDelete
				return m, nil
			case "3":
				m.state = stateTransform
				m.transformModel = newTransformModel(m.uploadTaskDir)
				return m, m.transformModel.Init()
			}
		}
	case stateUpload:
		var cmd tea.Cmd
		m.uploadModel, cmd = m.uploadModel.Update(msg)
		return m, cmd
	case stateTransform:
		var cmd tea.Cmd
		m.transformModel, cmd = m.transformModel.Update(msg)
		return m, cmd
	case stateDelete:
		// Handle input in delete and transform states
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
			case "b":
				// Go back to main menu
				m.state = stateMenu
				return m, nil
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	switch m.state {
	case stateMenu:
		s := "Select an action:\n\n"
		s += "1. Upload / replace a task\n"
		s += "2. Delete a task with given ID\n"
		s += "3. Transform external file structure\n\n"
		s += "Press q to quit.\n"
		return s
	case stateUpload:
		return m.uploadModel.View()
	case stateTransform:
		return m.transformModel.View()
	case stateDelete:
		s := "Functionality not implemented yet.\n"
		s += "Press 'b' to return to the main menu or 'q' to quit.\n"
		return s
	default:
		return "Functionality not implemented yet.\n"
	}
}
