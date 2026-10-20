package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/backlog"
	"github.com/kaelence-01/draco-hq/internal/config"
)

func init() {
	register(&command{run: runBacklog, help: backlogHelp, summary: "Detecta deuda técnica en tus repos y la propone como tareas"}, "backlog")
}

var backlogScanFlags = []flag{
	{long: "--project", short: "-p", value: true},
	{long: "--repo", short: "-r", value: true},
	{long: "--path", value: true},
	{long: "--since", value: true},
	{long: "--modulo", short: "-m", value: true},
	{long: "--limit", short: "-l", value: true},
	{long: "--output", short: "-o", value: true, optional: true},
	{long: "--json"},
	{long: "--deps"},
}

const defaultBacklogLimit = 10

// scanRepo is replaced in tests, which feed the command fixed reports.
var scanRepo = backlog.Scan

func runBacklog(a *App, args []string) error {
	return a.subcommand("backlog", map[string]func(*App, []string) error{"scan": runBacklogScan}, backlogHelp, args)
}

// runBacklogScan only reads the repos: it needs no credentials and no profile.
func runBacklogScan(a *App, args []string) error {
	f, err := parseFlags(args, backlogScanFlags)
	if err != nil {
		return usageError("%v", err)
	}
	if len(f.args) > 0 {
		return usageError("argumento inesperado: %s", f.args[0])
	}
	asJSON, saveReport := f.has("--json"), f.has("--output")
	output, pathFlag := f.get("--output"), f.get("--path")
	modules := splitModules(f.get("--modulo"))

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	proj, projName, err := cfg.GetProject(f.get("--project"))
	if err != nil {
		return err
	}
	targets, err := resolveBacklogTargets(pathFlag, f.get("--repo"), proj, projName)
	if err != nil {
		return err
	}
	limit, err := parseBacklogLimit(f.get("--limit"))
	if err != nil {
		return usageError("%v", err)
	}
	if saveReport {
		if err := validateBacklogOutput(output, len(targets)); err != nil {
			return usageError("%v", err)
		}
	}

	var reports []*backlog.Report
	var failed []string
	for i, t := range targets {
		if !asJSON {
			a.step(i+1, len(targets), "Analizando "+t.describe(projName, pathFlag != "")+"...")
		}

		opts := backlogOptions(proj.BacklogFor(t.repo), f.get("--since"))
		rep, err := scanRepo(t.dir, backlog.ScanOptions{Options: opts, Dependencies: f.has("--deps")})
		if err != nil {
			if len(targets) == 1 {
				return err
			}
			a.backlogFailure(asJSON, t.repo, err)
			failed = append(failed, t.repo)
			continue
		}

		if len(modules) > 0 {
			if len(rep.Modules) == 0 {
				a.backlogFailure(asJSON, t.repo, fmt.Errorf("-m necesita backlog.%s.modules en la config del proyecto (ver: draco help backlog)", t.repo))
				failed = append(failed, t.repo)
				continue
			}
			rep.FilterModules(modules)
		}

		// The saved file keeps every finding; --limit only trims what is shown.
		savedTo := ""
		if saveReport {
			name := backlogReportName(projName, t.dir, t.repo, modules)
			if savedTo, err = saveBacklogReport(rep, backlogOutputPath(output, len(targets), name), name); err != nil {
				return err
			}
		}
		if limit > 0 && len(rep.Findings) > limit {
			rep.Findings = rep.Findings[:limit]
		}
		reports = append(reports, rep)

		if !asJSON {
			a.printBacklogReport(rep)
			if savedTo != "" {
				a.ok("Reporte completo guardado en " + savedTo)
			} else if len(rep.Findings) > 0 {
				a.printf("Usa %s para guardar el reporte completo en JSON.\n", a.style(bold, "-o"))
			}
			a.printf("\n")
		}
	}

	if asJSON {
		enc := json.NewEncoder(a.Out)
		enc.SetIndent("", "  ")
		if err := enc.Encode(reports); err != nil {
			return err
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("no se pudo analizar: %s", strings.Join(failed, ", "))
	}
	return nil
}

type backlogTarget struct {
	repo string // "backend" or "frontend"
	dir  string
}

func (t backlogTarget) describe(projName string, fromPathFlag bool) string {
	if projName == "" || fromPathFlag {
		return t.dir
	}
	return fmt.Sprintf("%s (%s %s)", t.dir, projName, t.repo)
}

// resolveBacklogTargets: --path scans that folder; --repo, that repo of the
// project; neither, every repo the project has a path for. Without a project
// (or one with no paths), the current folder.
func resolveBacklogTargets(pathFlag, repoFlag string, proj *config.ProjectConfig, projName string) ([]backlogTarget, error) {
	if repoFlag != "" && repoFlag != "backend" && repoFlag != "frontend" {
		return nil, fmt.Errorf("--repo debe ser backend o frontend, no %q", repoFlag)
	}
	repo := repoFlag
	if repo == "" {
		repo = "backend"
	}
	if pathFlag != "" {
		return []backlogTarget{{repo: repo, dir: pathFlag}}, nil
	}

	var targets []backlogTarget
	if proj != nil {
		if repoFlag != "frontend" && proj.BackendPath != "" {
			targets = append(targets, backlogTarget{repo: "backend", dir: proj.BackendPath})
		}
		if repoFlag != "backend" && proj.FrontendPath != "" {
			targets = append(targets, backlogTarget{repo: "frontend", dir: proj.FrontendPath})
		}
	}
	if len(targets) > 0 {
		return targets, nil
	}
	if proj != nil && repoFlag != "" {
		return nil, fmt.Errorf("el proyecto %s no tiene %s_path configurado. Usa --path o agrégalo al proyecto en %s", projName, repoFlag, configPathHint())
	}
	return []backlogTarget{{repo: repo, dir: "."}}, nil
}

// validateBacklogOutput: with several repos each needs its own file, so -o
// must be a folder.
func validateBacklogOutput(output string, targets int) error {
	if targets > 1 && strings.EqualFold(filepath.Ext(output), ".json") {
		return fmt.Errorf("con varios repos -o recibe una carpeta, no un archivo (%s). Usa -r para analizar uno solo", output)
	}
	return nil
}

// backlogOutputPath: the given file with one repo, <folder>/<name> with
// several, and "" (the default location) when -o has no value.
func backlogOutputPath(output string, targets int, name string) string {
	if output == "" || targets == 1 {
		return output
	}
	return filepath.Join(output, name)
}

// backlogFailure goes to Err with --json, so Out stays valid JSON.
func (a *App) backlogFailure(asJSON bool, repo string, err error) {
	if asJSON {
		fmt.Fprintf(a.Err, "No se pudo analizar %s: %v\n", repo, err)
		return
	}
	a.fail(fmt.Sprintf("No se pudo analizar %s: %v", repo, err))
	a.printf("\n")
}

// backlogReportName is <project>-<repo>.json, or <folder>-<repo>.json without
// a project; a module filter adds the modules, so a filtered report never
// overwrites the full one.
func backlogReportName(projName, dir, repo string, modules []string) string {
	base := projName
	if base == "" {
		if abs, err := filepath.Abs(dir); err == nil {
			base = filepath.Base(abs)
		}
	}
	name := base + "-" + repo
	for _, m := range modules {
		name += "-" + backlog.ModuleKey(m)
	}
	return name + ".json"
}

func splitModules(v string) []string {
	var out []string
	for _, m := range strings.Split(v, ",") {
		if m = strings.TrimSpace(m); m != "" {
			out = append(out, m)
		}
	}
	return out
}

// saveBacklogReport writes UTF-8 JSON itself (PowerShell 5.1 `>` writes
// UTF-16). Without a path it goes next to config.yaml; the report holds
// internal paths and commit messages, so it is private (0600).
func saveBacklogReport(rep *backlog.Report, path, defaultName string) (string, error) {
	if path == "" {
		cfgPath, err := config.ConfigPath()
		if err != nil {
			return "", fmt.Errorf("no se pudo resolver la carpeta de configuración: %w", err)
		}
		path = filepath.Join(filepath.Dir(cfgPath), "backlog", defaultName)
	}
	path = filepath.Clean(path)

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("no se pudo crear la carpeta de %s: %w", path, err)
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return "", fmt.Errorf("no se pudo guardar el reporte en %s: %w", path, err)
	}
	return path, nil
}

// parseBacklogLimit: empty means the default, 0 means all.
func parseBacklogLimit(v string) (int, error) {
	if v == "" {
		return defaultBacklogLimit, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("--limit debe ser un número mayor o igual a 0, no %q", v)
	}
	return n, nil
}

var sinceShorthand = regexp.MustCompile(`^(\d+)([dwm])$`)

// normalizeSince accepts 90d, 12w and 3m on top of anything `git log --since`
// understands.
func normalizeSince(v string) string {
	m := sinceShorthand.FindStringSubmatch(v)
	if m == nil {
		return v
	}
	unit := map[string]string{"d": "days", "w": "weeks", "m": "months"}[m[2]]
	return m[1] + " " + unit + " ago"
}

// backlogOptions layers --since over the project's config; what stays empty
// takes the defaults of internal/backlog.
func backlogOptions(rc *config.BacklogRepoConfig, since string) backlog.Options {
	var o backlog.Options
	if rc != nil {
		o = backlog.Options{
			Since:       rc.Since,
			Extensions:  rc.Extensions,
			Exclude:     rc.Exclude,
			MaxLines:    rc.MaxLines,
			MinCommits:  rc.MinCommits,
			FixKeywords: rc.FixKeywords,
			ClassGlobs:  rc.ClassGlobs,
			TestsDir:    rc.TestsDir,
			TestGlobs:   rc.TestGlobs,
			Modules:     rc.Modules,
		}
	}
	if since != "" {
		o.Since = since
	}
	o.Since = normalizeSince(o.Since)
	return o
}

func severityColor(s backlog.Severity) string {
	switch s {
	case backlog.SeverityHigh:
		return red
	case backlog.SeverityMedium:
		return yellow
	default:
		return cyan
	}
}

func (a *App) printBacklogReport(rep *backlog.Report) {
	for _, s := range rep.Skipped {
		a.warn(fmt.Sprintf("Detector %s omitido: %s", s.Detector, s.Reason))
	}

	if len(rep.Findings) == 0 {
		if len(rep.ModuleFilter) > 0 {
			a.warn("Sin hallazgos en el módulo " + strings.Join(rep.ModuleFilter, ", ") + ". Módulos con hallazgos: " + formatModuleCounts(rep.Modules))
			return
		}
		a.ok("Sin hallazgos en el período " + rep.Since)
		return
	}
	a.printf("\n")

	points := 0
	for i, f := range rep.Findings {
		points += f.Points
		module := ""
		if f.Module != "" {
			module = a.style(bold, f.Module) + " · "
		}
		a.printf("%2d. %s %s  %s%s %s\n",
			i+1,
			a.style(severityColor(f.Severity)+bold, fmt.Sprintf("[%-5s]", f.Severity)),
			a.style(bold, fmt.Sprintf("%d pts", f.Points)),
			module,
			f.Title,
			a.style(cyan, "("+string(f.Kind)+" · "+f.ID+")"))
		// The report keeps slash-separated paths (stable IDs across systems);
		// the screen uses the local separator.
		for _, file := range f.Files {
			a.printf("      %s\n", filepath.FromSlash(file))
		}
		for _, e := range f.Evidence {
			a.printf("      · %s\n", e)
		}
	}

	a.printf("\n")
	if len(rep.Modules) > 0 {
		a.printf("Por módulo: %s\n", formatModuleCounts(rep.Modules))
	}
	a.printf("Mostrando %d de %d hallazgos · %s sugeridos en total\n", len(rep.Findings), rep.Total, a.style(bold, fmt.Sprintf("%d pts", points)))
	if len(rep.Findings) < rep.Total {
		a.printf("Usa %s para verlos todos.\n", a.style(bold, "--limit 0"))
	}
}

// formatModuleCounts renders "Pagos 24 · Facturacion 9".
func formatModuleCounts(counts []backlog.ModuleCount) string {
	parts := make([]string, 0, len(counts))
	for _, c := range counts {
		name := c.Module
		if name == "" {
			name = "(sin módulo)"
		}
		parts = append(parts, fmt.Sprintf("%s %d", name, c.Count))
	}
	return strings.Join(parts, " · ")
}

func backlogHelp(a *App) {
	a.title("draco backlog scan", "Detecta deuda técnica y la propone como tareas")
	a.printf("  Revisa el historial de git y el código de tus repos y lista lo que conviene\n")
	a.printf("  atacar: archivos que cambian demasiado, clases sin tests, TODO/FIXME pendientes\n")
	a.printf("  y archivos enormes. Cada hallazgo trae prioridad y puntos sugeridos.\n")
	a.printf("  Solo lee: no modifica nada, no usa internet (salvo --deps) y no necesita credenciales.\n\n")

	a.section("Uso:")
	a.printf("  draco backlog scan [flags]\n\n")

	a.section("Qué analiza (se puede ejecutar desde cualquier carpeta):")
	a.printf("  1. El proyecto de -p, o el default_project de la config\n")
	a.printf("  2. Todos sus repos con ruta: backend_path y frontend_path (con -r, solo uno)\n")
	a.printf("  3. Con --path, esa carpeta en lugar del proyecto\n")
	a.printf("  4. Sin proyecto ni --path, la carpeta actual\n\n")

	a.section("Flags:")
	a.printf("  -p, --project   Proyecto de la config                        (por defecto: default_project)\n")
	a.printf("  -r, --repo      backend o frontend: analiza solo ese repo    (por defecto: todos)\n")
	a.printf("      --path      Carpeta a analizar, aunque no sea un proyecto\n")
	a.printf("      --since     Historial a mirar: 30d, 12w, 3m o una fecha  (por defecto: 90d)\n")
	a.printf("  -m, --modulo    Solo esos módulos, separados por coma         (requiere modules)\n")
	a.printf("  -l, --limit     Hallazgos a mostrar por repo; 0 = todos      (por defecto: 10)\n")
	a.printf("  -o, --output    Guarda el reporte completo en JSON, un archivo por repo\n")
	a.printf("                  Sin ruta: %s<proyecto>-<repo>.json\n", defaultBacklogDirHint())
	a.printf("      --json      Imprime JSON en vez de la tabla (para otras herramientas)\n")
	a.printf("      --deps      Revisa paquetes vulnerables con composer audit (usa internet)\n")
	a.printf("  -h, --help      Esta ayuda\n\n")

	a.section("Qué detecta:")
	a.printf("  hotspot         Archivos con 10+ commits en el período. Alta prioridad si además son grandes\n")
	a.printf("  sin-test        Clases que no aparecen en ningún test (requiere class_globs, ver abajo)\n")
	a.printf("  marcador        TODO / FIXME / HACK pendientes en el código\n")
	a.printf("  archivo-grande  Archivos de más de 800 líneas que casi no cambian (prioridad baja)\n")
	a.printf("  dependencia     Paquetes con vulnerabilidades reportadas (solo con --deps)\n\n")

	a.section("Configuración (opcional):")
	a.printf("  Sin configurar nada funcionan todos los detectores menos sin-test, y los hallazgos\n")
	a.printf("  no llevan módulo. Agrega dentro de tu proyecto en %s:\n\n", configPathHint())
	a.printf("%s", a.style(cyan, "    backlog:\n"+
		"      backend:\n"+
		"        class_globs: [\"app/**/Services/**/*.php\"]   # archivos que deberían tener test\n"+
		"        modules: [\"app/Http/{modulo}/**\"]           # dónde está el módulo en la ruta\n"+
		"      frontend:\n"+
		"        class_globs: [\"src/app/**/*.service.ts\"]\n"+
		"        modules: [\"src/app/{modulo}/**\"]")+"\n\n")
	a.printf("  modules: gana el primero que coincide. Para casos sueltos o para unificar nombres\n")
	a.printf("  entre repos, Nombre=patrón: \"Facturacion=resources/views/reportesFac/**\".\n\n")
	a.printf("  También acepta: since, min_commits, max_lines, exclude, extensions, fix_keywords,\n")
	a.printf("  tests_dir y test_globs.\n\n")

	a.section("Ejemplos:")
	a.printf("  draco backlog scan                           # backend y frontend del proyecto por defecto\n")
	a.printf("  draco backlog scan -o                        # lo mismo y guarda el reporte completo\n")
	a.printf("  draco backlog scan -r frontend --since 30d   # solo el frontend, último mes\n")
	a.printf("  draco backlog scan -m pagos                  # solo el módulo Pagos, en los dos repos\n")
	a.printf("  draco backlog scan -l 0                      # todos los hallazgos, no solo el top 10\n")
	a.printf("  draco backlog scan --path %s    # un repo que no está en tus proyectos\n", exampleRepoPath())
	a.printf("  draco backlog scan -r backend -o %s   # un solo repo, a un archivo\n", exampleReportPath())
}

func defaultBacklogDirHint() string {
	if runtime.GOOS == "windows" {
		return `%USERPROFILE%\.config\draco\backlog\`
	}
	return "~/.config/draco/backlog/"
}

func exampleRepoPath() string {
	if runtime.GOOS == "windows" {
		return `C:\repos\pagos-api`
	}
	return "~/repos/pagos-api"
}

func exampleReportPath() string {
	if runtime.GOOS == "windows" {
		return `C:\reportes\backlog.json`
	}
	return "~/reportes/backlog.json"
}
