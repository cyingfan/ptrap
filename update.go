package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type heartbeatMsg struct{}

func heartbeatCmd() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(time.Time) tea.Msg { return heartbeatMsg{} })
}

func (m model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, textinput.Blink, heartbeatCmd())
	if len(m.nodes) > 0 {
		cmds = append(cmds, m.runPipeline())
	}
	return tea.Batch(cmds...)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
		cmd  tea.Cmd
	)

	switch msg := msg.(type) {
	case heartbeatMsg:
		cmds = append(cmds, heartbeatCmd())
	case tea.KeyMsg:
		key := msg
		// If modal is open, route keys to modal
		if m.showModal {
			if key.Type == tea.KeyEnter {
				entry := strings.TrimSpace(m.modalInput.Value())
				if entry == "" {
					// If no nodes, keep prompting; otherwise just close.
					if len(m.nodes) == 0 {
						// keep modal open
					} else {
						m.showModal = false
					}
				} else {
					cmdName, base := parseCmdLine(entry)
					if cmdName != "" {
						ni := textinput.New()
						ni.Prompt = ""
						ni.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
						ni.SetValue("")
						ni.CursorEnd()
						ni.Focus()
						m.nodes = append(m.nodes, cmdNode{command: cmdName, baseArgs: base, arg: "", inputModel: ni})
						m.focusIdx = len(m.nodes) - 1
						m.showModal = false
						m.modalInput.SetValue("")
						cmds = append(cmds, m.runPipeline())
					}
				}
				// still update modal for cursor blink etc
				m.modalInput, cmd = m.modalInput.Update(msg)
				cmds = append(cmds, cmd)
				break
			}
			if key.Type == tea.KeyEsc {
				// close modal
				m.showModal = false
				break
			}
			m.modalInput, cmd = m.modalInput.Update(msg)
			cmds = append(cmds, cmd)
			break
		}

		switch key.Type {
		case tea.KeyCtrlC:
			if m.cancel != nil {
				m.cancel()
			}
			return m, func() tea.Msg { fmt.Print("\r\n"); os.Exit(0); return nil }
		case tea.KeyCtrlY:
			m.copyPipelineStringToClipboard()
		case tea.KeyEnter:
			clipboard.WriteAll(m.output)
		case tea.KeyCtrlD:
			if len(m.nodes) > 0 {
				// delete current
				idx := m.focusIdx
				m.nodes = append(m.nodes[:idx], m.nodes[idx+1:]...)
				if len(m.nodes) == 0 {
					// prompt for new command
					m.showModal = true
					m.modalInput.SetValue("")
					m.modalInput.CursorEnd()
					m.modalInput.Focus()
					m.viewport.SetContent("")
					m.output = ""
				} else {
					if idx >= len(m.nodes) {
						m.focusIdx = len(m.nodes) - 1
					}
					cmds = append(cmds, m.runPipeline())
				}
			}
		default:
			// open modal on '|' only
			if key.String() == "|" {
				m.showModal = true
				m.modalInput.SetValue("")
				m.modalInput.CursorEnd()
				m.modalInput.Focus()
				break
			}
			// navigation
			if key.Type == tea.KeyCtrlOpenBracket {
				if m.focusIdx > 0 {
					// blur current
					m.nodes[m.focusIdx].inputModel.Blur()
					m.focusIdx--
					m.nodes[m.focusIdx].inputModel.Focus()
				}
				break
			}
			if key.Type == tea.KeyCtrlCloseBracket {
				if m.focusIdx < len(m.nodes)-1 {
					m.nodes[m.focusIdx].inputModel.Blur()
					m.focusIdx++
					m.nodes[m.focusIdx].inputModel.Focus()
				}
				break
			}
		}
	case tea.WindowSizeMsg:
		ws := msg
		footerHeight := lipgloss.Height(m.footerView())
		if !m.ready {
			m.viewport = viewport.New(ws.Width, ws.Height-footerHeight)
			m.ready = true
		} else {
			m.viewport.Width = ws.Width
			m.viewport.Height = ws.Height - footerHeight
		}
	case PipelineUpdated:
		pu := msg
		m.output = pu.output
		m.viewport.SetContent(pu.output)
	case rerunPipelineMsg:
		// only run if this is the latest scheduled seq
		if msg.id == m.runSeq {
			cmds = append(cmds, m.runPipeline())
		}
	}

	// update the focused input if not in modal
	if !m.showModal && len(m.nodes) > 0 && m.focusIdx >= 0 && m.focusIdx < len(m.nodes) {
		n := m.nodes[m.focusIdx]
		n.inputModel, cmd = n.inputModel.Update(msg)
		m.nodes[m.focusIdx] = n
		cmds = append(cmds, cmd)
		// sync arg from input
		val := n.inputModel.Value()
		if val != m.nodes[m.focusIdx].arg {
			m.nodes[m.focusIdx].arg = val
			// debounce rerun: schedule after short delay
			m.runSeq++
			seq := m.runSeq
			cmds = append(cmds, tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg { return rerunPipelineMsg{id: seq} }))
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}
