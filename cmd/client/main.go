package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ar4ie13/gophkeeper/internal/client/api"
	"github.com/ar4ie13/gophkeeper/internal/client/config"
	"github.com/ar4ie13/gophkeeper/internal/client/service"
	"github.com/ar4ie13/gophkeeper/internal/client/storage"
	"github.com/ar4ie13/gophkeeper/internal/client/tui"
	tea "github.com/charmbracelet/bubbletea"
)

// main starts run functions which contains all objects initialization
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run initializes all client components and launches the TUI application.
func run() error {
	cfg := config.NewConfig()

	apiClient, err := api.NewClient(*cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	cache := storage.NewCache()
	svc := service.NewService(apiClient, cache)

	// Launch TUI.
	model := tui.New(svc, cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err = p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	return nil
}
