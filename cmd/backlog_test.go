package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/backlog"
	"github.com/kaelence-01/draco-hq/internal/config"
)

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

func TestResolveBacklogTargetsErrors(t *testing.T) {
	onlyAPI := &config.ProjectConfig{BackendPath: "/repos/api"}
	if _, err := resolveBacklogTargets("", "frontend", onlyAPI, "pagos"); err == nil || !strings.Contains(err.Error(), "el proyecto pagos no tiene frontend_path configurado. Usa --path") {
		t.Errorf("un repo sin ruta se explica con los flags: %v", err)
	}
	if _, err := resolveBacklogTargets("", "movil", onlyAPI, "pagos"); err == nil || !strings.Contains(err.Error(), "--repo debe ser backend o frontend") {
		t.Errorf("un --repo desconocido se explica con el flag: %v", err)
	}
	if got, err := resolveBacklogTargets("/otro", "", onlyAPI, "pagos"); err != nil || len(got) != 1 || got[0].Dir != "/otro" {
		t.Errorf("--path analiza esa carpeta: %v %v", got, err)
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

// backlogEnv runs the command over cfg with a scanner that returns a fixed
// report per folder, or fails for folders not in reports.
func backlogEnv(t *testing.T, cfg *config.Config, reports map[string]*backlog.Report) *[]backlog.ScanOptions {
	t.Helper()
	prevLoad, prevScan := loadConfig, scanRepo
	t.Cleanup(func() { loadConfig, scanRepo = prevLoad, prevScan })
	loadConfig = func() (*config.Config, error) { return cfg, nil }
	var seen []backlog.ScanOptions
	scanRepo = func(_ context.Context, dir string, so backlog.ScanOptions) (*backlog.Report, error) {
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
