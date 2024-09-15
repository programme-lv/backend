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
	state       state
	dir         string
	uploadModel uploadModel
}

func initialModel(dir string) model {
	return model{
		state: stateMenu,
		dir:   dir,
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
				m.uploadModel = newUploadModel(m.dir)
				return m, m.uploadModel.Init()
			case "2":
				m.state = stateDelete
				return m, nil
			case "3":
				m.state = stateTransform
				return m, nil
			}
		}
	case stateUpload:
		var cmd tea.Cmd
		m.uploadModel, cmd = m.uploadModel.Update(msg)
		return m, cmd
	case stateDelete, stateTransform:
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
		s += "1. Upload/Replace a task\n"
		s += "2. Delete a task with given ID\n"
		s += "3. Transform from one file structure to another\n\n"
		s += "Press q to quit.\n"
		return s
	case stateUpload:
		return m.uploadModel.View()
	case stateDelete, stateTransform:
		s := "Functionality not implemented yet.\n"
		s += "Press 'b' to return to the main menu or 'q' to quit.\n"
		return s
	default:
		return "Functionality not implemented yet.\n"
	}
}
