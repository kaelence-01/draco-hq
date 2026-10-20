package cmd

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/backlog"
	"github.com/kaelence-01/draco-hq/internal/config"
)

func TestBacklogReportName(t *testing.T) {
	if got := backlogReportName("pagos", "/repos/api", "frontend", nil); got != "pagos-frontend.json" {
		t.Errorf("con proyecto: %q", got)
	}
	if got := backlogReportName("", filepath.Join("repos", "pagos-api"), "backend", nil); got != "pagos-api-backend.json" {
		t.Errorf("sin proyecto usa el nombre de la carpeta: %q", got)
	}
	if got := backlogReportName("pagos", "/repos/api", "backend", []string{"Facturacion", "notas-credito"}); got != "pagos-backend-facturacion-notascredito.json" {
		t.Errorf("un reporte filtrado nunca pisa el completo: %q", got)
	}
}

func TestSplitModules(t *testing.T) {
	if got := splitModules(" pagos, ,Facturacion "); !reflect.DeepEqual(got, []string{"pagos", "Facturacion"}) {
		t.Errorf("splitModules: %#v", got)
	}
	if got := splitModules(""); got != nil {
		t.Errorf("sin -m no hay filtro: %#v", got)
	}
}

func TestFormatModuleCounts(t *testing.T) {
	got := formatModuleCounts([]backlog.ModuleCount{{Module: "Pagos", Count: 24}, {Module: "", Count: 3}})
	if got != "Pagos 24 · (sin módulo) 3" {
		t.Errorf("formatModuleCounts: %q", got)
	}
}

func TestSaveBacklogReportWritesUTF8JSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "reporte.json")
	rep := &backlog.Report{Version: backlog.ReportVersion, Findings: []backlog.Finding{{Title: "Extraer módulo"}}}

	saved, err := saveBacklogReport(rep, path, "ignorado.json")
	if err != nil || saved != path {
		t.Fatalf("saveBacklogReport: %q %v", saved, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var back backlog.Report
	if err := json.Unmarshal(data, &back); err != nil || back.Findings[0].Title != "Extraer módulo" {
		t.Errorf("el archivo debe ser JSON UTF-8 legible: %v %#v", err, back)
	}
}

func TestSaveBacklogReportDefaultsNextToConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	saved, err := saveBacklogReport(&backlog.Report{}, "", "pagos-backend.json")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".config", "draco", "backlog", "pagos-backend.json"); saved != want {
		t.Errorf("sin ruta debe guardar junto al config.yaml: %q, want %q", saved, want)
	}
}

func TestNormalizeSince(t *testing.T) {
	for in, want := range map[string]string{
		"90d":         "90 days ago",
		"12w":         "12 weeks ago",
		"3m":          "3 months ago",
		"2026-07-01":  "2026-07-01",
		"90 days ago": "90 days ago",
		"":            "",
	} {
		if got := normalizeSince(in); got != want {
			t.Errorf("normalizeSince(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseBacklogLimit(t *testing.T) {
	if n, err := parseBacklogLimit(""); err != nil || n != defaultBacklogLimit {
		t.Errorf("vacío debe usar el default: %d %v", n, err)
	}
	if n, err := parseBacklogLimit("0"); err != nil || n != 0 {
		t.Errorf("0 significa todos: %d %v", n, err)
	}
	for _, bad := range []string{"-1", "diez"} {
		if _, err := parseBacklogLimit(bad); err == nil {
			t.Errorf("%q debe ser error", bad)
		}
	}
}

func TestResolveBacklogTargets(t *testing.T) {
	both := &config.ProjectConfig{BackendPath: "/repos/api", FrontendPath: "/repos/web"}
	onlyAPI := &config.ProjectConfig{BackendPath: "/repos/api"}

	cases := []struct {
		name, path, repo string
		proj             *config.ProjectConfig
		want             []backlogTarget
	}{
		{"sin flags analiza todos los repos del proyecto", "", "", both,
			[]backlogTarget{{"backend", "/repos/api"}, {"frontend", "/repos/web"}}},
		{"-r frontend analiza solo ese", "", "frontend", both, []backlogTarget{{"frontend", "/repos/web"}}},
		{"-r backend analiza solo ese", "", "backend", both, []backlogTarget{{"backend", "/repos/api"}}},
		{"proyecto con un solo repo", "", "", onlyAPI, []backlogTarget{{"backend", "/repos/api"}}},
		{"--path gana y es uno solo", "/otro", "", both, []backlogTarget{{"backend", "/otro"}}},
		{"--path con -r conserva la etiqueta", "/otro", "frontend", both, []backlogTarget{{"frontend", "/otro"}}},
		{"sin proyecto usa la carpeta actual", "", "", nil, []backlogTarget{{"backend", "."}}},
		{"proyecto sin rutas usa la carpeta actual", "", "", &config.ProjectConfig{}, []backlogTarget{{"backend", "."}}},
	}
	for _, c := range cases {
		got, err := resolveBacklogTargets(c.path, c.repo, c.proj, "pagos")
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %v %v, want %v", c.name, got, err, c.want)
		}
	}
}

func TestResolveBacklogTargetsErrors(t *testing.T) {
	onlyAPI := &config.ProjectConfig{BackendPath: "/repos/api"}
	if _, err := resolveBacklogTargets("", "frontend", onlyAPI, "pagos"); err == nil {
		t.Error("pedir un repo sin ruta configurada no cae a la carpeta actual: debe ser error")
	}
	if _, err := resolveBacklogTargets("", "movil", onlyAPI, "pagos"); err == nil {
		t.Error("un --repo desconocido debe ser error")
	}
}

func TestBacklogOutputRules(t *testing.T) {
	if err := validateBacklogOutput(filepath.Join("x", "reporte.json"), 2); err == nil {
		t.Error("con varios repos, -o con un archivo .json debe ser error")
	}
	if err := validateBacklogOutput(filepath.Join("x", "reporte.json"), 1); err != nil {
		t.Errorf("con un repo, -o acepta un archivo: %v", err)
	}

	folder := filepath.Join("x", "reportes")
	cases := []struct {
		flag    string
		targets int
		want    string
	}{
		{"", 2, ""},
		{"r.json", 1, "r.json"},
		{folder, 2, filepath.Join(folder, "pagos-frontend.json")},
	}
	for _, c := range cases {
		if got := backlogOutputPath(c.flag, c.targets, "pagos-frontend.json"); got != c.want {
			t.Errorf("backlogOutputPath(%q, %d) = %q, want %q", c.flag, c.targets, got, c.want)
		}
	}
}

func TestBacklogOptionsLayering(t *testing.T) {
	rc := &config.BacklogRepoConfig{Since: "30d", ClassGlobs: []string{"app/**/*.php"}, MaxLines: 500}

	fromConfig := backlogOptions(rc, "")
	if fromConfig.Since != "30 days ago" || fromConfig.MaxLines != 500 || !reflect.DeepEqual(fromConfig.ClassGlobs, rc.ClassGlobs) {
		t.Errorf("config del proyecto: %#v", fromConfig)
	}
	if got := backlogOptions(rc, "7d").Since; got != "7 days ago" {
		t.Errorf("--since debe ganarle a la config: %q", got)
	}
	if got := backlogOptions(nil, ""); !reflect.DeepEqual(got, backlog.Options{}) {
		t.Errorf("sin config ni flag todo queda vacío para que apliquen los defaults: %#v", got)
	}
}

// backlogEnv runs the command over cfg with a scanner that returns a fixed
// report per folder, or fails for folders not in reports.
func backlogEnv(t *testing.T, cfg *config.Config, reports map[string]*backlog.Report) *[]backlog.ScanOptions {
	t.Helper()
	prevLoad, prevScan := loadConfig, scanRepo
	t.Cleanup(func() { loadConfig, scanRepo = prevLoad, prevScan })
	loadConfig = func() (*config.Config, error) { return cfg, nil }
	var seen []backlog.ScanOptions
	scanRepo = func(dir string, so backlog.ScanOptions) (*backlog.Report, error) {
		seen = append(seen, so)
		rep, ok := reports[dir]
		if !ok {
			return nil, errors.New(dir + " no es un repositorio git")
		}
		cp := *rep
		cp.Findings = append([]backlog.Finding(nil), rep.Findings...)
		return &cp, nil
	}
	return &seen
}

func pagosConfig() *config.Config {
	return &config.Config{DefaultProject: "pagos", Projects: map[string]*config.ProjectConfig{
		"pagos": {BackendPath: "/repos/api", FrontendPath: "/repos/web", Backlog: &config.BacklogConfig{
			Backend: &config.BacklogRepoConfig{Since: "30d", Modules: []string{"app/{modulo}/**"}},
		}},
	}}
}

func findings(n int, module string) []backlog.Finding {
	out := make([]backlog.Finding, n)
	for i := range out {
		out[i] = backlog.Finding{ID: "h" + string(rune('a'+i)), Kind: backlog.KindHotspot, Severity: backlog.SeverityHigh, Points: 3,
			Title: "Dividir archivo", Module: module, Files: []string{"app/Pagos/Cobro.php"}}
	}
	return out
}

func TestBacklogScanBothReposWithLimit(t *testing.T) {
	seen := backlogEnv(t, pagosConfig(), map[string]*backlog.Report{
		"/repos/api": {Since: "30 days ago", Total: 12, Findings: findings(12, "Pagos"), Modules: []backlog.ModuleCount{{Module: "Pagos", Count: 12}}},
		"/repos/web": {Since: "90 days ago"},
	})
	code, out, errOut := run("backlog", "scan")
	if code != exitOK {
		t.Fatalf("código %d\n%s\n%s", code, out, errOut)
	}
	for _, want := range []string{"[1/2]", "(pagos backend)", "Mostrando 10 de 12 hallazgos", "30 pts", "--limit 0", "Por módulo: Pagos 12", "[2/2]", "Sin hallazgos en el período 90 days ago"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en la salida:\n%s", want, out)
		}
	}
	if len(*seen) != 2 || (*seen)[0].Since != "30 days ago" || (*seen)[1].Since != "" || (*seen)[0].Dependencies {
		t.Errorf("opciones por repo: %+v", *seen)
	}
}

func TestBacklogScanJSONKeepsStdoutClean(t *testing.T) {
	backlogEnv(t, pagosConfig(), map[string]*backlog.Report{
		"/repos/api": {Total: 1, Findings: findings(1, "")},
	})
	code, out, errOut := run("backlog", "scan", "--json", "--deps")
	if code != exitError || !strings.Contains(errOut, "No se pudo analizar frontend") {
		t.Fatalf("un repo que falla es error al final y va a stderr: %d\n%s", code, errOut)
	}
	var reps []backlog.Report
	if err := json.Unmarshal([]byte(out), &reps); err != nil || len(reps) != 1 {
		t.Errorf("stdout debe ser JSON válido con el repo que sí se analizó: %v\n%s", err, out)
	}
}

func TestBacklogScanSavesFullReport(t *testing.T) {
	backlogEnv(t, pagosConfig(), map[string]*backlog.Report{
		"/repos/api": {Total: 12, Findings: findings(12, "")},
	})
	path := filepath.Join(t.TempDir(), "reporte.json")
	code, out, errOut := run("backlog", "scan", "-r", "backend", "-l", "2", "-o", path)
	if code != exitOK || !strings.Contains(out, "Reporte completo guardado en "+path) {
		t.Fatalf("código %d\n%s\n%s", code, out, errOut)
	}
	data, _ := os.ReadFile(path)
	var rep backlog.Report
	if err := json.Unmarshal(data, &rep); err != nil || len(rep.Findings) != 12 {
		t.Errorf("el archivo guarda todos los hallazgos, --limit solo recorta la pantalla: %v %d", err, len(rep.Findings))
	}
}

func TestBacklogScanOutputWithoutValue(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	backlogEnv(t, pagosConfig(), map[string]*backlog.Report{
		"/repos/web": {Total: 1, Findings: findings(1, "")},
	})
	code, out, errOut := run("backlog", "scan", "-o", "-r", "frontend")
	if code != exitOK {
		t.Fatalf("-o sin ruta no consume el flag siguiente: %d\n%s\n%s", code, out, errOut)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "draco", "backlog", "pagos-frontend.json")); err != nil {
		t.Errorf("sin ruta guarda junto a la config: %v", err)
	}
}

func TestBacklogScanModuleFilter(t *testing.T) {
	backlogEnv(t, pagosConfig(), map[string]*backlog.Report{
		"/repos/api": {Total: 2, Findings: append(findings(1, "Pagos"), findings(1, "Facturacion")...),
			Modules: []backlog.ModuleCount{{Module: "Pagos", Count: 1}, {Module: "Facturacion", Count: 1}}},
		"/repos/web": {Total: 1, Findings: findings(1, "")},
	})
	code, out, _ := run("backlog", "scan", "-m", "facturacion")
	if code != exitError || !strings.Contains(out, "Mostrando 1 de 1") || !strings.Contains(out, "-m necesita backlog.frontend.modules") {
		t.Errorf("filtra el backend y explica el frontend sin módulos: %d\n%s", code, out)
	}
}

func TestBacklogUsageErrors(t *testing.T) {
	backlogEnv(t, pagosConfig(), nil)
	cases := map[string][]string{
		"subcomando desconocido": {"backlog", "limpiar"},
		"--limit debe ser":       {"backlog", "scan", "-l", "diez"},
		"recibe una carpeta":     {"backlog", "scan", "-o", "todo.json"},
		"flag desconocido":       {"backlog", "scan", "--issue", "OPS-7"},
	}
	for want, args := range cases {
		if code, _, errOut := run(args...); code != exitUsage || !strings.Contains(errOut, want) {
			t.Errorf("%v: código %d, esperaba %q en %q", args, code, want, errOut)
		}
	}
	if code, out, _ := run("backlog"); code != exitOK || !strings.Contains(out, "draco backlog scan") {
		t.Errorf("backlog solo muestra la ayuda: %d\n%s", code, out)
	}
}
