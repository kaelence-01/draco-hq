package cmd

import (
	"strconv"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/cli"
	"github.com/kaelence-01/draco-hq/internal/history"
)

func init() {
	register(&command{run: runHistory, help: historyHelp, flags: historyFlags, summary: "Lista los documentos de despliegue que publicaste"}, "history", "hist")
}

var historyFlags = []cli.Flag{
	{Long: "--project", Short: "-p", Value: true, Desc: "Solo los documentos de este proyecto"},
	{Long: "--limit", Short: "-n", Value: true, Desc: "Cuántos mostrar (por defecto 10)"},
}

// historyPath is replaced in tests, which keep the history in a temporary
// folder.
var historyPath = history.Path

func runHistory(a *App, args []string) error {
	f, err := cli.Parse(args, historyFlags)
	if err != nil {
		return usageError("%v", err)
	}
	if len(f.Args) > 0 {
		return usageError("argumento inesperado: %s", f.Args[0])
	}
	limit := 10
	if v := f.Get("--limit"); v != "" {
		if limit, err = strconv.Atoi(v); err != nil || limit < 1 {
			return usageError("-n/--limit es un número mayor que cero, no %q", v)
		}
	}
	path, err := historyPath()
	if err != nil {
		return err
	}
	all, err := history.Load(path)
	if err != nil {
		return err
	}
	project := f.Get("--project")
	entries := history.Of(all, project)
	if len(entries) == 0 {
		if project != "" && len(all) > 0 {
			a.printf("No hay documentos de %s en el historial.\n", project)
			return nil
		}
		a.printf("Todavía no hay documentos en el historial: se agregan al publicar con draco g o desde el modo visual.\n")
		return nil
	}

	shown := entries[:min(limit, len(entries))]
	width := 0
	for _, e := range shown {
		width = max(width, len(e.Issue))
	}
	indent := "  " + spaces(width+2)
	for i, e := range shown {
		if i > 0 {
			a.printf("\n")
		}
		action := "creado"
		if e.Updated {
			action = "actualizado"
		}
		line := "  " + a.style(bold, e.Issue) + spaces(width-len(e.Issue)+2) + e.At.Local().Format("2006-01-02 15:04") + "  " + action
		if e.Project != "" && project == "" {
			line += spaces(len("actualizado")-len(action)+2) + a.style(cyan, e.Project)
		}
		a.printf("%s\n%s%s\n%s%s\n", line, indent, e.Title, indent, a.style(cyan, e.URL))
		if e.LinkFailed {
			a.printf("%s%s\n", indent, a.style(yellow, "⚠ sin enlace desde la tarea: el consolidado QA la marcará sin documentación (draco g lo vuelve a enlazar)"))
		}
	}
	if len(entries) > len(shown) {
		a.printf("\n%d de %d. Más con -n %d.\n", len(shown), len(entries), len(entries))
	}
	return nil
}

// record adds a published document to the history; the document is already
// published, so a failure is only a warning.
func (a *App) record(e history.Entry) {
	path, err := historyPath()
	if err == nil {
		err = history.Record(path, e)
	}
	if err != nil {
		a.warn("no se guardó en el historial: " + err.Error())
	}
}

func spaces(n int) string { return strings.Repeat(" ", max(n, 0)) }

func historyHelp(a *App) {
	a.title("draco history, hist", "Lista los documentos de despliegue que publicaste")

	a.section("Uso:")
	a.printf("  draco history [-p <PROYECTO>] [-n <CANTIDAD>]\n\n")

	a.section("Flags:")
	a.printf("  -p, --project   Solo los documentos de este proyecto\n")
	a.printf("  -n, --limit     Cuántos mostrar                      (por defecto 10)\n")
	a.printf("  -h, --help      Esta ayuda\n\n")

	a.section("Qué guarda:")
	a.printf("  Cada documento que publicas con draco g o desde el modo visual, una vez por\n")
	a.printf("  documento y del más reciente al más antiguo: tarea, título, dirección, fecha,\n")
	a.printf("  proyecto y si quedó sin enlace desde la tarea. Vive en tu equipo, junto a\n")
	a.printf("  config.yaml (history.json), y se lee sin red. En el modo visual se abre,\n")
	a.printf("  actualiza o exporta desde la principal o desde Historial.\n\n")

	a.section("Ejemplos:")
	a.printf("  draco history\n")
	a.printf("  draco history -p pagos\n")
	a.printf("  draco hist -n 30\n")
}
