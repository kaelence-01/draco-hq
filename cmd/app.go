// Package cmd is draco's command mode: a router of commands over the flows,
// with text prompts where the flow leaves a decision to the user.
package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strings"

	"golang.org/x/term"
)

// Exit codes.
const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

// App runs commands reading answers from In and writing to Out and Err.
type App struct {
	In       *bufio.Reader
	Out, Err io.Writer
	color    bool
	// secretFd is the terminal behind In, read without echo for tokens; -1
	// when In is not a terminal.
	secretFd int
	// tty means a person at a terminal (input and output): only then draco
	// opens the visual mode, offers to install itself or prints the
	// new-version notice.
	tty bool
	// rawIn is In without the buffer, for the visual mode's raw terminal.
	rawIn io.Reader
}

// New returns an App; color is on only when out is a terminal and NO_COLOR is
// unset, so pipes, files and CI logs never get escape codes.
func New(in io.Reader, out, err io.Writer) *App {
	a := &App{In: bufio.NewReader(in), Out: out, Err: err, color: isTerminal(out) && os.Getenv("NO_COLOR") == "", secretFd: -1, rawIn: in}
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		a.secretFd = int(f.Fd())
	}
	a.tty = a.secretFd >= 0 && isTerminal(out)
	return a
}

// context is canceled by Ctrl+C, so a slow request stops instead of hanging.
func (a *App) context() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt)
}

// ask prints prompt and returns the trimmed answer; at the end of the input
// the answer is empty, which every prompt treats as its default.
func (a *App) ask(prompt string) string {
	a.printf("%s", prompt)
	line, _ := a.In.ReadString('\n')
	return strings.TrimSpace(line)
}

// confirm asks a yes/no question; def is the answer for Enter.
func (a *App) confirm(question string, def bool) bool {
	hint := "[S/n]"
	if !def {
		hint = "[s/N]"
	}
	switch strings.ToLower(a.ask(question + " " + hint + ": ")) {
	case "":
		return def
	case "s", "si", "sí", "y", "yes":
		return true
	}
	return false
}

// errEndOfInput stops a wizard whose input ended before a required answer.
var errEndOfInput = errors.New("la entrada terminó antes de completar las preguntas")

func (a *App) askLine(prompt string) (string, error) {
	a.printf("%s", prompt)
	line, err := a.In.ReadString('\n')
	if err != nil && line == "" {
		return "", errEndOfInput
	}
	return strings.TrimSpace(line), nil
}

// askValue asks for a value: Enter takes def; a required value with no def is
// asked again.
func (a *App) askValue(label, def string, required bool) (string, error) {
	prompt := "  " + label + ": "
	switch {
	case def != "":
		prompt = fmt.Sprintf("  %s [%s]: ", label, def)
	case !required:
		prompt = "  " + label + " (Enter para omitir): "
	}
	for {
		v, err := a.askLine(prompt)
		if err != nil {
			return "", err
		}
		if v == "" {
			v = def
		}
		if v != "" || !required {
			return v, nil
		}
		a.warn("este dato es obligatorio")
	}
}

// askSecret reads without echo on a terminal, and a plain line elsewhere.
func (a *App) askSecret(prompt string) (string, error) {
	if a.secretFd < 0 {
		return a.askLine(prompt)
	}
	a.printf("%s", prompt)
	b, err := term.ReadPassword(a.secretFd)
	a.printf("\n")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// command is one entry of the router.
type command struct {
	run     func(a *App, args []string) error
	help    func(a *App)
	summary string
}

// commands maps every name and alias to its command. TestEveryCommandHasHelp
// keeps each one with its help.
var commands = map[string]*command{}

func register(c *command, names ...string) {
	for _, n := range names {
		commands[n] = c
	}
}

// errUsage marks errors in how a command was called; they exit with exitUsage.
var errUsage = errors.New("uso incorrecto")

func usageError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errUsage, fmt.Sprintf(format, args...))
}

// Run executes the command line and returns the exit code.
func (a *App) Run(args []string) int {
	if len(args) == 0 {
		if a.tty {
			if err := runTUI(a); err != nil {
				a.errorf("%v", err)
				return exitError
			}
			return exitOK
		}
		a.usage()
		return exitOK
	}
	name := args[0]
	if isHelpArg(name) {
		if len(args) > 1 {
			if c, ok := commands[args[1]]; ok {
				c.help(a)
				return exitOK
			}
			a.errorf("no existe el comando %q", args[1])
			return exitUsage
		}
		a.usage()
		return exitOK
	}
	if name == "--version" || name == "-v" {
		name = "version"
	}
	c, ok := commands[name]
	if !ok {
		a.errorf("comando desconocido: %s (ver draco help)", name)
		return exitUsage
	}
	if wantsHelp(args[1:]) {
		c.help(a)
		return exitOK
	}
	if err := c.run(a, args[1:]); err != nil {
		if errors.Is(err, errUsage) {
			a.errorf("%s (ver draco help %s)", strings.TrimPrefix(err.Error(), errUsage.Error()+": "), name)
			return exitUsage
		}
		a.errorf("%v", err)
		return exitError
	}
	return exitOK
}

func isHelpArg(arg string) bool { return arg == "help" || arg == "--help" || arg == "-h" }

// wantsHelp answers -h before the command runs: a command must never do its
// job when the user only asked how to use it.
func wantsHelp(args []string) bool {
	for _, a := range args {
		if a == "--" {
			return false
		}
		if a == "-h" || a == "--help" {
			return true
		}
	}
	return false
}

// ANSI styles.
const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
)

func (a *App) style(code, text string) string {
	if !a.color {
		return text
	}
	return code + text + reset
}

func (a *App) printf(format string, args ...any) { fmt.Fprintf(a.Out, format, args...) }

func (a *App) errorf(format string, args ...any) {
	fmt.Fprintf(a.Err, "%s %s\n", a.style(red+bold, "Error:"), fmt.Sprintf(format, args...))
}

func (a *App) ok(msg string)   { a.printf("  %s %s\n", a.style(green, "✓"), msg) }
func (a *App) warn(msg string) { a.printf("  %s %s\n", a.style(yellow, "⚠"), msg) }
func (a *App) fail(msg string) { a.printf("  %s %s\n", a.style(red, "✗"), msg) }

func (a *App) step(n, total int, msg string) {
	a.printf("%s %s\n", a.style(cyan+bold, fmt.Sprintf("[%d/%d]", n, total)), msg)
}

func (a *App) title(name, summary string) {
	a.printf("%s — %s\n\n", a.style(cyan+bold, name), summary)
}

func (a *App) section(name string) { a.printf("%s\n", a.style(bold, name)) }

// configPathHint shows where config.yaml lives, as the user's system writes paths.
func configPathHint() string {
	if runtime.GOOS == "windows" {
		return `%USERPROFILE%\.config\draco\config.yaml`
	}
	return "~/.config/draco/config.yaml"
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
