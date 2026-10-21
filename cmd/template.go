package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/cli"
	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/document"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

func init() {
	register(&command{run: runTemplate, help: templateHelp, summary: "Copia, lista, valida y previsualiza las plantillas de los documentos", subs: []cli.Command{
		{Names: []string{"init"}, Summary: "Copia la plantilla incluida para editarla", Flags: templateInitFlags},
		{Names: []string{"list", "ls"}, Summary: "Qué carpetas se buscan y cuál está en uso", Flags: templateLookupFlags},
		{Names: []string{"validate"}, Summary: "Carga y llena la plantilla con datos de ejemplo", Flags: templateLookupFlags},
		{Names: []string{"preview"}, Summary: "Muestra el título y el Markdown con datos de ejemplo", Flags: templateLookupFlags},
	}}, "template")
}

var kindNames = map[string]string{"deploy": "documento de despliegue", "qa": "consolidado QA"}

var templateLookupFlags = []cli.Flag{
	{Long: "--template", Value: true, Desc: "Carpeta de una plantilla (requiere el tipo)"},
	{Long: "--profile", Value: true, Desc: "Perfil cuya carpeta de plantillas se usa"},
	{Long: "--project", Short: "-p", Value: true, Desc: "Proyecto: su repo backend es el de .draco/templates/deploy"},
}

var templateInitFlags = []cli.Flag{
	{Long: "--user", Desc: "En la carpeta de plantillas del usuario, para todos tus repos"},
	{Long: "--dir", Value: true, Desc: "En esa carpeta (por ejemplo, la del perfil)"},
	{Long: "--force", Desc: "Sobrescribe los archivos que ya existan"},
}

func runTemplate(a *App, args []string) error {
	return a.subcommand("template", map[string]func(*App, []string) error{
		"init":     runTemplateInit,
		"list":     runTemplateList,
		"ls":       runTemplateList,
		"validate": runTemplateValidate,
		"preview":  runTemplatePreview,
	}, templateHelp, args)
}

// kindsArg reads the optional kind argument; without it, every kind. A
// --template folder is of one kind, so it needs the argument.
func kindsArg(f *cli.Parsed, usage string) ([]string, error) {
	args := f.Args
	switch len(args) {
	case 0:
		if f.Has("--template") {
			return nil, usageError("--template es la carpeta de una plantilla: indica cuál (deploy o qa)")
		}
		return document.Kinds, nil
	case 1:
		if !slices.Contains(document.Kinds, args[0]) {
			return nil, usageError("plantilla desconocida %q: usa %s", args[0], strings.Join(document.Kinds, " o "))
		}
		return args, nil
	}
	return nil, usageError("uso: %s", usage)
}

// templateSource is one folder that may customize a template.
type templateSource struct {
	label, dir string
}

// lookup resolves the folders of a kind as the commands do: --template or the
// profile's, the repo's (for deploy, the project's backend) and the user's.
type lookup struct {
	cfg         *config.Config
	flag        string // --template: the folder of one kind
	profile     *config.Profile
	profileName string
	repoDir     string
	userRoot    string
}

func newLookup(f *cli.Parsed) (*lookup, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	l := &lookup{cfg: cfg, flag: f.Get("--template")}
	if l.profile, l.profileName, err = cfg.Profile(f.Get("--profile")); err != nil {
		return nil, err
	}
	proj, _, err := cfg.GetProject(f.Get("--project"))
	if err != nil {
		return nil, err
	}
	if proj != nil {
		l.repoDir = proj.BackendPath
	}
	l.userRoot, _ = config.TemplatesDir()
	return l, nil
}

func (l *lookup) sources(kind string) []templateSource {
	repoDir := l.repoDir
	if kind != "deploy" {
		repoDir = ""
	}
	explicit, label := l.flag, "--template"
	if explicit == "" {
		explicit, label = l.profile.TemplateDir(kind), "perfil "+l.profileName
	}
	dirs := document.Dirs(kind, explicit, repoDir, l.userRoot)
	labels := []string{label, "repositorio", "usuario"}
	var out []templateSource
	for i, d := range dirs {
		if d != "" {
			out = append(out, templateSource{labels[i], d})
		}
	}
	return out
}

func (l *lookup) load(kind string) (*document.Template, error) {
	var dirs []string
	for _, s := range l.sources(kind) {
		dirs = append(dirs, s.dir)
	}
	return document.LoadTemplate(kind, dirs...)
}

func runTemplateList(a *App, args []string) error {
	f, err := cli.Parse(args, templateLookupFlags)
	if err != nil {
		return usageError("%v", err)
	}
	kinds, err := kindsArg(f, "draco template list [deploy|qa]")
	if err != nil {
		return err
	}
	l, err := newLookup(f)
	if err != nil {
		return err
	}
	for _, kind := range kinds {
		a.section(fmt.Sprintf("%s — %s", kind, kindNames[kind]))
		used := false
		for _, s := range l.sources(kind) {
			state := "no existe"
			if info, err := os.Stat(s.dir); err == nil && info.IsDir() {
				state = "ignorada: gana la de arriba"
				if !used {
					used, state = true, a.style(green, "en uso")
				}
			}
			a.printf("  %-14s %s  (%s)\n", s.label, s.dir, state)
		}
		incluida := "base de todas"
		if !used {
			incluida = a.style(green, "en uso")
		}
		a.printf("  %-14s %s  (%s)\n\n", "incluida", "—", incluida)
	}
	a.printf("Copia una para editarla con: draco template init <deploy|qa>\n")
	return nil
}

func runTemplateInit(a *App, args []string) error {
	f, err := cli.Parse(args, templateInitFlags)
	if err != nil {
		return usageError("%v", err)
	}
	if len(f.Args) != 1 || !slices.Contains(document.Kinds, f.Args[0]) {
		return usageError("uso: draco template init <deploy|qa> [--user | --dir <carpeta>] [--force]")
	}
	kind := f.Args[0]
	if f.Has("--user") && f.Has("--dir") {
		return usageError("usa --user o --dir, no los dos")
	}
	dir := filepath.Join(".draco", "templates", kind)
	switch {
	case f.Has("--dir"):
		dir = f.Get("--dir")
	case f.Has("--user"):
		root, err := config.TemplatesDir()
		if err != nil {
			return err
		}
		dir = filepath.Join(root, kind)
	}

	files, err := document.BuiltinFiles(kind)
	if err != nil {
		return err
	}
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	if !f.Has("--force") {
		var existing []string
		for _, n := range names {
			if _, err := os.Stat(filepath.Join(dir, n)); err == nil {
				existing = append(existing, n)
			} else if !errors.Is(err, fs.ErrNotExist) {
				return err
			}
		}
		if len(existing) > 0 {
			return fmt.Errorf("%s ya tiene %s: no se reemplaza lo que editaste (usa --force para sobrescribir)", dir, strings.Join(existing, ", "))
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), files[n], 0o644); err != nil {
			return err
		}
	}
	a.ok(fmt.Sprintf("Plantilla %s copiada en %s (%s)", kind, dir, strings.Join(names, ", ")))
	a.printf("\n  Edita template.yaml (título, prefijo, secciones a conservar) y main.md.tmpl.\n")
	a.printf("  Para cambiar solo un bloque, deja en main.md.tmpl únicamente su\n")
	a.printf("  {{define \"<bloque>\"}}...{{end}}: el resto sigue la plantilla incluida.\n")
	a.printf("  Revisa el resultado con: draco template preview %s\n", kind)
	return nil
}

func runTemplateValidate(a *App, args []string) error {
	f, err := cli.Parse(args, templateLookupFlags)
	if err != nil {
		return usageError("%v", err)
	}
	kinds, err := kindsArg(f, "draco template validate [deploy|qa]")
	if err != nil {
		return err
	}
	l, err := newLookup(f)
	if err != nil {
		return err
	}
	var failed []string
	for _, kind := range kinds {
		t, title, md, err := l.render(kind)
		if err != nil {
			a.fail(fmt.Sprintf("%s: %v", kind, err))
			failed = append(failed, kind)
			continue
		}
		a.ok(fmt.Sprintf("%s (%s): «%s»", kind, sourceName(t), title))
		if missing := t.MissingPreserved(md); len(missing) > 0 {
			a.warn(fmt.Sprintf("%s: el documento no tiene %s; lo escrito ahí a mano se perdería al actualizar", kind, quoted(missing)))
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("plantillas con errores: %s", strings.Join(failed, ", "))
	}
	return nil
}

func runTemplatePreview(a *App, args []string) error {
	f, err := cli.Parse(args, templateLookupFlags)
	if err != nil {
		return usageError("%v", err)
	}
	if len(f.Args) > 1 || len(f.Args) == 1 && !slices.Contains(document.Kinds, f.Args[0]) {
		return usageError("uso: draco template preview [deploy|qa]")
	}
	kind := "deploy"
	if len(f.Args) == 1 {
		kind = f.Args[0]
	}
	l, err := newLookup(f)
	if err != nil {
		return err
	}
	t, title, md, err := l.render(kind)
	if err != nil {
		return err
	}
	a.printf("%s %s\n%s %s\n\n", a.style(bold, "Plantilla:"), sourceName(t), a.style(bold, "Título:"), title)
	a.printf("%s\n", md)
	a.printf("%s\n", a.style(yellow, "(vista previa con datos de ejemplo)"))
	return nil
}

func sourceName(t *document.Template) string {
	if t.Source == "" {
		return "incluida"
	}
	return t.Source
}

// render fills the template of kind with sample data, so a template is
// checked without a tracker or a destination.
func (l *lookup) render(kind string) (t *document.Template, title, md string, err error) {
	if t, err = l.load(kind); err != nil {
		return nil, "", "", err
	}
	proj, _, _ := l.cfg.GetProject("")
	if kind == "qa" {
		doc := document.QADoc{Sprint: 12, Module: "Pagos", Date: now(), Tasks: []provider.QAIssue{
			{Key: "OPS-7", URL: "https://example.test/browse/OPS-7", HasDeployDoc: true, PRMerged: true,
				ReviewTaskKey: "OPS-8", ReviewTaskURL: "https://example.test/browse/OPS-8"},
			{Key: "OPS-9", URL: "https://example.test/browse/OPS-9", HasDevReturns: true, DeployDocUnknown: true,
				Observations: "Falta validar el filtro por fecha"},
		}}
		if r := l.cfg.QAReport; r != nil {
			doc.LiderTecnico, doc.PMO, doc.QA = r.LiderTecnico, r.PMO, r.QA
		}
		if title, err = document.QATitle(t, doc.Module, doc.Sprint, ""); err == nil {
			md, err = document.QA(t, doc)
		}
		return t, title, md, err
	}
	doc := document.DeployDoc{
		IssueKey: "OPS-7", IssueSummary: "Filtro de pagos", IssueURL: "https://example.test/browse/OPS-7",
		VCSHost: "https://github.com", VCSOrg: "mi-org",
		BackendRepo: "pagos-api", BackendCommit: "a1b2c3d",
		BackendFiles: map[string][]string{"app/Http/Pagos": {"FiltroController.php"}, "routes": {"api.php"}},
		FrontendRepo: "pagos-web", FrontendCommit: "e4f5a6b",
		FrontendFiles: map[string][]string{"src/app/pagos": {"filtro.component.ts"}},
		Checklist:     l.cfg.ResolveDeployChecklist(proj),
	}
	if title, err = document.DeployTitle(t, doc.IssueKey, doc.IssueSummary); err == nil {
		md, err = document.Deploy(t, doc)
	}
	return t, title, md, err
}

func templateHelp(a *App) {
	a.title("draco template", "Plantillas del documento de despliegue (deploy) y del consolidado QA (qa)")

	a.section("Uso:")
	a.printf("  draco template <subcomando> [deploy|qa] [flags]\n\n")

	a.section("Subcomandos:")
	a.printf("  init <deploy|qa>     Copia la plantilla incluida para editarla\n")
	a.printf("                       (por defecto en .draco/templates/<tipo> del repo actual)\n")
	a.printf("      --user           En %s<tipo>, para todos tus repos\n", userTemplatesHint())
	a.printf("      --dir <carpeta>  En esa carpeta (por ejemplo, la del perfil)\n")
	a.printf("      --force          Sobrescribe los archivos que ya existan\n")
	a.printf("  list, ls [tipo]      Qué carpetas se buscan y cuál está en uso\n")
	a.printf("  validate [tipo]      Carga y llena la plantilla con datos de ejemplo; avisa si\n")
	a.printf("                       falta una sección que se conserva al actualizar\n")
	a.printf("  preview [tipo]       Muestra el título y el Markdown con datos de ejemplo\n")
	a.printf("                       (por defecto: deploy)\n\n")

	a.section("Flags de list, validate y preview:")
	a.printf("      --template       Carpeta de una plantilla, como en draco g y draco qa\n")
	a.printf("                       (requiere el tipo)\n")
	a.printf("      --profile        Perfil cuya carpeta de plantillas se usa\n")
	a.printf("  -p, --project        Proyecto: su repo backend es el de .draco/templates/deploy\n\n")

	a.section("Orden de búsqueda (gana la primera carpeta que existe):")
	a.printf("  1. --template, o <carpeta de plantillas del perfil>/<tipo>\n")
	a.printf("  2. .draco/templates/<tipo> del repo (deploy: el backend del proyecto; qa: la carpeta actual)\n")
	a.printf("  3. %s<tipo>\n", userTemplatesHint())
	a.printf("  La plantilla incluida es la base: la carpeta solo reemplaza lo que trae.\n\n")

	a.section("Ejemplos:")
	a.printf("  draco template init deploy        # .draco/templates/deploy en este repo\n")
	a.printf("  draco template init qa --user\n")
	a.printf("  draco template init deploy --dir ~/plantillas/equipo/deploy   # perfil con template: ~/plantillas/equipo\n")
	a.printf("  draco template list\n")
	a.printf("  draco template validate\n")
	a.printf("  draco template preview qa\n")
}

func userTemplatesHint() string {
	return strings.TrimSuffix(configPathHint(), "config.yaml") + "templates" + string(filepath.Separator)
}
