// main.go
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file:", err)
	}

	var dir string
	flag.StringVar(&dir, "dir", "", "Directory of the task to upload")
	flag.Parse()

	if dir == "" {
		fmt.Println("Please provide a directory using the -dir flag.")
		os.Exit(1)
	}

	p := tea.NewProgram(initialModel(dir), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
