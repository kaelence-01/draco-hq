package backlog

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/config"
)

func TestTargets(t *testing.T) {
	both := &config.ProjectConfig{BackendPath: "/repos/api", FrontendPath: "/repos/web"}
	onlyAPI := &config.ProjectConfig{BackendPath: "/repos/api"}

	cases := []struct {
		name, path, repo string
		proj             *config.ProjectConfig
		want             []Target
	}{
		{"sin repo ni carpeta analiza todos los repos del proyecto", "", "", both,
			[]Target{{"backend", "/repos/api"}, {"frontend", "/repos/web"}}},
		{"frontend analiza solo ese", "", "frontend", both, []Target{{"frontend", "/repos/web"}}},
		{"backend analiza solo ese", "", "backend", both, []Target{{"backend", "/repos/api"}}},
		{"proyecto con un solo repo", "", "", onlyAPI, []Target{{"backend", "/repos/api"}}},
		{"la carpeta gana y es una sola", "/otro", "", both, []Target{{"backend", "/otro"}}},
		{"la carpeta con repo conserva la etiqueta", "/otro", "frontend", both, []Target{{"frontend", "/otro"}}},
		{"sin proyecto usa la carpeta actual", "", "", nil, []Target{{"backend", "."}}},
		{"proyecto sin rutas usa la carpeta actual", "", "", &config.ProjectConfig{}, []Target{{"backend", "."}}},
	}
	for _, c := range cases {
		got, err := Targets(c.proj, c.repo, c.path)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %v %v, want %v", c.name, got, err, c.want)
		}
	}
}

func TestTargetsErrors(t *testing.T) {
	onlyAPI := &config.ProjectConfig{BackendPath: "/repos/api"}
	if _, err := Targets(onlyAPI, "frontend", ""); !errors.Is(err, ErrNoRepoPath) {
		t.Errorf("pedir un repo sin ruta no cae a la carpeta actual: %v", err)
	}
	if _, err := Targets(onlyAPI, "movil", ""); err == nil {
		t.Error("un repo desconocido debe ser error")
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

func TestOptionsForLayering(t *testing.T) {
	rc := &config.BacklogRepoConfig{Since: "30d", ClassGlobs: []string{"app/**/*.php"}, MaxLines: 500}

	fromConfig := OptionsFor(rc, "")
	if fromConfig.Since != "30 days ago" || fromConfig.MaxLines != 500 || !reflect.DeepEqual(fromConfig.ClassGlobs, rc.ClassGlobs) {
		t.Errorf("config del proyecto: %#v", fromConfig)
	}
	if got := OptionsFor(rc, "7d").Since; got != "7 days ago" {
		t.Errorf("el período pedido le gana a la config: %q", got)
	}
	if got := OptionsFor(nil, ""); !reflect.DeepEqual(got, Options{}) {
		t.Errorf("sin config ni período todo queda vacío para que apliquen los defaults: %#v", got)
	}
}

func TestSplitModules(t *testing.T) {
	if got := SplitModules(" pagos, ,Facturacion "); !reflect.DeepEqual(got, []string{"pagos", "Facturacion"}) {
		t.Errorf("SplitModules: %#v", got)
	}
	if got := SplitModules(""); got != nil {
		t.Errorf("vacío no tiene módulos: %#v", got)
	}
}

func TestApplyModules(t *testing.T) {
	plain := &Report{Findings: []Finding{{Title: "a"}}}
	if err := ApplyModules(plain, nil); err != nil || len(plain.Findings) != 1 {
		t.Errorf("sin filtro no cambia nada: %v", err)
	}
	if err := ApplyModules(plain, []string{"pagos"}); !errors.Is(err, ErrNoModules) {
		t.Errorf("filtrar un repo sin modules es ErrNoModules: %v", err)
	}
	rep := &Report{Findings: []Finding{{Module: "Pagos"}, {Module: "Facturacion"}}, Modules: []ModuleCount{{"Pagos", 1}, {"Facturacion", 1}}}
	if err := ApplyModules(rep, []string{"pagos"}); err != nil || len(rep.Findings) != 1 || rep.Findings[0].Module != "Pagos" {
		t.Errorf("con modules filtra: %v %+v", err, rep.Findings)
	}
}

func TestReportName(t *testing.T) {
	if got := ReportName("pagos", "/repos/api", "frontend", nil); got != "pagos-frontend.json" {
		t.Errorf("con proyecto: %q", got)
	}
	if got := ReportName("", filepath.Join("repos", "pagos-api"), "backend", nil); got != "pagos-api-backend.json" {
		t.Errorf("sin proyecto, la carpeta: %q", got)
	}
	if got := ReportName("pagos", "/repos/api", "backend", []string{"Facturacion", "notas-credito"}); got != "pagos-backend-facturacion-notascredito.json" {
		t.Errorf("con módulos: %q", got)
	}
}

func TestSaveReportWritesUTF8JSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "reporte.json")
	rep := &Report{Version: ReportVersion, Findings: []Finding{{Title: "Extraer módulo"}}}

	saved, err := SaveReport(rep, path, "ignorado.json")
	if err != nil || saved != path {
		t.Fatalf("SaveReport: %q %v", saved, err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var back Report
	if err := json.Unmarshal(data, &back); err != nil || back.Findings[0].Title != "Extraer módulo" {
		t.Errorf("el archivo debe ser JSON UTF-8 legible: %v %#v", err, back)
	}
}

func TestSaveReportDefaultsNextToConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	saved, err := SaveReport(&Report{}, "", "pagos-backend.json")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".config", "draco", "backlog", "pagos-backend.json"); saved != want {
		t.Errorf("sin ruta debe guardar junto al config.yaml: %q, want %q", saved, want)
	}
}

func TestModuleSummary(t *testing.T) {
	if got := ModuleSummary([]ModuleCount{{Module: "Pagos", Count: 24}, {Module: "", Count: 3}}); got != "Pagos 24 · (sin módulo) 3" {
		t.Errorf("ModuleSummary: %q", got)
	}
}
