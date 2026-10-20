// Package cmd is draco's command mode: a router of commands over the flows,
// with text prompts where the flow leaves a decision to the user.
package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

// Exit codes.
const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

// App runs commands writing to Out and Err.
type App struct {
	Out, Err io.Writer
	color    bool
}

// New returns an App; color is on only when out is a terminal and NO_COLOR is
// unset, so pipes, files and CI logs never get escape codes.
func New(out, err io.Writer) *App {
	return &App{Out: out, Err: err, color: isTerminal(out) && os.Getenv("NO_COLOR") == ""}
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
		// The visual mode (phase 4) will open here on an interactive terminal.
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
