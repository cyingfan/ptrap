package main

import (
	"context"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

type PipelineUpdated struct{ output string }

type rerunPipelineMsg struct{ id int }

func (m *model) runPipeline() tea.Cmd {
	// Cancel any previous execution
	if m.cancel != nil {
		m.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	// prepare immutable snapshot
	nodes := make([]cmdNode, len(m.nodes))
	copy(nodes, m.nodes)
	input := m.inputString
	return func() tea.Msg {
		defer func() { _ = ctx.Err() }()
		prevOut := input
		for _, n := range nodes {
			args := append([]string{}, n.baseArgs...)
			arg := strings.TrimSpace(n.arg)
			if arg != "" {
				args = append(args, arg)
			}
			cmd := exec.CommandContext(ctx, n.command, args...)
			cmd.Stdin = strings.NewReader(prevOut)
			out, err := cmd.CombinedOutput()
			prevOut = string(out)
			if err != nil {
				// include error text
				prevOut = prevOut + "\n" + err.Error()
				break
			}
		}
		return PipelineUpdated{output: prevOut}
	}
}

func (m *model) copyPipelineStringToClipboard() {
	var parts []string
	for _, n := range m.nodes {
		p := append([]string{n.command}, n.baseArgs...)
		cmdBase := strings.TrimSpace(strings.Join(p, " "))
		arg := strings.TrimSpace(n.arg)
		if arg != "" {
			parts = append(parts, strings.TrimSpace(cmdBase+" "+arg))
		} else {
			parts = append(parts, cmdBase)
		}
	}
	clipboard.WriteAll(strings.Join(parts, " | "))
}

func parseCmdLine(line string) (string, []string) {
	toks := strings.Fields(strings.TrimSpace(line))
	if len(toks) == 0 {
		return "", nil
	}
	return toks[0], toks[1:]
}
