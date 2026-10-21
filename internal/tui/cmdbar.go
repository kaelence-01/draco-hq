package tui

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/cli"
	"github.com/kaelence-01/draco-hq/internal/config"
)

var (
	barKey      = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "comando (/help: ayuda)"))
	completeKey = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "completar"))
	pickSugKey  = key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "sugerencias"))
	runKey      = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "ejecutar"))
	closeBarKey = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cerrar"))
)

// helpCommand is the bar's own help: the command mode answers help before
// its router, so it is not among the commands.
var helpCommand = cli.Command{Names: []string{"help"}, Summary: "Ayuda de draco o de un comando (/help g)"}

// barCommands are the commands the bar knows: help and those of the command
// mode.
func barCommands(d Deps) []cli.Command {
	return append([]cli.Command{helpCommand}, d.Commands...)
}

// suggestion completes the word being typed in the command bar.
type suggestion struct {
	text  string // replaces the word
	label string
	desc  string
}

// barWords splits a bar line into the finished words and the one being
// typed. The leading "/" and "draco" are optional.
func barWords(line string) (done []string, typing string) {
	line = strings.TrimPrefix(strings.TrimLeft(line, " "), "/")
	words := strings.Fields(line)
	if len(words) > 0 && !strings.HasSuffix(line, " ") {
		done, typing = words[:len(words)-1], words[len(words)-1]
	} else {
		done = words
	}
	if len(done) > 0 && done[0] == "draco" {
		done = done[1:]
	}
	return done, typing
}

// suggest lists what can follow the line: commands, subcommands, unused
// flags or the values the config knows for the flag being filled.
func suggest(cmds []cli.Command, cfg *config.Config, line string) []suggestion {
	done, typing := barWords(line)
	if len(done) == 0 {
		return nameSuggestions(cmds, typing)
	}
	c := cli.Find(cmds, done[0])
	if c == nil {
		return nil
	}
	rest, flags := done[1:], c.Flags
	switch {
	case c.Names[0] == helpCommand.Names[0]:
		if len(rest) > 0 {
			return nil
		}
		var others []cli.Command
		for _, o := range cmds {
			if o.Names[0] != helpCommand.Names[0] {
				others = append(others, o)
			}
		}
		return nameSuggestions(others, typing)
	case len(c.Subcommands) > 0 && len(rest) == 0:
		return nameSuggestions(c.Subcommands, typing)
	case len(c.Subcommands) > 0:
		sub := cli.Find(c.Subcommands, rest[0])
		if sub == nil {
			return nil
		}
		rest, flags = rest[1:], sub.Flags
	}
	if len(rest) > 0 && !strings.HasPrefix(typing, "-") {
		if f := flagNamed(flags, rest[len(rest)-1]); f != nil && f.Value && !f.Optional {
			return valueSuggestions(f, cfg, typing)
		}
	}
	if typing != "" && !strings.HasPrefix(typing, "-") {
		return nil
	}
	return flagSuggestions(flags, rest, typing)
}

func nameSuggestions(cmds []cli.Command, typing string) []suggestion {
	var out []suggestion
	for _, c := range cmds {
		text := ""
		for _, n := range c.Names {
			if strings.HasPrefix(n, typing) && (text == "" || len(n) < len(text)) {
				text = n
			}
		}
		if text != "" {
			out = append(out, suggestion{text: text, label: strings.Join(c.Names, ", "), desc: c.Summary})
		}
	}
	return out
}

// flagNamed is the flag word names, without an inline value ("-i" or
// "--issue", not "-i=OPS-7").
func flagNamed(flags []cli.Flag, word string) *cli.Flag {
	for i, f := range flags {
		if word == f.Long || f.Short != "" && word == f.Short {
			return &flags[i]
		}
	}
	return nil
}

func flagSuggestions(flags []cli.Flag, words []string, typing string) []suggestion {
	used := map[string]bool{}
	for _, w := range words {
		name, _, _ := strings.Cut(w, "=")
		if f := flagNamed(flags, name); f != nil {
			used[f.Long] = true
		}
	}
	var out []suggestion
	for _, f := range flags {
		if used[f.Long] || !strings.HasPrefix(f.Long, typing) && (f.Short == "" || !strings.HasPrefix(f.Short, typing)) {
			continue
		}
		text, label := f.Long, f.Long
		if f.Short != "" {
			label = f.Short + ", " + f.Long
			if !strings.HasPrefix(typing, "--") {
				text = f.Short
			}
		}
		out = append(out, suggestion{text: text, label: label, desc: f.Desc})
	}
	return out
}

func valueSuggestions(f *cli.Flag, cfg *config.Config, typing string) []suggestion {
	var values []string
	var desc string
	switch f.Long {
	case "--project":
		if cfg != nil {
			values, desc = cfg.ProjectNames(), "proyecto"
		}
	case "--profile":
		if cfg != nil {
			values, desc = cfg.ProfileNames(), "perfil"
		}
	case "--repo":
		values, desc = []string{"backend", "frontend"}, "repo"
	}
	var out []suggestion
	for _, v := range values {
		if strings.HasPrefix(v, typing) {
			out = append(out, suggestion{text: v, label: v, desc: desc})
		}
	}
	return out
}

// complete replaces the word being typed with s.
func complete(line string, s suggestion) string {
	_, typing := barWords(line)
	return strings.TrimSuffix(line, typing) + s.text + " "
}

// barLine is a line of the bar read as the command mode would read it.
type barLine struct {
	cmd, sub *cli.Command
	flags    *cli.Parsed
}

// readLine reads the line with the same flags as the command: an unknown
// flag, or one without its value, is an error, as it exits with 2 there. An
// empty line is a nil command.
func readLine(cmds []cli.Command, line string) (barLine, error) {
	done, typing := barWords(line)
	words := done
	if typing != "" {
		words = append(slices.Clip(done), typing)
	}
	if len(words) == 0 {
		return barLine{}, nil
	}
	l := barLine{cmd: cli.Find(cmds, words[0])}
	if l.cmd == nil {
		return l, fmt.Errorf("comando desconocido: %s (/help lista los comandos)", words[0])
	}
	rest, flags := words[1:], l.cmd.Flags
	if len(l.cmd.Subcommands) > 0 && len(rest) > 0 {
		if l.sub = cli.Find(l.cmd.Subcommands, rest[0]); l.sub == nil {
			return l, fmt.Errorf("subcomando desconocido: %s %s", words[0], rest[0])
		}
		rest, flags = rest[1:], l.sub.Flags
	}
	var err error
	l.flags, err = cli.Parse(rest, flags)
	return l, err
}

// cmdBar is the command line of the home screen.
type cmdBar struct {
	st     *state
	text   string
	cursor int
	note   string
	failed bool
}

func (b *cmdBar) keys() []key.Binding {
	return []key.Binding{completeKey, pickSugKey, runKey, closeBarKey}
}

func (b *cmdBar) suggestions() []suggestion {
	return suggest(barCommands(b.st.deps), b.st.cfg, b.text)
}

// handle takes every key while the bar is open; closed reports that esc, or
// an empty enter, closed it.
func (b *cmdBar) handle(k tea.KeyPressMsg) (cmd tea.Cmd, closed bool) {
	sugs := b.suggestions()
	switch {
	case key.Matches(k, closeBarKey):
		return nil, true
	case key.Matches(k, completeKey):
		if len(sugs) > 0 {
			b.text = complete(b.text, sugs[min(b.cursor, len(sugs)-1)])
			b.cursor, b.note = 0, ""
		}
	case k.Code == tea.KeyUp:
		b.cursor = max(b.cursor-1, 0)
	case k.Code == tea.KeyDown:
		b.cursor = min(b.cursor+1, max(len(sugs)-1, 0))
	case key.Matches(k, runKey):
		return b.run()
	case k.Code == tea.KeyBackspace:
		if r := []rune(b.text); len(r) > 0 {
			b.text = string(r[:len(r)-1])
		}
		b.cursor, b.note = 0, ""
	case k.Text != "":
		b.text += k.Text
		b.cursor, b.note = 0, ""
	}
	return nil, false
}

func (b *cmdBar) run() (tea.Cmd, bool) {
	l, err := readLine(barCommands(b.st.deps), b.text)
	switch {
	case err != nil:
		b.note, b.failed = err.Error(), true
	case l.cmd == nil:
		return nil, true
	case l.cmd.Names[0] == helpCommand.Names[0]:
		name := ""
		if len(l.flags.Args) > 0 {
			c := cli.Find(b.st.deps.Commands, l.flags.Args[0])
			if c == nil {
				b.note, b.failed = "no existe el comando «"+l.flags.Args[0]+"»", true
				return nil, false
			}
			name = c.Names[0]
		}
		return open(newHelpScreen(b.st, name)), true
	case l.cmd.Names[0] == "version":
		b.note, b.failed = "draco "+b.st.deps.Version, false
	default:
		b.note, b.failed = "La línea es válida, pero la barra todavía no abre «"+l.cmd.Names[0]+"»: elígelo en el menú.", false
	}
	return nil, false
}

// sugRows is how many suggestions the bar shows.
const sugRows = 5

func (b *cmdBar) view(width int) string {
	lines := []string{ruleStyle.Render(strings.Repeat("─", width)), " " + accentStyle.Render("/") + b.text + accentStyle.Render("█")}
	sugs := b.suggestions()
	cursor := min(b.cursor, max(len(sugs)-1, 0))
	labelWidth := 0
	for _, s := range sugs {
		labelWidth = max(labelWidth, lipgloss.Width(s.label))
	}
	start := window(cursor, len(sugs), sugRows)
	for i := start; i < min(start+sugRows, len(sugs)); i++ {
		s := sugs[i]
		mark, label := "   ", s.label
		if i == cursor {
			mark, label = accentStyle.Render(" ▸ "), accentStyle.Render(label)
		}
		row := mark + label + strings.Repeat(" ", labelWidth-lipgloss.Width(s.label)+2) + faintStyle.Render(s.desc)
		lines = append(lines, lipgloss.NewStyle().MaxWidth(width).Render(row))
	}
	switch {
	case b.note != "" && b.failed:
		lines = append(lines, lipgloss.NewStyle().Width(width-1).Render(" "+errStyle.Render("✗ "+b.note)))
	case b.note != "":
		lines = append(lines, lipgloss.NewStyle().Width(width-1).Render(" "+b.note))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
