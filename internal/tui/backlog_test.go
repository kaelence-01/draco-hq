package tui

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kaelence-01/draco-hq/internal/backlog"
	"github.com/kaelence-01/draco-hq/internal/config"
)

var scanReports = map[string]*backlog.Report{
	"/repos/api": {Since: "30 days ago", Total: 2,
		Findings: []backlog.Finding{
			{ID: "h1", Kind: backlog.KindHotspot, Severity: backlog.SeverityHigh, Points: 8, Title: "Dividir CobroController", Module: "Pagos",
				Files: []string{"app/Http/Pagos/CobroController.php"}, Evidence: []string{"14 commits en el período"}},
			{ID: "m1", Kind: backlog.KindMarker, Severity: backlog.SeverityMedium, Points: 2, Title: "Resolver FIXME de Factura", Module: "Facturacion",
				Files: []string{"app/Http/Facturacion/Factura.php"}, Evidence: []string{"FIXME validar monto"}},
		},
		Modules: []backlog.ModuleCount{{Module: "Pagos", Count: 1}, {Module: "Facturacion", Count: 1}}},
	"/repos/web": {Since: "90 days ago", Total: 1,
		Findings: []backlog.Finding{{ID: "g1", Kind: backlog.KindLargeFile, Severity: backlog.SeverityLow, Points: 1, Title: "Partir lista-pagos.component.ts",
			Files: []string{"src/app/pagos/lista-pagos.component.ts"}}},
		Skipped: []backlog.Skipped{{Detector: "sin-test", Reason: "sin class_globs en la config del proyecto"}}},
}

type backlogEnv struct {
	m        *app
	reports  map[string]*backlog.Report
	home     string
	scanned  map[string]backlog.ScanOptions
	canceled chan struct{}
}

// newBacklogApp opens Backlog with a scanner that returns scanReports, fails
// for other folders and waits for "lento" until its step is canceled.
func newBacklogApp(t *testing.T) *backlogEnv {
	t.Helper()
	e := &backlogEnv{home: t.TempDir(), reports: maps.Clone(scanReports), scanned: map[string]backlog.ScanOptions{}, canceled: make(chan struct{})}
	t.Setenv("HOME", e.home)
	t.Setenv("USERPROFILE", e.home)
	cfg := &config.Config{
		DefaultProject: "pagos",
		Projects: map[string]*config.ProjectConfig{
			"pagos": {BackendPath: "/repos/api", FrontendPath: "/repos/web", Backlog: &config.BacklogConfig{
				Backend: &config.BacklogRepoConfig{Since: "30d", Modules: []string{"app/Http/{modulo}/**"}},
			}},
			"facturacion": {BackendPath: "/repos/facturacion-api"},
		},
	}
	e.m = newApp(Deps{
		Version:    "v9.9.9",
		LoadConfig: func() (*config.Config, error) { return cfg, nil },
		Scan: func(ctx context.Context, dir string, so backlog.ScanOptions) (*backlog.Report, error) {
			if dir == "lento" {
				<-ctx.Done()
				close(e.canceled)
				return nil, ctx.Err()
			}
			e.scanned[dir] = so
			rep, ok := e.reports[dir]
			if !ok {
				return nil, errors.New(dir + " no es un repositorio git")
			}
			cp := *rep
			cp.Findings = append([]backlog.Finding(nil), rep.Findings...)
			return &cp, nil
		},
	})
	e.m.Update(tea.WindowSizeMsg{Width: 110, Height: 34})
	for range 4 {
		send(e.m, keyDown)
	}
	send(e.m, keyEnter)
	return e
}

// fill answers the form: repoDowns moves in Repos (1 backend, 2 frontend).
func (e *backlogEnv) fill(repoDowns int, path, since, modules string) {
	send(e.m, keyEnter)
	for range repoDowns {
		send(e.m, keyDown)
	}
	send(e.m, keyEnter)
	for _, v := range []string{path, since, modules} {
		typeText(e.m, v)
		send(e.m, keyEnter)
	}
	send(e.m, keyEnter)
}

func TestBacklogScansEveryRepoOfTheProject(t *testing.T) {
	e := newBacklogApp(t)
	out := view(e.m)
	for _, want := range []string{"Inicio › Backlog técnico", "Proyecto", "Repos", "Carpeta"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en la primera página del formulario:\n%s", want, out)
		}
	}
	e.fill(0, "", "", "")
	out = view(e.m)
	for _, want := range []string{"› Analizar", "✓ backend · /repos/api · 2 hallazgos", "✓ frontend · /repos/web · 1 hallazgo", "$ draco backlog scan -p pagos -l 0", "enter ver hallazgos"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	if so := e.scanned["/repos/api"]; so.Since != "30 days ago" || len(so.Modules) != 1 || so.Dependencies {
		t.Errorf("el backend usa la config del proyecto: %+v", so)
	}
	if so := e.scanned["/repos/web"]; so.Since != "" || len(so.Modules) != 0 {
		t.Errorf("el frontend, sin config, deja los defaults: %+v", so)
	}

	send(e.m, keyEnter)
	out = view(e.m)
	for _, want := range []string{"› Hallazgos", "▸ backend (2)", "frontend (1)", "[alta ] 8 pts  Pagos · Dividir CobroController",
		"[media] 2 pts  Facturacion · Resolver FIXME", "2 hallazgos · 10 pts sugeridos · por módulo: Pagos 1 · Facturacion 1",
		filepath.FromSlash("app/Http/Pagos/CobroController.php"), "14 commits en el período", "hotspot · h1", "tab otro repo", "s guardar reporte"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	send(e.m, tea.KeyPressMsg{Code: tea.KeyTab})
	out = view(e.m)
	for _, want := range []string{"▸ frontend (1)", "[baja ] 1 pts  Partir lista-pagos", "⚠ Detector sin-test omitido: sin class_globs"} {
		if !strings.Contains(out, want) {
			t.Errorf("tab cambia al frontend; falta %q:\n%s", want, out)
		}
	}
}

func TestBacklogFilterAndDetail(t *testing.T) {
	e := newBacklogApp(t)
	e.fill(0, "", "", "")
	send(e.m, keyEnter)
	send(e.m, keyDown)
	if out := view(e.m); !strings.Contains(out, "marcador · m1") || !strings.Contains(out, "FIXME validar monto") {
		t.Errorf("el detalle sigue al cursor:\n%s", out)
	}
	send(e.m, press('/', "/"))
	typeText(e.m, "cobro")
	send(e.m, keyEnter)
	out := view(e.m)
	if strings.Contains(out, "Resolver FIXME") || !strings.Contains(out, "Dividir CobroController") || !strings.Contains(out, "filtro: cobro") {
		t.Errorf("el filtro busca también en los archivos:\n%s", out)
	}
	send(e.m, keyEsc)
	if out := view(e.m); !strings.Contains(out, "Resolver FIXME") || !strings.Contains(out, "› Hallazgos") {
		t.Errorf("esc quita el filtro sin salir:\n%s", out)
	}
}

func TestBacklogSavesFullReports(t *testing.T) {
	e := newBacklogApp(t)
	e.fill(0, "", "", "")
	send(e.m, keyEnter)
	send(e.m, press('s', "s"))
	dir := filepath.Join(e.home, ".config", "draco", "backlog")
	for repo, n := range map[string]int{"backend": 2, "frontend": 1} {
		data, err := os.ReadFile(filepath.Join(dir, "pagos-"+repo+".json"))
		if err != nil {
			t.Fatalf("se guarda donde lo deja -o: %v\n%s", err, view(e.m))
		}
		var rep backlog.Report
		if err := json.Unmarshal(data, &rep); err != nil || len(rep.Findings) != n {
			t.Errorf("%s: reporte completo en JSON: %v %d", repo, err, len(rep.Findings))
		}
	}
	if out := view(e.m); strings.Count(out, "✓ Reporte guardado en") != 2 {
		t.Errorf("dice dónde guardó:\n%s", out)
	}
}

func TestBacklogFailedRepoDoesNotStopTheOther(t *testing.T) {
	e := newBacklogApp(t)
	delete(e.reports, "/repos/web")
	e.fill(0, "", "", "")
	out := view(e.m)
	if !strings.Contains(out, "✗ frontend · /repos/web: /repos/web no es un repositorio git") || !strings.Contains(out, "✓ backend") {
		t.Fatalf("un repo que falla se marca y el otro sigue:\n%s", out)
	}
	send(e.m, keyEnter)
	if out := view(e.m); strings.Contains(out, "tab otro repo") || !strings.Contains(out, "Dividir CobroController") {
		t.Errorf("los hallazgos son solo del repo analizado:\n%s", out)
	}
}

func TestBacklogModulesNeedTheirConfig(t *testing.T) {
	e := newBacklogApp(t)
	e.fill(0, "", "7d", "pagos")
	out := view(e.m)
	if !strings.Contains(out, "✓ backend · /repos/api · 1 hallazgo") || !strings.Contains(out, "✗ frontend") || !strings.Contains(out, "backlog.frontend.modules al proyecto") {
		t.Errorf("el backend se filtra y el frontend, sin modules, lo explica:\n%s", out)
	}
	if !strings.Contains(out, "$ draco backlog scan -p pagos --since 7d -m pagos -l 0") || e.scanned["/repos/api"].Since != "7 days ago" {
		t.Errorf("el período del formulario le gana a la config: %+v\n%s", e.scanned["/repos/api"], out)
	}
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "1 hallazgo · 8 pts sugeridos · módulos pagos") {
		t.Errorf("el resumen dice el filtro:\n%s", out)
	}
}

func TestBacklogRepoWithoutPath(t *testing.T) {
	e := newBacklogApp(t)
	send(e.m, keyUp)
	e.fill(2, "", "", "")
	if out := view(e.m); !strings.Contains(out, "el proyecto facturacion no tiene frontend_path") || !strings.Contains(out, "Repos") {
		t.Errorf("un repo sin ruta se explica en el formulario:\n%s", out)
	}
}

func TestBacklogEscCancelsTheScan(t *testing.T) {
	e := newBacklogApp(t)
	e.fill(0, "lento", "", "")
	if out := view(e.m); !strings.Contains(out, "Analizando backend · lento") {
		t.Fatalf("una carpeta reemplaza al proyecto:\n%s", out)
	}
	send(e.m, keyEsc)
	<-e.canceled
	if out := view(e.m); !strings.Contains(out, "Carpeta") || strings.Contains(out, "Analizar") {
		t.Errorf("esc cancela y vuelve al formulario:\n%s", out)
	}
}

func TestBacklogFitsIn80Columns(t *testing.T) {
	e := newBacklogApp(t)
	e.fill(0, "", "", "")
	send(e.m, keyEnter)
	e.m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	out := view(e.m)
	for _, l := range strings.Split(out, "\n") {
		if w := ansi.StringWidth(l); w > 80 {
			t.Errorf("línea de %d columnas: %q", w, l)
		}
	}
	if !strings.Contains(out, "Dividir CobroController") || !strings.Contains(out, "hotspot · h1") {
		t.Errorf("en 80 columnas el detalle va debajo de la lista:\n%s", out)
	}
}

func TestBacklogFormFitsIn24Lines(t *testing.T) {
	e := newBacklogApp(t)
	e.m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	for range 3 {
		send(e.m, keyEnter)
	}
	out := view(e.m)
	for _, want := range []string{"Período", "Módulos", "composer audit", "Sí"} {
		if !strings.Contains(out, want) {
			t.Errorf("la segunda página cabe entera; falta %q:\n%s", want, out)
		}
	}
}
