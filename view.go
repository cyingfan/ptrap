package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if !m.ready {
		return "Initializing...\n"
	}
	return fmt.Sprintf(
		"%s\n%s",
		m.viewport.View(),
		m.footerView(),
	)
}

func (m model) footerView() string {
	info := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
	keyHelp := infoStyle.Render("[Ctrl+C] Quit  [Enter] Copy output  [Ctrl+Y] Copy pipeline  [|] Add  [Ctrl+[ Prev]  [Ctrl+] Next  [Ctrl+D] Del")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)-lipgloss.Width(keyHelp)))

	// Build pipeline prompt
	var segments []string
	for i, n := range m.nodes {
		cmdBase := strings.TrimSpace(strings.Join(append([]string{n.command}, n.baseArgs...), " "))
		if i == m.focusIdx {
			segments = append(segments, strings.TrimSpace(cmdBase+" "+n.inputModel.View()))
		} else {
			if strings.TrimSpace(n.arg) == "" {
				segments = append(segments, cmdBase)
			} else {
				segments = append(segments, strings.TrimSpace(cmdBase+" "+strings.TrimSpace(n.arg)))
			}
		}
	}
	prompt := "Command: "
	if len(segments) == 0 {
		prompt += "(press | to add a command)"
	} else {
		prompt += strings.Join(segments, " | ")
	}

	footer := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Center, keyHelp, line, info),
		lipgloss.JoinHorizontal(lipgloss.Center, prompt),
	)

	// If modal is shown, overlay a simple line for modal input
	if m.showModal {
		modal := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Render(m.modalInput.View())
		footer = lipgloss.JoinVertical(lipgloss.Left, footer, modal)
	}
	return footer
}
