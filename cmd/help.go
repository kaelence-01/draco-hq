package cmd

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/build"
	"github.com/kaelence-01/draco-hq/internal/cli"
	"github.com/kaelence-01/draco-hq/internal/updater"
)

func init() {
	register(&command{run: runVersion, help: versionHelp, summary: "Muestra la versión"}, "version")
}

// usage is the general help: one line per command; the detail is in each
// command's help (draco help <comando>).
func (a *App) usage() {
	a.title("draco "+build.Version, "documentación de despliegues desde la terminal")

	a.section("Uso:")
	a.printf("  draco                    Modo visual: menús, listas y formularios (en una terminal)\n")
	a.printf("  draco <comando> [flags]\n")
	a.printf("  draco help <comando>     Ayuda de un comando (también: draco <comando> -h)\n\n")

	a.section("Comandos:")
	for _, line := range commandLines() {
		a.printf("  %s\n", line)
	}
	a.printf("\n")

	a.printf("Configuración en %s.\n", a.style(cyan, configPathHint()))
	a.printf("Documentación y versiones: %s\n", a.style(cyan, "https://github.com/"+updater.Repo))
}

// commandLines lists each command once, with its aliases, sorted by name.
func commandLines() []string {
	names := map[*command][]string{}
	for n, c := range commands {
		names[c] = append(names[c], n)
	}
	var lines []string
	width := 0
	type entry struct{ label, summary string }
	var entries []entry
	for c, ns := range names {
		sort.Slice(ns, func(i, j int) bool { return len(ns[i]) < len(ns[j]) || len(ns[i]) == len(ns[j]) && ns[i] < ns[j] })
		label := strings.Join(ns, ", ")
		if len(label) > width {
			width = len(label)
		}
		entries = append(entries, entry{label, c.summary})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].label < entries[j].label })
	for _, e := range entries {
		lines = append(lines, e.label+strings.Repeat(" ", width-len(e.label)+2)+e.summary)
	}
	return lines
}

// commandSpecs describes every command once, sorted by name, for the visual
// mode's command bar.
func commandSpecs() []cli.Command {
	seen := map[*command]bool{}
	var out []cli.Command
	for _, c := range commands {
		if !seen[c] {
			seen[c] = true
			out = append(out, cli.Command{Names: c.names, Summary: c.summary, Flags: c.flags, Subcommands: c.subs})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Names[0] < out[j].Names[0] })
	return out
}

// helpText is the help of a command without color, as draco help <name>
// prints it.
func helpText(name string) (string, error) {
	c, ok := commands[name]
	if !ok {
		return "", fmt.Errorf("no existe el comando %q", name)
	}
	var out bytes.Buffer
	c.help(New(strings.NewReader(""), &out, &out))
	return out.String(), nil
}

func runVersion(a *App, args []string) error {
	if _, err := cli.Parse(args, nil); err != nil {
		return usageError("%v", err)
	}
	a.printf("draco %s\n", build.Version)
	return nil
}

func versionHelp(a *App) {
	a.title("draco version", "Muestra la versión instalada")
	a.section("Uso:")
	a.printf("  draco version     (también: draco --version, draco -v)\n")
}
