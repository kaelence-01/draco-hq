package cmd

import (
	"sort"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/build"
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

func runVersion(a *App, args []string) error {
	if _, err := parseFlags(args, nil); err != nil {
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
