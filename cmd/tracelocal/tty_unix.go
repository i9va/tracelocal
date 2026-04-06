//go:build !windows

package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func tuiOptions() []tea.ProgramOption {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open tty: %v\n", err)
		os.Exit(1)
	}
	return []tea.ProgramOption{
		tea.WithAltScreen(),
		tea.WithInput(tty),
		tea.WithOutput(tty),
	}
}
