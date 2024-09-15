package main

import tea "github.com/charmbracelet/bubbletea"

type transformState int

const (
	transformStateDetectingFormat transformState = iota
)

type transformModel struct {
}

func newTransformModel(dir string) transformModel {
	return transformModel{}
}

func (tm transformModel) Init() tea.Cmd {
	return nil
}

func (tm transformModel) View() string {
	return ""
}

func (tm transformModel) Update(msg tea.Msg) (transformModel, tea.Cmd) {
	return tm, nil
}
