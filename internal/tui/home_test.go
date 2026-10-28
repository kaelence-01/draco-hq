package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kaelence-01/draco-hq/internal/config"
)

// fullProject is a project with everything the active-project box shows.
func fullProject(cfg *config.Config) *config.Config {
	cfg.Projects["pagos"] = &config.ProjectConfig{
		BackendPath: "~/code/pagos-api", FrontendPath: "~/code/pagos-web",
		VCSHost: "github.com", VCSOrg: "mi-org", ConfluenceSpaceKey: "DEV",
	}
	return cfg
}

func lineWith(t *testing.T, lines []string, want string) string {
	t.Helper()
	for _, l := range lines {
		if strings.Contains(l, want) {
			return l
		}
	}
	t.Fatalf("ninguna línea tiene %q:\n%s", want, strings.Join(lines, "\n"))
	return ""
}

func assertFits(t *testing.T, lines []string, width, height int) {
	t.Helper()
	if len(lines) != height {
		t.Errorf("la pantalla ocupa %d líneas, se esperaban %d", len(lines), height)
	}
	for _, l := range lines {
		if w := ansi.StringWidth(l); w > width {
			t.Errorf("una línea mide %d y la terminal %d: %q", w, width, l)
		}
	}
}

func TestHomeFillsAWideTerminal(t *testing.T) {
	m := newTestApp(t, fullProject(testConfig()), nil, 160, 40)
	lines := strings.Split(view(m), "\n")
	assertFits(t, lines, 160, 40)

	if at := strings.Index(lineWith(t, lines, "▸ Generar documento de despliegue"), "▸"); at > 3 {
		t.Errorf("el menú empieza arriba a la izquierda, no centrado: columna %d", at)
	}
	if at := strings.Index(lineWith(t, lines, "│ Generar documento de despliegue"), "│"); at < 60 {
		t.Errorf("el panel va a la derecha del menú: columna %d", at)
	}
	out := strings.Join(lines, "\n")
	for _, want := range []string{
		"Desde la tarea y tus commits, conservando lo escrito", "Conexión, proyectos, perfiles, QA y tema",
		"1 Tarea  ›  2 Commits  ›  3 Vista previa  ›  4 Ubicación  ›  5 Publicar", "En modo comando, para scripts y CI:",
		"PROYECTO ACTIVO", "pagos · por defecto", "backend  ~/code/pagos-api", "frontend ~/code/pagos-web", "vcs      github.com / mi-org", "space    DEV",
		"TU CONFIGURACIÓN", "✓ Conexión  tu@example.test", "✓ Proyectos 1", "✓ Perfil    equipo", "· QA        sin configurar", "✓ Tema      ansi",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	if !strings.Contains(lineWith(t, lines, "PROYECTO ACTIVO"), "TU CONFIGURACIÓN") {
		t.Errorf("las dos cajas van lado a lado:\n%s", out)
	}
}

func TestHomeFitsTheSmallestTerminal(t *testing.T) {
	m := newTestApp(t, testConfig(), nil, 80, 24)
	lines := strings.Split(view(m), "\n")
	assertFits(t, lines, 80, 24)
	out := strings.Join(lines, "\n")
	for _, want := range []string{"1 ▸ Generar documento de despliegue", "2   Proyectos", "6   Configuración", "Lee la tarea", "$ draco g -i OPS-7 -b a1b2c3d", "1-6 abrir directo", "? más"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	for _, not := range []string{"╭", "PROYECTO ACTIVO", "Desde la tarea y tus commits"} {
		if strings.Contains(out, not) {
			t.Errorf("sin ancho para el panel ni las descripciones, sobra %q:\n%s", not, out)
		}
	}
}

func TestHomeKeepsTheCommandWhenNoticesTakeTheRoom(t *testing.T) {
	m := newTestApp(t, &config.Config{}, nil, 80, 24)
	m.st.newVersion = "v9.9.10"
	lines := strings.Split(view(m), "\n")
	assertFits(t, lines, 80, 24)
	out := strings.Join(lines, "\n")
	for _, want := range []string{"Falta configurar la conexión con Atlassian", "draco init)", "Nueva versión v9.9.10", "Lee la tarea", "$ draco g -i OPS-7 -b a1b2c3d", "? más"} {
		if !strings.Contains(out, want) {
			t.Errorf("con dos avisos, la principal sigue mostrando el texto y el comando de la opción: falta %q\n%s", want, out)
		}
	}
}

func TestHomeDescriptionsGiveWayBeforeTheMenuIsCut(t *testing.T) {
	m := newTestApp(t, testConfig(), nil, 100, 18)
	lines := strings.Split(view(m), "\n")
	assertFits(t, lines, 100, 18)
	out := strings.Join(lines, "\n")
	for _, label := range []string{"Generar documento de despliegue", "Proyectos", "Consolidado QA", "Exportar página", "Backlog técnico", "Configuración"} {
		if !strings.Contains(out, label) {
			t.Errorf("con poca altura el menú pierde las descripciones, no opciones: falta %q\n%s", label, out)
		}
	}
	if strings.Contains(out, "Desde la tarea") {
		t.Errorf("sin altura para las descripciones, no se dibujan cortadas:\n%s", out)
	}
}

func TestHomeContextNeedsRoom(t *testing.T) {
	for _, size := range [][2]int{{100, 40}, {120, 22}} {
		out := view(newTestApp(t, testConfig(), nil, size[0], size[1]))
		if strings.Contains(out, "PROYECTO ACTIVO") || strings.Contains(out, "TU CONFIGURACIÓN") {
			t.Errorf("%dx%d: sin ancho o alto para las cajas, no se dibujan cortadas:\n%s", size[0], size[1], out)
		}
		if !strings.Contains(out, "$ draco g -i OPS-7 -b a1b2c3d") {
			t.Errorf("%dx%d: el panel sigue completo:\n%s", size[0], size[1], out)
		}
	}
}

func TestHomeWithoutActiveProject(t *testing.T) {
	m := newTestApp(t, &config.Config{AtlassianEmail: "tu@example.test", AtlassianToken: "x", BaseURL: "https://example.test"}, nil, 160, 40)
	out := view(m)
	for _, want := range []string{"Ninguno. Crea uno en Proyectos.", "⚠ Proyectos ninguno"} {
		if !strings.Contains(out, want) {
			t.Errorf("sin proyecto lo dice y lo marca: falta %q\n%s", want, out)
		}
	}
}

func TestHomeNumberKeysOpenTheFunction(t *testing.T) {
	m := newApp(Deps{
		Version:    "v9.9.9",
		LoadConfig: func() (*config.Config, error) { return testConfig(), nil },
		LoadFile:   func() (*config.Config, error) { return testConfig(), nil },
	})
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	for key, title := range map[string]string{"1": "Generar documento", "2": "Proyectos", "3": "Consolidado QA", "4": "Exportar página", "5": "Backlog técnico", "6": "Configuración"} {
		send(m, press(rune(key[0]), key))
		if len(m.stack) != 2 || m.top().title() != title {
			t.Errorf("%s abre %q, hay %d pantallas y arriba %q", key, title, len(m.stack), m.top().title())
		}
		send(m, keyEsc)
	}
	for _, k := range []string{"7", "9", "0"} {
		send(m, press(rune(k[0]), k))
		if len(m.stack) != 1 {
			t.Errorf("%s no es una opción y no abre nada", k)
		}
	}

	withFutureItem(m)
	send(m, press('7', "7"))
	out := view(m)
	if len(m.stack) != 1 || !strings.Contains(out, "«Función futura» todavía no está en el modo visual: usa draco futura") || !strings.Contains(out, "▸ Función futura") {
		t.Errorf("7 elige la función sin pantalla y dice dónde está:\n%s", out)
	}
}

func TestHomeNumbersAreTypedInTheBar(t *testing.T) {
	m := newBarApp(t)
	send(m, press('/', "/"))
	typeText(m, "g 2")
	if h := m.stack[0].(*home); len(m.stack) != 1 || h.cursor != 0 || h.bar == nil {
		t.Errorf("con la barra abierta, un número se escribe y no abre una función: pila %d, cursor %d", len(m.stack), h.cursor)
	}
}

func TestConfigChecks(t *testing.T) {
	report := &config.QAReportConfig{PMO: "Ana"}
	qa := &config.QAJiraConfig{
		Project: "OPS", Statuses: []string{"10001"},
		Fields: config.QAJiraFields{CodingErrors: "customfield_1", DevReturns: "customfield_2", DevStatus: "customfield_3"},
	}
	cases := []struct {
		name    string
		cfg     config.Config
		profile string
		label   string
		want    configLine
	}{
		{"sin credenciales", config.Config{}, "", "Conexión", configLine{lineWarn, "Conexión", "faltan el email, el token o la URL"}},
		{"con credenciales", config.Config{AtlassianEmail: "tu@example.test", AtlassianToken: "x", BaseURL: "https://example.test"}, "", "Conexión", configLine{lineOK, "Conexión", "tu@example.test"}},
		{"sin proyectos", config.Config{}, "", "Proyectos", configLine{lineWarn, "Proyectos", "ninguno"}},
		{"dos proyectos", config.Config{Projects: map[string]*config.ProjectConfig{"a": {}, "b": {}}}, "", "Proyectos", configLine{lineOK, "Proyectos", "2"}},
		{"sin perfil", config.Config{}, "", "Perfil", configLine{lineWarn, "Perfil", "ninguno"}},
		{"con perfil", config.Config{}, "equipo", "Perfil", configLine{lineOK, "Perfil", "equipo"}},
		{"QA sin configurar", config.Config{}, "", "QA", configLine{lineNone, "QA", "sin configurar"}},
		{"QA incompleta", config.Config{QAJira: &config.QAJiraConfig{Project: "OPS"}}, "", "QA", configLine{lineWarn, "QA", "qa_jira incompleta"}},
		{"QA sin cuenta", config.Config{QAJira: qa}, "", "QA", configLine{lineWarn, "QA", "falta qa_email"}},
		{"QA sin qa_report", config.Config{QAJira: qa, QAEmail: "qa@example.test"}, "", "QA", configLine{lineWarn, "QA", "falta qa_report"}},
		{"QA lista", config.Config{QAJira: qa, QAEmail: "qa@example.test", QAReport: report}, "", "QA", configLine{lineOK, "QA", "proyecto OPS"}},
	}
	for _, c := range cases {
		var got *configLine
		for _, l := range configChecks(&state{cfg: &c.cfg, profile: c.profile}) {
			if l.label == c.label {
				got = &l
			}
		}
		if got == nil || *got != c.want {
			t.Errorf("%s: se esperaba %+v y se obtuvo %+v", c.name, c.want, got)
		}
	}
}
