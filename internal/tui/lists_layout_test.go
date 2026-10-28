package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kaelence-01/draco-hq/internal/config"
)

func resizeApp(m *app, width, height int) { m.Update(tea.WindowSizeMsg{Width: width, Height: height}) }

// footerOf is the last line of the screen: the keys.
func footerOf(out string) string {
	lines := strings.Split(out, "\n")
	return lines[len(lines)-1]
}

func TestPanelsLayout(t *testing.T) {
	var leftWidth, rightWidth int
	var gotLines [2]int
	out := ansi.Strip(panels(100, 10, 6, 4,
		func(w, lines int) string { leftWidth, gotLines[0] = w, lines; return "izquierda" },
		func(w, lines int) string { rightWidth, gotLines[1] = w, lines; return "derecha" }))
	lines := strings.Split(out, "\n")
	assertFits(t, lines, 100, 10)
	if leftWidth != 53 || rightWidth != 35 || gotLines[0] != 8 || gotLines[1] != 8 {
		t.Errorf("cada lado recibe el sitio dentro de su caja (6/10 y 4/10, menos borde y margen): %d y %d columnas, %v líneas", leftWidth, rightWidth, gotLines)
	}
	if !strings.Contains(lines[1], "izquierda") || !strings.Contains(lines[1], "derecha") || !strings.HasPrefix(lines[len(lines)-1], " ╰") {
		t.Errorf("dos cajas que llegan abajo:\n%s", out)
	}
}

func TestProjectsFillAWideTerminal(t *testing.T) {
	m, _ := newProjectsApp(t)
	resizeApp(m, 160, 40)
	lines := strings.Split(view(m), "\n")
	assertFits(t, lines, 160, 40)
	out := strings.Join(lines, "\n")
	for _, want := range []string{"/ filtrar", "facturacion", "▸ pagos", "activo", "por defecto", "backend_path", "~/code/pagos-api", "deploy_checklist"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	if at := col(lineWith(t, lines, "backend_path"), "backend_path"); at < 60 {
		t.Errorf("el detalle va a la derecha de la lista (4/10 y 6/10): columna %d", at)
	}
	if !strings.Contains(bottomEdge(lines), "╰") {
		t.Errorf("las cajas llegan hasta el pie:\n%s", out)
	}
	send(m, press('/', "/"))
	typeText(m, "fa")
	if out := view(m); !strings.Contains(out, "/ fa") || strings.Contains(out, "▸ pagos") {
		t.Errorf("el filtro se ve donde se escribe:\n%s", out)
	}
}

func TestProjectsKeepTheirListAndDetailInNarrowTerminals(t *testing.T) {
	m, _ := newProjectsApp(t)
	for _, size := range [][2]int{{80, 24}, {90, 24}, {60, 24}} {
		resizeApp(m, size[0], size[1])
		lines := strings.Split(view(m), "\n")
		assertFits(t, lines, size[0], size[1])
		if out := strings.Join(lines, "\n"); !strings.Contains(out, "/ filtrar") || !strings.Contains(out, "backend_path") || !strings.Contains(out, "▸ pagos") {
			t.Errorf("%dx%d: la lista con su filtro y el detalle:\n%s", size[0], size[1], out)
		}
	}
	resizeApp(m, 80, 24)
	if short := footerOf(view(m)); !strings.Contains(short, "? más") || strings.Contains(short, "…") {
		t.Errorf("a 80 columnas el pie cabe, con ? más: %q", short)
	}
}

func TestProfilesFillAWideTerminalAndFitNarrow(t *testing.T) {
	e := newSettingsApp(t, savedConfig())
	e.toProfiles(t)
	resizeApp(e.m, 160, 40)
	lines := strings.Split(view(e.m), "\n")
	assertFits(t, lines, 160, 40)
	out := strings.Join(lines, "\n")
	for _, want := range []string{"/ filtrar", "▸ equipo", "por defecto", "externo", "tracker", "destino", "confluence"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	if at := col(lineWith(t, lines, "tracker"), "tracker"); at < 60 || !strings.Contains(bottomEdge(lines), "╰") {
		t.Errorf("el detalle a la derecha y las cajas hasta el pie: columna %d\n%s", at, out)
	}
	resizeApp(e.m, 80, 24)
	lines = strings.Split(view(e.m), "\n")
	assertFits(t, lines, 80, 24)
	if short := footerOf(strings.Join(lines, "\n")); !strings.Contains(short, "? más") || strings.Contains(short, "…") {
		t.Errorf("a 80 columnas el pie cabe, con ? más: %q", short)
	}
}

func TestSettingsShowWhatEachSectionHoldsBesideTheList(t *testing.T) {
	e := newSettingsApp(t, savedConfig())
	resizeApp(e.m, 160, 40)
	lines := strings.Split(view(e.m), "\n")
	assertFits(t, lines, 160, 40)
	if at := col(lineWith(t, lines, "base_url"), "base_url"); at < 60 || !strings.Contains(bottomEdge(lines), "╰") {
		t.Errorf("el detalle a la derecha y las cajas hasta el pie: columna %d", at)
	}

	sections := []struct {
		down int
		want []string
		not  []string
	}{
		{0, []string{"email", "tu@example.test", "base_url", "https://empresa.example.test", "token", "definido, no se muestra"}, []string{"token-del-archivo"}},
		{1, []string{"pagos", "por defecto", "enter abre Proyectos"}, nil},
		{2, []string{"equipo", "jira → confluence (space DEV)", "externo", "jira → markdown", "enter abre Perfiles"}, nil},
		{3, []string{"qa_email", "qa_jira", "qa_report", "Sin qa_report: Líder Técnico, PMO y QA saldrán vacíos"}, nil},
		{4, []string{"ansi", "alto-contraste", "mono", "Colores brillantes y negrita", "con vista previa en vivo"}, nil},
	}
	for _, s := range sections {
		e := newSettingsApp(t, savedConfig())
		resizeApp(e.m, 160, 40)
		for range s.down {
			send(e.m, keyDown)
		}
		out := view(e.m)
		for _, want := range s.want {
			if !strings.Contains(out, want) {
				t.Errorf("sección %d: falta %q:\n%s", s.down, want, out)
			}
		}
		for _, not := range s.not {
			if strings.Contains(out, not) {
				t.Errorf("sección %d: nunca se muestra %q:\n%s", s.down, not, out)
			}
		}
	}
}

func TestSettingsQADetailWithEverythingSet(t *testing.T) {
	cfg := savedConfig()
	cfg.QAEmail = "qa@example.test"
	cfg.QAReport = &config.QAReportConfig{LiderTecnico: "Ana", QA: "Luis"}
	e := newSettingsApp(t, cfg)
	resizeApp(e.m, 160, 40)
	for range 3 {
		send(e.m, keyDown)
	}
	out := view(e.m)
	if !strings.Contains(out, "qa@example.test") || !strings.Contains(out, "Líder Técnico: Ana · QA: Luis") || strings.Contains(out, "PMO:") || strings.Contains(out, "Sin qa_report") {
		t.Errorf("el detalle de QA lleva los nombres que hay y no avisa de lo que no falta:\n%s", out)
	}
}

func TestSettingsStayAListInNarrowTerminals(t *testing.T) {
	e := newSettingsApp(t, savedConfig())
	resizeApp(e.m, 80, 24)
	lines := strings.Split(view(e.m), "\n")
	assertFits(t, lines, 80, 24)
	out := strings.Join(lines, "\n")
	if !strings.Contains(out, "▸ Conexión con Atlassian") || !strings.Contains(out, "✓ tu@example.test en https://empresa.example.test") || strings.Contains(out, "base_url") || strings.Contains(out, "╭") {
		t.Errorf("en 80 columnas, la lista de siempre:\n%s", out)
	}
}

func TestFindingsFillAWideTerminalWithTheFilterLine(t *testing.T) {
	e := newBacklogApp(t)
	e.fill(0, "", "", "")
	send(e.m, keyEnter)
	resizeApp(e.m, 160, 40)
	lines := strings.Split(view(e.m), "\n")
	assertFits(t, lines, 160, 40)
	out := strings.Join(lines, "\n")
	for _, want := range []string{"/ filtrar", "▸ [alta ] 8 pts", "Dividir CobroController", "hotspot · h1", "2 hallazgos"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	if at := col(lineWith(t, lines, "hotspot · h1"), "hotspot · h1"); at < 90 || !strings.Contains(bottomEdge(lines), "╰") {
		t.Errorf("el detalle a la derecha (6/10 y 4/10) y las cajas hasta el pie: columna %d\n%s", at, out)
	}
	send(e.m, keyDown)
	if out := view(e.m); !strings.Contains(out, "▸ [media] 2 pts") || !strings.Contains(out, "FIXME") {
		t.Errorf("el detalle sigue al cursor:\n%s", out)
	}

	resizeApp(e.m, 80, 24)
	lines = strings.Split(view(e.m), "\n")
	assertFits(t, lines, 80, 24)
	if out := strings.Join(lines, "\n"); !strings.Contains(out, "/ filtrar") || !strings.Contains(out, "hotspot") && !strings.Contains(out, "marcador") {
		t.Errorf("en 80 columnas, la lista con el filtro y el detalle debajo:\n%s", out)
	}
	if short := footerOf(strings.Join(lines, "\n")); !strings.Contains(short, "? más") || strings.Contains(short, "…") {
		t.Errorf("el pie cabe, con ? más: %q", short)
	}
}

func TestExportListHasADetailBesideIt(t *testing.T) {
	e := newExportApp(t, deployDoc, "Consolidado de Pruebas QA - Pagos - Sprint 12")
	e.search(t, "ops-7")
	resizeApp(e.m, 160, 40)
	lines := strings.Split(view(e.m), "\n")
	assertFits(t, lines, 160, 40)
	out := strings.Join(lines, "\n")
	for _, want := range []string{"2 documentos mencionan OPS-7", "/ filtrar", "▸ Consolidado de Pruebas QA", deployDoc, "enter lee el documento y propone"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q (el más reciente va primero):\n%s", want, out)
		}
	}
	if at := col(lineWith(t, lines, "enter lee el documento"), "enter lee el documento"); at < 90 || !strings.Contains(bottomEdge(lines), "╰") {
		t.Errorf("el detalle a la derecha y las cajas hasta el pie: columna %d\n%s", at, out)
	}
	if strings.Count(out, "Consolidado de Pruebas QA - Pagos - Sprint 12") < 2 {
		t.Errorf("el detalle es el del documento elegido, y el título está en la lista y en el detalle:\n%s", out)
	}

	send(e.m, keyDown)
	out = view(e.m)
	if !strings.Contains(out, "▸ "+deployDoc) || strings.Count(out, deployDoc) < 2 {
		t.Errorf("↓ baja al otro documento y el detalle lo sigue:\n%s", out)
	}

	send(e.m, press('/', "/"))
	typeText(e.m, "zzz")
	if out := view(e.m); !strings.Contains(out, "Ningún documento coincide") || !strings.Contains(out, "Ningún documento coincide con el filtro.") {
		t.Errorf("sin coincidencias, la lista y el detalle lo dicen:\n%s", out)
	}
}

func TestExportListInNarrowTerminalsKeepsFilterAndKeys(t *testing.T) {
	e := newExportApp(t, deployDoc, "Consolidado de Pruebas QA - Pagos - Sprint 12")
	e.search(t, "ops-7")
	resizeApp(e.m, 80, 24)
	lines := strings.Split(view(e.m), "\n")
	assertFits(t, lines, 80, 24)
	out := strings.Join(lines, "\n")
	if !strings.Contains(out, "/ filtrar") || !strings.Contains(out, deployDoc) || strings.Contains(out, "╭") {
		t.Errorf("en 80 columnas, la lista de siempre con el filtro visible:\n%s", out)
	}
	if short := footerOf(out); !strings.Contains(short, "↑↓ elegir") || !strings.Contains(short, "enter elegir") || !strings.Contains(short, "? más") {
		t.Errorf("pie compacto: %q", short)
	}
	send(e.m, keyHelp)
	if full := squash(view(e.m)); !strings.Contains(full, "/ filtrar") || !strings.Contains(full, "↑/k subir") {
		t.Errorf("la ayuda completa lleva el filtro y las demás:\n%s", full)
	}
}
