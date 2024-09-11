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

	p := tea.NewProgram(initialModel(absPath))
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
	textInput textinput.Model
	err       error
	dirPath   string
	fullName  string
}

func initialModel(dirPath string) model {
	ti := textinput.New()
	ti.Placeholder = "Pikachu"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	return model{
		textInput: ti,
		err:       nil,
		dirPath:   dirPath,
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
		case tea.KeyEnter, tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		}

	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {
	greenDirPath := lipgloss.NewStyle().Foreground(lipgloss.Color("#0000ff")).Render(m.dirPath)
	return fmt.Sprintf(`
	Directory: %s
	Full name: %s

	`,
		greenDirPath, m.fullName)
}
