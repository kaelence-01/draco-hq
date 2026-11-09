package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/cli"
	"github.com/kaelence-01/draco-hq/internal/deploy"
	"github.com/kaelence-01/draco-hq/internal/history"
	"github.com/kaelence-01/draco-hq/internal/profile"
)

func init() {
	register(&command{run: runHistory, help: historyHelp, flags: historyFlags, summary: "Lista los documentos de despliegue que publicaste",
		subs: []cli.Command{{Names: []string{"remove", "rm"}, Summary: "Borra un documento del historial y su enlace desde la tarea", Flags: historyRemoveFlags}}}, "history", "hist")
}

var historyFlags = []cli.Flag{
	{Long: "--project", Short: "-p", Value: true, Desc: "Solo los documentos de este proyecto"},
	{Long: "--limit", Short: "-n", Value: true, Desc: "Cuántos mostrar (por defecto 10)"},
}

var historyRemoveFlags = []cli.Flag{
	{Long: "--issue", Short: "-i", Value: true, Desc: "Tarea del documento a borrar (requerido)"},
}

// historyPath is replaced in tests, which keep the history in a temporary
// folder.
var historyPath = history.Path

func runHistory(a *App, args []string) error {
	if len(args) > 0 && (args[0] == "remove" || args[0] == "rm") {
		return runHistoryRemove(a, args[1:])
	}
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

// runHistoryRemove deletes a document draco published, after the user types
// its task, and then takes it out of the history.
func runHistoryRemove(a *App, args []string) error {
	f, err := cli.Parse(args, historyRemoveFlags)
	if err != nil {
		return usageError("%v", err)
	}
	if len(f.Args) > 0 {
		return usageError("argumento inesperado: %s", f.Args[0])
	}
	issue := strings.TrimSpace(f.Get("--issue"))
	if issue == "" {
		return usageError("falta -i/--issue, la tarea del documento")
	}
	path, err := historyPath()
	if err != nil {
		return err
	}
	all, err := history.Load(path)
	if err != nil {
		return err
	}
	matches := history.OfIssue(all, issue)
	if len(matches) == 0 {
		return fmt.Errorf("%s no tiene documentos en el historial: draco solo borra lo que publicaste con draco (ver draco history)", issue)
	}
	e := matches[0]
	if len(matches) > 1 {
		a.printf("%s tiene %d documentos en el historial:\n\n", e.Issue, len(matches))
		for i, m := range matches {
			a.printf("  %s %s  %s\n      %s\n", a.style(bold+yellow, fmt.Sprintf("[%d]", i+1)), m.At.Local().Format("2006-01-02 15:04"), m.Title, a.style(cyan, m.URL))
		}
		n, err := strconv.Atoi(a.ask(fmt.Sprintf("\n  Opción (1-%d): ", len(matches))))
		if err != nil || n < 1 || n > len(matches) {
			return fmt.Errorf("opción inválida: ingresa un número entre 1 y %d", len(matches))
		}
		e = matches[n-1]
		a.printf("\n")
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	bundle, err := openProfile(cfg, profile.Options{Name: e.Profile, AnyScope: true})
	if err != nil {
		return explainProfile(err)
	}
	if err := e.CheckDest(bundle.Profile.Docs.Provider); err != nil {
		return err
	}

	a.printf("%s %s\n  %s\n\n", a.style(bold, "Documento:"), e.Title, a.style(cyan, e.URL))
	if e.Dest == "confluence" {
		a.printf("  Va a la papelera de Confluence: se puede restaurar desde ahí.\n")
	} else {
		a.printf("  Se borra el archivo: no hay papelera.\n")
	}
	a.printf("  Se quita el enlace desde %s en %s.\n\n", e.Issue, bundle.Profile.Tracker.Provider)
	if typed := a.ask("Escribe " + e.Issue + " para confirmar: "); !strings.EqualFold(typed, e.Issue) {
		a.printf("Cancelado: no se borró nada.\n")
		return nil
	}

	ctx, cancel := a.context()
	defer cancel()
	res, err := deploy.Remove(ctx, bundle.Tracker, bundle.Docs, e.Issue, e.Ref())
	if err != nil {
		return err
	}
	if res.Missing {
		a.warn("el documento ya no existía")
	} else {
		a.ok("Documento borrado")
	}
	if res.LinkErr != nil {
		a.warn(fmt.Sprintf("no se pudo quitar el enlace desde %s: %v", e.Issue, res.LinkErr))
		a.warn("quítalo a mano en la tarea; si no, el consolidado QA la dará por documentada.")
	} else {
		a.ok("Enlace quitado desde " + e.Issue)
	}
	if err := history.Remove(path, e); err != nil {
		a.warn("sigue en el historial: " + err.Error())
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
	a.printf("  draco history [-p <PROYECTO>] [-n <CANTIDAD>]\n")
	a.printf("  draco history remove -i <TAREA>     (también: rm)\n\n")

	a.section("Flags:")
	a.printf("  -p, --project   Solo los documentos de este proyecto\n")
	a.printf("  -n, --limit     Cuántos mostrar                      (por defecto 10)\n")
	a.printf("  -h, --help      Esta ayuda\n\n")

	a.section("Borrar (remove, rm):")
	a.printf("  -i, --issue     Tarea del documento a borrar         %s\n\n", a.style(yellow, "(requerido)"))
	a.printf("  Borra un documento que publicaste con draco y quita su enlace desde la tarea,\n")
	a.printf("  para que el consolidado QA no la dé por documentada. En Confluence la página\n")
	a.printf("  va a la papelera y se puede restaurar; un archivo Markdown se borra. Pide\n")
	a.printf("  escribir la tarea para confirmar. Solo borra lo que está en el historial.\n\n")

	a.section("Qué guarda:")
	a.printf("  Cada documento que publicas con draco g o desde el modo visual, una vez por\n")
	a.printf("  documento y del más reciente al más antiguo: tarea, título, dirección, fecha,\n")
	a.printf("  proyecto y si quedó sin enlace desde la tarea. Vive en tu equipo, junto a\n")
	a.printf("  config.yaml (history.json), y se lee sin red. En el modo visual se abre,\n")
	a.printf("  actualiza, exporta o borra desde la principal o desde Historial.\n\n")

	a.section("Ejemplos:")
	a.printf("  draco history\n")
	a.printf("  draco history -p pagos\n")
	a.printf("  draco hist -n 30\n")
	a.printf("  draco history rm -i OPS-7\n")
}
