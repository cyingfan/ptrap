package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Right = "├"
		return lipgloss.NewStyle().BorderStyle(b).Padding(0, 1)
	}()

	infoStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.Left = "┤"
		return titleStyle.BorderStyle(b)
	}()
)

type model struct {
	ready       bool
	inputString string
	runner      commandrunner
	viewport    viewport.Model
	userInput   textinput.Model
}

func newModel(initialValue string) (m model) {
	m.inputString = initialValue

	m.runner = NewRunner(initialValue)

	i := textinput.New()
	i.Prompt = ""
	i.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	i.SetValue("")
	i.CursorEnd()
	i.Focus()
	m.userInput = i

	return
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmds []tea.Cmd
		cmd  tea.Cmd
	)

	switch msg.(type) {
	case tea.KeyMsg:
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.Type {
			case tea.KeyCtrlC, tea.KeyEscape:
				return m, tea.Quit

			case tea.KeyEnter:
				clipboard.WriteAll(m.runner.output)
			}

		}

	case tea.WindowSizeMsg:
		msg := msg.(tea.WindowSizeMsg)
		footerHeight := lipgloss.Height(m.footerView())
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-footerHeight)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - footerHeight
		}

	case RunnerUpdated:
		msg := msg.(RunnerUpdated)
		m.runner = msg.runner
		m.viewport.SetContent(msg.runner.output)
	}

	m.userInput, cmd = m.userInput.Update(msg)
	cmds = append(cmds, cmd)

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	cmd = m.runner.Update(m.userInput.Value())
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

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
	keyHelp := infoStyle.Render("[Esc] Quit  [Ctrl+C] Quit  [Enter] Copy to clipboard")
	line := strings.Repeat("─", max(0, m.viewport.Width-lipgloss.Width(info)-lipgloss.Width(keyHelp)))

	prompt := fmt.Sprintf("Command: %s %s ", m.runner.command, m.userInput.View())

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Center, keyHelp, line, info),
		lipgloss.JoinHorizontal(lipgloss.Center, prompt),
	)
}

type RunnerUpdated struct {
	runner commandrunner
}

type commandrunner struct {
	input    string
	command  string
	baseArgs []string
	output   string
	arg      string
}

func NewRunner(input string) (c commandrunner) {
	if len(os.Args) < 2 {
		fmt.Println("Usage: trap <command> [args...]")
		os.Exit(1)
	}
	c.input = input
	c.command = os.Args[1]
	if len(os.Args) < 3 {
		c.baseArgs = []string{}
	} else {
		c.baseArgs = os.Args[2:]
	}

	c.arg = " "
	return
}

func (c commandrunner) Update(arg string) tea.Cmd {
	if arg != c.arg {
		return func() tea.Msg {
			c.arg = arg
			c.output = c.exec()
			return RunnerUpdated{runner: c}
		}
	}
	return func() tea.Msg {
		return nil
	}
}

func (c commandrunner) exec() string {
	cmd := exec.Command(c.command, append(c.baseArgs, c.arg)...)
	cmd.Stdin = strings.NewReader(c.input)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out) + "\n" + err.Error()
	}
	return string(out)
}

func main() {
	stat, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	if stat.Mode()&os.ModeNamedPipe == 0 && stat.Size() == 0 {
		fmt.Println("Try piping in some text.")
		os.Exit(1)
	}

	reader := bufio.NewReader(os.Stdin)
	var b strings.Builder

	for {
		r, _, err := reader.ReadRune()
		if err != nil && err == io.EOF {
			break
		}
		_, err = b.WriteRune(r)
		if err != nil {
			fmt.Println("Error getting input:", err)
			os.Exit(1)
		}
	}

	model := newModel(strings.TrimSpace(b.String()))

	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Println("Couldn't start program:", err)
		os.Exit(1)
	}
}
