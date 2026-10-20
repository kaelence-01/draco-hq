package cmd

import (
	"fmt"
	"strings"
)

// flag is one flag a command accepts.
type flag struct {
	long  string // "--issue"
	short string // "-i"; empty when it has none
	value bool   // takes a value
}

// parsed is a command line split into flags and positional arguments.
type parsed struct {
	values map[string]string
	args   []string
}

func (p *parsed) get(long string) string { return p.values[long] }

func (p *parsed) has(long string) bool {
	_, ok := p.values[long]
	return ok
}

// parseFlags reads args against the flags of a command. It accepts
// "--issue X", "--issue=X", "-i X" and "-i=X"; "--" ends the flags. A flag the
// command does not declare, or one missing its value, is an error, so a typo
// never runs a command with the wrong options.
func parseFlags(args []string, flags []flag) (*parsed, error) {
	byName := map[string]flag{}
	for _, f := range flags {
		byName[f.long] = f
		if f.short != "" {
			byName[f.short] = f
		}
	}
	p := &parsed{values: map[string]string{}}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			p.args = append(p.args, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			p.args = append(p.args, arg)
			continue
		}
		name, value, inline := strings.Cut(arg, "=")
		f, ok := byName[name]
		if !ok {
			return nil, fmt.Errorf("flag desconocido: %s", name)
		}
		switch {
		case !f.value && inline:
			return nil, fmt.Errorf("%s no lleva valor", name)
		case !f.value:
			value = ""
		case !inline:
			if i+1 >= len(args) {
				return nil, fmt.Errorf("falta el valor de %s", name)
			}
			i++
			value = args[i]
		}
		p.values[f.long] = value
	}
	return p, nil
}
