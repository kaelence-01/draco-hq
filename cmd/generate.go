package cmd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/deploy"
	"github.com/kaelence-01/draco-hq/internal/profile"
)

func init() {
	register(&command{run: runGenerate, help: generateHelp, summary: "Crea o actualiza el documento de despliegue de una tarea"}, "generate", "gen", "g")
}

var generateFlags = []flag{
	{long: "--issue", short: "-i", value: true},
	{long: "--commit-backend", short: "-b", value: true},
	{long: "--commit-frontend", short: "-f", value: true},
	{long: "--project", short: "-p", value: true},
	{long: "--profile", value: true},
	{long: "--space", short: "-s", value: true},
	{long: "--template", value: true},
	{long: "--vcs-host", value: true},
	{long: "--vcs-org", value: true},
	{long: "--dry-run"},
}

// loadConfig and openProfile are replaced in tests, which run the commands
// against fakes and a temporary Markdown folder instead of the network.
var (
	loadConfig  = config.LoadLocal
	openProfile = profile.Open
)

func runGenerate(a *App, args []string) error {
	f, err := parseFlags(args, generateFlags)
	if err != nil {
		return usageError("%v", err)
	}
	if len(f.args) > 0 {
		return usageError("argumento inesperado: %s", f.args[0])
	}
	backend, frontend := deploy.SplitHashes(f.get("--commit-backend")), deploy.SplitHashes(f.get("--commit-frontend"))
	if f.get("--issue") == "" {
		return usageError("falta -i/--issue, la clave de la tarea")
	}
	if len(backend) == 0 && len(frontend) == 0 {
		return usageError("falta al menos un commit: -b (backend) o -f (frontend)")
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	prof, _, err := cfg.Profile(f.get("--profile"))
	if err != nil {
		return err
	}
	tmpl := f.get("--template")
	if tmpl == "" {
		tmpl = prof.Template
	}
	target, err := deploy.Resolve(cfg, deploy.Request{
		Issue: f.get("--issue"), BackendHashes: backend, FrontendHashes: frontend,
		Project: f.get("--project"), Space: f.get("--space"), Template: tmpl,
		VCSHost: f.get("--vcs-host"), VCSOrg: f.get("--vcs-org"),
	})
	if err != nil {
		return err
	}
	bundle, err := openProfile(cfg, profile.Options{Name: f.get("--profile"), Scope: target.SpaceKey})
	if err != nil {
		return explainProfile(err)
	}

	if target.ProjectName != "" {
		a.printf("%s %s\n", a.style(bold, "Proyecto:"), a.style(cyan, target.ProjectName))
	}
	a.printf("%s %s (%s → %s)\n\n", a.style(bold, "Perfil:"), a.style(cyan, bundle.Name), bundle.Profile.Tracker.Provider, bundle.Profile.Docs.Provider)
	for _, w := range target.Warnings {
		a.warn(w)
	}
	if len(target.Warnings) > 0 {
		a.printf("\n")
	}

	ctx, cancel := a.context()
	defer cancel()
	s := deploy.NewSession(bundle.Tracker, bundle.Docs, bundle.Changes, target)

	a.step(1, 4, "Buscando la tarea "+a.style(bold, target.Issue)+"...")
	if err := s.FetchIssue(ctx); err != nil {
		return err
	}
	a.ok(a.style(bold, s.Issue.Key) + " — " + s.Issue.Summary)
	a.printf("\n")

	a.step(2, 4, "Verificando documentos existentes...")
	if err := s.FindExisting(ctx); err != nil {
		return err
	}
	update := false
	if s.Existing != nil {
		a.warn("Ya existe un documento para " + target.Issue + ":")
		a.printf("      Título : %s\n      URL    : %s\n\n", s.Existing.Title, a.style(cyan, s.Existing.URL))
		switch a.ask("  [1] Actualizar    [2] Crear nuevo    [3] Cancelar\n  Opción: ") {
		case "1":
			update = true
		case "2":
		default:
			a.printf("Cancelado.\n")
			return nil
		}
	} else {
		a.ok("Ninguno encontrado")
	}
	a.printf("\n")

	a.step(3, 4, "Leyendo commits...")
	if err := s.ReadCommits(ctx); err != nil {
		return err
	}
	for _, side := range []struct {
		name string
		c    *deploy.Commits
	}{{"backend ", s.Backend}, {"frontend", s.Frontend}} {
		switch {
		case side.c == nil:
		case side.c.Err != nil:
			a.fail(fmt.Sprintf("%s [%s]: %v", side.name, strings.Join(side.c.Hashes, ", "), side.c.Err))
		default:
			a.ok(fmt.Sprintf("%s %s → %s archivos", side.name, a.style(bold, strings.Join(side.c.Hashes, ", ")), a.style(green, strconv.Itoa(side.c.Count))))
		}
	}
	if s.PartialFailure() {
		a.warn("uno de los lados falló: el documento se arma con lo que se pudo leer.")
	}
	a.printf("\n")

	var draft *deploy.Draft
	if update {
		if draft, err = s.DraftUpdate(ctx); err != nil {
			return err
		}
		if len(draft.NotPreserved) > 0 {
			a.warn("el documento actual no tiene " + quoted(draft.NotPreserved) + ": se publicaría como lo trae la plantilla.")
			if !a.confirm("  ¿Continuar de todas formas?", false) {
				a.printf("Cancelado.\n")
				return nil
			}
		} else {
			a.ok("lo escrito a mano en " + quoted(target.Template.Manifest.Preserve) + " se conserva")
		}
	} else if draft, err = s.DraftCreate(); err != nil {
		return err
	}

	if f.has("--dry-run") {
		a.printf("%s %s\n%s %s\n\n", a.style(bold, "Título:"), draft.Title, a.style(bold, "Destino:"), bundle.Profile.Docs.Provider)
		a.printf("%s\n", draft.Markdown)
		a.printf("%s\n", a.style(yellow, "(--dry-run: no se publicó nada)"))
		return nil
	}

	var res *deploy.Result
	if update {
		a.printf("%s %s\n\n", a.style(bold, "Título:"), draft.Title)
		if !a.confirm("¿Confirmas la actualización?", true) {
			a.printf("Cancelado.\n")
			return nil
		}
		a.step(4, 4, "Actualizando el documento...")
		if res, err = s.Publish(ctx, draft, nil); err != nil {
			return err
		}
	} else {
		a.step(4, 4, "Eligiendo la ubicación...")
		loc, err := s.Locations(ctx)
		if errors.Is(err, deploy.ErrNoLocations) {
			return fmt.Errorf("%w (o usa un perfil con destino Markdown, que no lo necesita)", err)
		}
		if err != nil {
			return err
		}
		if loc.OwnerUnknown {
			a.warn("no se pudo verificar tu usuario: se muestran documentos de todo el equipo.")
		} else if loc.TeamWide {
			a.warn("no tienes documentos propios aquí: se muestran los de todo el equipo como referencia.")
		}
		a.printf("\n")
		for i, d := range loc.Docs {
			a.printf("  %s %s\n      %s\n", a.style(bold+yellow, fmt.Sprintf("[%d]", i+1)), d.Title, a.style(cyan, d.URL))
		}
		n, err := strconv.Atoi(a.ask(fmt.Sprintf("\n  Opción (1-%d): ", len(loc.Docs))))
		if err != nil || n < 1 || n > len(loc.Docs) {
			return fmt.Errorf("opción inválida: ingresa un número entre 1 y %d", len(loc.Docs))
		}
		a.printf("\n%s %s\n\n", a.style(bold, "Título:"), draft.Title)
		if !a.confirm("¿Confirmas la creación?", true) {
			a.printf("Cancelado.\n")
			return nil
		}
		if res, err = s.Publish(ctx, draft, &loc.Docs[n-1]); err != nil {
			return err
		}
	}

	if res.LinkErr != nil {
		a.warn(fmt.Sprintf("no se pudo enlazar el documento desde %s: %v", target.Issue, res.LinkErr))
		a.warn("agrégalo a mano en la tarea; si no, el consolidado QA la marcará sin documentación.")
	}
	if res.Updated {
		a.ok(a.style(green+bold, "Documento actualizado"))
	} else {
		a.ok(a.style(green+bold, "Documento creado"))
	}
	a.printf("\n  %s\n\n", a.style(cyan, res.Page.URL))
	return nil
}

// explainProfile adds what to do to the errors of opening a profile.
func explainProfile(err error) error {
	switch {
	case errors.Is(err, profile.ErrNoSpace):
		return fmt.Errorf("%w. Opciones:\n"+
			"  1. --space <KEY> en este comando\n"+
			"  2. confluence_space_key en el proyecto\n"+
			"  3. docs.space en el perfil, o confluence_space_key en %s", err, configPathHint())
	case errors.Is(err, profile.ErrNoCredentials):
		return fmt.Errorf("%w (faltan atlassian_email, atlassian_token o base_url en %s, o sus variables ATLASSIAN_*)", err, configPathHint())
	}
	return err
}

func quoted(names []string) string {
	q := make([]string, len(names))
	for i, n := range names {
		q[i] = "«" + n + "»"
	}
	return strings.Join(q, ", ")
}

func generateHelp(a *App) {
	a.title("draco g, gen, generate", "Crea o actualiza el documento de despliegue de una tarea")

	a.section("Uso:")
	a.printf("  draco g -i <TAREA> [-b <HASH>] [-f <HASH>] [flags]\n\n")

	a.section("Flags:")
	a.printf("  -i, --issue            Clave de la tarea                    %s\n", a.style(yellow, "(requerido)"))
	a.printf("  -b, --commit-backend   Commits del backend                  (separados por coma)\n")
	a.printf("  -f, --commit-frontend  Commits del frontend                 (separados por coma)\n")
	a.printf("  -p, --project          Proyecto de la config\n")
	a.printf("      --profile          Perfil de la config (tracker y destino)\n")
	a.printf("  -s, --space            Space de Confluence o carpeta Markdown para esta ejecución\n")
	a.printf("      --template         Carpeta con una plantilla de despliegue\n")
	a.printf("      --vcs-host         Host del VCS para esta ejecución\n")
	a.printf("      --vcs-org          Organización o workspace del VCS para esta ejecución\n")
	a.printf("      --dry-run          Muestra el documento (Markdown) sin publicar nada\n")
	a.printf("  -h, --help             Esta ayuda\n\n")
	a.printf("  Al menos uno de -b o -f. Se aceptan -i OPS-7, -i=OPS-7 y --issue OPS-7.\n\n")

	a.section("Destino:")
	a.printf("  El del perfil: Confluence (space: --space > proyecto > perfil) o una carpeta\n")
	a.printf("  Markdown (carpeta: --space > perfil).\n\n")

	a.section("Sección «A considerar»:")
	a.printf("  Al %s se usa el checklist del proyecto o de la config; al %s\n", a.style(bold, "crear"), a.style(bold, "actualizar"))
	a.printf("  se conserva tal como quedó editada en el documento.\n\n")

	a.section("Ejemplos:")
	a.printf("  draco g -i OPS-7 -b a1b2c3d -f e4f5a6b\n")
	a.printf("  draco g -i OPS-7 -b a1b2c3d,9f8e7d6            # varios commits\n")
	a.printf("  draco g -i OPS-7 -b a1b2c3d -p pagos --space DEV\n")
	a.printf("  draco g -i OPS-7 -b a1b2c3d --profile externo  # a la carpeta Markdown del perfil\n")
	a.printf("  draco g -i OPS-7 -b a1b2c3d --dry-run          # vista previa sin publicar\n")
}
