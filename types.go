package main

import (
	"os"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

type cmdNode struct {
	command    string
	baseArgs   []string
	arg        string
	inputModel textinput.Model
}

type model struct {
	ready       bool
	inputString string
	viewport    viewport.Model
	// pipeline
	nodes    []cmdNode
	focusIdx int
	cancel   func()
	output   string
	// modal for adding new command
	showModal  bool
	modalInput textinput.Model
	// debounce sequence id for rerunning pipeline
	runSeq   int
	quitting bool
}

func newModel(initialValue string) (m model) {
	m.inputString = initialValue

	// initialize first command from os.Args if provided
	if len(os.Args) >= 2 {
		cmd := os.Args[1]
		var base []string
		if len(os.Args) > 2 {
			base = os.Args[2:]
		}
		ti := textinput.New()
		ti.Prompt = ""
		ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
		ti.SetValue("")
		ti.CursorEnd()
		ti.Focus()
		m.nodes = []cmdNode{{
			command:    cmd,
			baseArgs:   base,
			arg:        "",
			inputModel: ti,
		}}
		m.focusIdx = 0
	}

	// modal input
	mi := textinput.New()
	//mi.Placeholder = "new command (e.g. grep)"
	mi.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	mi.Prompt = "New command: "
	m.modalInput = mi

	return
}
