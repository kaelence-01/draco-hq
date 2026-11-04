package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kaelence-01/draco-hq/internal/cli"
	"github.com/kaelence-01/draco-hq/internal/profile"
	"github.com/kaelence-01/draco-hq/internal/qa"
)

func init() {
	register(&command{run: runQA, help: qaHelp, flags: qaFlags, summary: "Publica el consolidado de pruebas QA (Sprint o Kanban)"}, "qa")
}

var qaFlags = []cli.Flag{
	{Long: "--jira-project", Short: "-j", Value: true, Desc: "Proyecto de Jira de qa_jira (sin él: el por defecto)"},
	{Long: "--sprint", Short: "-s", Value: true, Desc: "Número del sprint (sin él: modo Kanban)"},
	{Long: "--module", Short: "-m", Value: true, Desc: "Módulo (solo modo Sprint)"},
	{Long: "--space", Value: true, Desc: "Space de Confluence o carpeta Markdown para esta ejecución"},
	{Long: "--profile", Value: true, Desc: "Perfil de la config (su tracker debe soportar QA)"},
	{Long: "--template", Value: true, Desc: "Carpeta con una plantilla del consolidado"},
	{Long: "--dry-run", Desc: "Muestra el consolidado sin publicar nada"},
}

// now is replaced in tests, which fix the Kanban window.
var now = time.Now

func runQA(a *App, args []string) error {
	f, err := cli.Parse(args, qaFlags)
	if err != nil {
		return usageError("%v", err)
	}
	if len(f.Args) > 0 {
		return usageError("argumento inesperado: %s", f.Args[0])
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	project := f.Get("--jira-project")
	if keys := cfg.QAJira.ProjectKeys(); !f.Has("--jira-project") && len(keys) > 1 {
		project = a.ask(fmt.Sprintf("Proyecto de Jira (%s; Enter: %s): ", strings.Join(keys, ", "), keys[0]))
	}
	sprintText, module := f.Get("--sprint"), f.Get("--module")
	if !f.Has("--sprint") {
		sprintText = a.ask("Número de sprint (Enter para modo Kanban): ")
	}
	sprint := 0
	if sprintText != "" {
		if sprint, err = strconv.Atoi(sprintText); err != nil || sprint < 1 {
			return usageError("--sprint debe ser un número positivo: %s", sprintText)
		}
		if module == "" {
			module = a.ask("Módulo: ")
		}
	}

	prof, _, err := cfg.Profile(f.Get("--profile"))
	if err != nil {
		return err
	}
	tmpl := f.Get("--template")
	if tmpl == "" {
		tmpl = prof.TemplateDir("qa")
	}
	target, err := qa.Resolve(cfg, qa.Request{Project: project, Sprint: sprint, Module: module, Space: f.Get("--space"), Template: tmpl}, now())
	if err != nil {
		return err
	}
	bundle, err := openProfile(cfg, profile.Options{Name: f.Get("--profile"), Scope: target.SpaceKey, QAProject: target.Project})
	if err != nil {
		return explainProfile(err)
	}
	tracker, err := bundle.QA()
	if err != nil {
		return err
	}

	ctx, cancel := a.context()
	defer cancel()
	s := qa.NewSession(tracker, bundle.Docs, target)

	if err := s.CheckAccount(ctx); err != nil {
		if errors.Is(err, qa.ErrNotQAAccount) {
			configured := target.QAEmail
			if configured == "" {
				configured = "(sin configurar)"
			}
			return fmt.Errorf("%w.\n  Cuenta autenticada  : %s\n  qa_email del config : %s\n  Si te corresponde generar el consolidado, ajusta qa_email en %s",
				err, s.Account.Email, configured, configPathHint())
		}
		return err
	}

	a.printf("%s %s\n", a.style(bold, "Proyecto:"), a.style(cyan, target.Project))
	if target.Kanban {
		a.printf("%s %s\n\n", a.style(bold, "Período:"), a.style(cyan, target.Period))
	} else {
		a.printf("%s %s\n%s %s\n\n", a.style(bold, "Módulo:"), a.style(cyan, target.Module), a.style(bold, "Sprint:"), a.style(cyan, strconv.Itoa(target.Sprint)))
	}
	for _, w := range target.Warnings {
		a.warn(w)
	}

	a.step(1, 3, "Buscando tareas...")
	if err := s.Collect(ctx); err != nil {
		return err
	}
	if len(s.Tasks) == 0 {
		a.warn("no hay tareas en revisión para este período")
	} else {
		a.ok(fmt.Sprintf("%s tareas para revisión", a.style(bold, strconv.Itoa(len(s.Tasks)))))
	}
	a.ok(fmt.Sprintf("%s tareas QA propias", a.style(bold, strconv.Itoa(len(s.QATasks)))))
	a.printf("\n")

	a.step(2, 3, "Evaluando cada tarea...")
	mark := func(ok bool) string {
		if ok {
			return a.style(green, "✓")
		}
		return a.style(red, "✗")
	}
	for i := range s.Tasks {
		for _, w := range s.EvaluateTask(ctx, i) {
			a.warn(w)
		}
		t := s.Tasks[i]
		doc := mark(t.HasDeployDoc)
		if t.DeployDocUnknown {
			doc = a.style(yellow, "?")
		}
		a.printf("  %-10s %s %s %s %s\n", t.Key, mark(!t.HasCodingErrors), mark(!t.HasDevReturns), doc, mark(t.PRMerged))
	}
	a.printf("\n")

	if err := s.FindExisting(ctx); err != nil {
		return err
	}
	var draft *qa.Draft
	if s.Existing != nil {
		if draft, err = s.DraftUpdate(ctx); err != nil {
			return err
		}
	} else if draft, err = s.DraftCreate(); err != nil {
		return err
	}

	if f.Has("--dry-run") {
		a.printf("%s %s\n%s %s\n\n%s\n%s\n", a.style(bold, "Título:"), draft.Title, a.style(bold, "Destino:"), bundle.Profile.Docs.Provider, draft.Markdown, a.style(yellow, "(--dry-run: no se publicó nada)"))
		return nil
	}

	a.step(3, 3, "Publicando...")
	a.printf("%s %s\n\n", a.style(bold, "Título:"), draft.Title)
	if draft.Replaces != nil {
		a.warn("ya existe este consolidado: se actualiza conservando lo escrito a mano.")
		if len(draft.NotPreserved) > 0 {
			a.warn("el consolidado actual no tiene " + quoted(draft.NotPreserved) + ": se publicaría como lo trae la plantilla.")
		}
		if !a.confirm("¿Confirmas la publicación?", len(draft.NotPreserved) == 0) {
			a.printf("Cancelado.\n")
			return nil
		}
		res, err := s.Publish(ctx, draft, nil)
		if err != nil {
			return err
		}
		a.ok(a.style(green+bold, "Consolidado actualizado"))
		a.printf("\n  %s\n\n", a.style(cyan, res.Page.URL))
		return nil
	}

	locs, err := s.Locations(ctx)
	if errors.Is(err, qa.ErrNoLocations) {
		return fmt.Errorf("%w (por ejemplo «%s»)", err, draft.Title)
	}
	if err != nil {
		return err
	}
	for i, d := range locs {
		a.printf("  %s %s\n      %s\n", a.style(bold+yellow, fmt.Sprintf("[%d]", i+1)), d.Title, a.style(cyan, d.URL))
	}
	n, err := strconv.Atoi(a.ask(fmt.Sprintf("\n  Crear junto a (1-%d): ", len(locs))))
	if err != nil || n < 1 || n > len(locs) {
		return fmt.Errorf("opción inválida: ingresa un número entre 1 y %d", len(locs))
	}
	if !a.confirm("¿Confirmas la publicación?", true) {
		a.printf("Cancelado.\n")
		return nil
	}
	res, err := s.Publish(ctx, draft, &locs[n-1])
	if err != nil {
		return err
	}
	a.ok(a.style(green+bold, "Consolidado creado"))
	a.printf("\n  %s\n\n", a.style(cyan, res.Page.URL))
	return nil
}

func qaHelp(a *App) {
	a.title("draco qa", "Publica el consolidado de pruebas QA")

	a.section("Uso:")
	a.printf("  draco qa [-j <PROYECTO>] [-s <SPRINT> -m <MÓDULO>] [flags]\n\n")

	a.section("Flags:")
	a.printf("  -j, --jira-project Proyecto de Jira de qa_jira  (sin él: el por defecto)\n")
	a.printf("  -s, --sprint       Número del sprint             (sin él: modo Kanban)\n")
	a.printf("  -m, --module       Módulo                        (solo modo Sprint)\n")
	a.printf("      --space        Space de Confluence o carpeta Markdown para esta ejecución\n")
	a.printf("      --profile      Perfil de la config (su tracker debe soportar QA: Jira)\n")
	a.printf("      --template     Carpeta con una plantilla del consolidado\n")
	a.printf("      --dry-run      Muestra el consolidado (Markdown) sin publicar nada\n")
	a.printf("  -h, --help         Esta ayuda\n\n")
	a.printf("  Sin -j, con varios proyectos en qa_jira, pregunta cuál; Enter: el por defecto.\n")
	a.printf("  Sin -s pregunta el sprint; Enter elige el modo Kanban.\n\n")

	a.section("Modos:")
	a.printf("  Sprint  — tareas del módulo en el sprint (nombre según qa_jira.sprint_name)\n")
	a.printf("  Kanban  — %s las tareas del proyecto que pasaron a Testing en los últimos\n", a.style(bold, "todas"))
	a.printf("            10 días hábiles\n\n")

	a.section("Configuración:")
	a.printf("  qa_email     Cuenta que publica el consolidado (protección, no permisos)\n")
	a.printf("  qa_jira      Proyecto por defecto, estados y campos de Jira; projects para\n")
	a.printf("               otros proyectos; review_prefixes y observation_marker para las\n")
	a.printf("               tareas de revisión y observaciones\n")
	a.printf("  qa_report    Nombres del encabezado; los que faltan salen como —\n")
	a.printf("  Todo en %s.\n\n", a.style(cyan, configPathHint()))

	a.section("Ejemplos:")
	a.printf("  draco qa                       # Kanban: últimos 10 días hábiles\n")
	a.printf("  draco qa -s 12 -m Pagos        # Sprint\n")
	a.printf("  draco qa -j PAG -s 12 -m Pagos # Sprint de otro proyecto de qa_jira\n")
	a.printf("  draco qa -s 12 -m Pagos --dry-run\n")
}
