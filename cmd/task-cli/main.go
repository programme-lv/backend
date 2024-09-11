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
	// Parse the dir flag
	dir := flag.String("dir", "", "directory path")
	flag.Parse()

	// If no dir is provided, display an error and exit
	if *dir == "" {
		fmt.Println("Please provide a directory path using the -dir flag.")
		os.Exit(1)
	}

	// Validate the provided directory path
	if err := validateDirectory(*dir); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Get the absolute path of the directory
	absPath, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Printf("Error retrieving absolute path: %v\n", err)
		os.Exit(1)
	}

	// Display the provided directory absolute path if valid
	fmt.Printf("Task directory path provided: %s\n", absPath)

	// Start the Bubble Tea program
	p := tea.NewProgram(initialModel(absPath))
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

// validateDirectory checks if the path exists and is a directory
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

	// Handle errors
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) View() string {

	return fmt.Sprintf(
		"What’s your favorite Pokémon?\n\n%s\n\n%s",
		m.textInput.View(),
		"(esc to quit)",
	) + "\n"
}
