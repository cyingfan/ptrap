package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
)

func printHelp() {
	prog := filepath.Base(os.Args[0])
	fmt.Println("ptrap - interactively run pipelines over piped stdin or a command's stdout")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("  %s [--run|-r \"<command>\"] [command] [args...]\n", prog)
	fmt.Println()
	fmt.Println("Description:")
	fmt.Println("  ptrap lets you build an interactive pipeline (e.g., jq | rg) and see live output.")
	fmt.Println("  Provide input via: (1) stdin (e.g., curl ... | ptrap jq), or (2) --run to execute a command and use its stdout as input.")
	fmt.Println("  Use Ctrl+C to quit.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  curl <API-endpoint> | ptrap jq")
	fmt.Println("  cat <file> | ptrap rg --color=always")
	fmt.Println("  ptrap -r \"cat file.json\" jq")
	fmt.Println()
	fmt.Println("Keyboard shortcuts:")
	fmt.Println("  Ctrl+U copy output, Ctrl+Y copy pipeline, Ctrl+N add command, Ctrl+C quit")
}

func runShellCommand(cmdStr string) (string, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Use cmd.exe for broad compatibility on Windows
		cmd = exec.Command("cmd", "/C", cmdStr)
	} else {
		cmd = exec.Command("sh", "-c", cmdStr)
	}
	out, err := cmd.Output() // stdout only
	return string(out), err
}

// resetTTY attempts to restore the terminal to a sane state on Unix-like systems.
// It runs `stty sane` against /dev/tty and also ensures the cursor is visible and
// the alternate screen is exited, writing directly to the controlling TTY.
func resetTTY() {
	if runtime.GOOS == "windows" {
		return
	}
	// Best-effort: restore terminal modes via stty sane on the controlling TTY
	if ttyIn, err := os.OpenFile("/dev/tty", os.O_RDONLY, 0); err == nil {
		cmd := exec.Command("stty", "sane")
		cmd.Stdin = ttyIn
		_ = cmd.Run()
		_ = ttyIn.Close()
	}
	// Also ensure cursor is shown and alt screen is left by writing to /dev/tty
	if ttyOut, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0); err == nil {
		_, _ = ttyOut.WriteString("\x1b[?25h\x1b[?1049l")
		_ = ttyOut.Close()
	} else {
		// As a last resort, write to stdout (may be redirected)
		fmt.Print("\x1b[?25h\x1b[?1049l")
	}
}

func main() {
	// Parse args: help and run, and collect remaining args as initial pipeline command
	runCmd := ""
	pipelineArgs := make([]string, 0, len(os.Args))

	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		switch a {
		case "-h", "--help":
			printHelp()
			os.Exit(0)
		case "-r", "--run":
			if i+1 >= len(os.Args) {
				fmt.Println("Error: --run/-r requires a command string argument")
				os.Exit(1)
			}
			i++
			runCmd = os.Args[i]
		default:
			pipelineArgs = append(pipelineArgs, a)
		}
	}

	// Rewrite os.Args so newModel can pick up the initial command without flags
	os.Args = append([]string{os.Args[0]}, pipelineArgs...)

	var inputData string
	var err error

	if runCmd != "" {
		inputData, err = runShellCommand(runCmd)
		if err != nil {
			fmt.Println("Error executing --run command:", err)
			os.Exit(1)
		}
	} else {
		// Fallback to stdin
		stat, statErr := os.Stdin.Stat()
		if statErr != nil {
			panic(statErr)
		}
		// If nothing is piped into stdin and no --run, show help
		if stat.Mode()&os.ModeNamedPipe == 0 && stat.Size() == 0 {
			printHelp()
			os.Exit(0)
		}

		reader := bufio.NewReader(os.Stdin)
		var b strings.Builder
		for {
			r, _, rErr := reader.ReadRune()
			if rErr != nil && rErr == io.EOF {
				break
			}
			_, wErr := b.WriteRune(r)
			if wErr != nil {
				fmt.Println("Error getting input:", wErr)
				os.Exit(1)
			}
		}
		inputData = b.String()
	}

	model := newModel(strings.TrimSpace(inputData))

	// Ensure we attempt to restore terminal on Unix on any exit path
	if runtime.GOOS != "windows" {
		defer resetTTY()
		// As a last-resort guard, also trap common termination signals and
		// restore the TTY before exiting in case Bubble Tea can't.
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		go func() {
			<-sigc
			resetTTY()
			os.Exit(130)
		}()
	}

	var p *tea.Program
	if runtime.GOOS != "windows" {
		// Bind both input and output to the controlling TTY so Bubble Tea can
		// properly manage and restore terminal state even when stdin/stdout are redirected.
		ttyIn, inErr := os.OpenFile("/dev/tty", os.O_RDONLY, 0)
		if inErr == nil {
			// We’ll try to open tty for output too; if it fails, fall back to default output
			ttyOut, outErr := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
			if outErr == nil {
				defer ttyIn.Close()
				defer ttyOut.Close()
				p = tea.NewProgram(&model, tea.WithAltScreen(), tea.WithInput(ttyIn), tea.WithOutput(ttyOut))
			} else {
				defer ttyIn.Close()
				p = tea.NewProgram(&model, tea.WithAltScreen(), tea.WithInput(ttyIn))
			}
		} else {
			// Fall back to default behavior
			p = tea.NewProgram(&model, tea.WithAltScreen())
		}
	} else {
		// Windows: default IO is fine
		p = tea.NewProgram(&model, tea.WithAltScreen())
	}

	if _, err := p.Run(); err != nil {
		if runtime.GOOS != "windows" {
			// Ensure cursor and screen are restored in case of abrupt termination
			resetTTY()
		}
		fmt.Println("Couldn't start program:", err)
		os.Exit(1)
	}
}
