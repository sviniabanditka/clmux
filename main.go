package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sviniabanditka/clmux/internal/app"
	"github.com/sviniabanditka/clmux/internal/store"
)

func main() {
	s, err := store.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading state: %v\n", err)
		os.Exit(1)
	}

	model := app.New(s)
	p := tea.NewProgram(&model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	model.SetProgram(p)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
