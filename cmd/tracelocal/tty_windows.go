//go:build windows

package main

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func tuiOptions() []tea.ProgramOption {
	return []tea.ProgramOption{
		tea.WithAltScreen(),
		tea.WithInput(os.Stdin),
		tea.WithOutput(os.Stdout),
	}
}
