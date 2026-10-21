// Package cli describes draco's commands (names, summary, subcommands and
// flags) and parses their flags. The command mode runs them; the visual
// mode's command bar suggests and reads them from the same description.
package cli

import (
	"fmt"
	"slices"
	"strings"
)

// Flag is one flag a command accepts.
type Flag struct {
	Long  string // "--issue"
	Short string // "-i"; empty when it has none
	Value bool   // takes a value
	// Optional lets a value flag go without one ("-o" alone): it only takes
	// the next argument when that is not another flag.
	Optional bool
	Desc     string // one line, for the command bar's suggestions
}

// Command describes a command; how it runs stays in the command mode.
type Command struct {
	Names       []string // the name first, then its aliases
	Summary     string
	Flags       []Flag
	Subcommands []Command
}

// Find returns the command called name, by its name or an alias.
func Find(cmds []Command, name string) *Command {
	for i := range cmds {
		if slices.Contains(cmds[i].Names, name) {
			return &cmds[i]
		}
	}
	return nil
}

// Parsed is a command line split into flags and positional arguments.
type Parsed struct {
	Values map[string]string // by long name
	Args   []string
}

// Get returns the value of a flag by its long name.
func (p *Parsed) Get(long string) string { return p.Values[long] }

// Has reports whether the flag was given.
func (p *Parsed) Has(long string) bool {
	_, ok := p.Values[long]
	return ok
}

// Parse reads args against the flags of a command. It accepts "--issue X",
// "--issue=X", "-i X" and "-i=X"; "--" ends the flags. A flag the command does
// not declare, or one missing its value, is an error, so a typo never runs a
// command with the wrong options.
func Parse(args []string, flags []Flag) (*Parsed, error) {
	byName := map[string]Flag{}
	for _, f := range flags {
		byName[f.Long] = f
		if f.Short != "" {
			byName[f.Short] = f
		}
	}
	p := &Parsed{Values: map[string]string{}}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			p.Args = append(p.Args, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			p.Args = append(p.Args, arg)
			continue
		}
		name, value, inline := strings.Cut(arg, "=")
		f, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("flag desconocido: %s", name)
		}
		switch {
		case !f.Value && inline:
			return nil, fmt.Errorf("%s no lleva valor", name)
		case !f.Value:
			value = ""
		case !inline && f.Optional:
			value = ""
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				value = args[i]
			}
		case !inline:
			if i+1 >= len(args) {
				return nil, fmt.Errorf("falta el valor de %s", name)
			}
			i++
			value = args[i]
		}
		p.Values[f.Long] = value
	}
	return p, nil
}
