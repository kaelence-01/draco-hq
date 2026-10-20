package tui

import (
	"errors"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kaelence-01/draco-hq/internal/config"
)

func testConfig() *config.Config {
	return &config.Config{
		AtlassianEmail: "tu@example.test", AtlassianToken: "x", BaseURL: "https://example.test",
		DefaultProject: "pagos",
		Projects:       map[string]*config.ProjectConfig{"pagos": {BackendRepo: "pagos-api"}},
		DefaultProfile: "equipo",
		Profiles: map[string]*config.Profile{"equipo": {
			Tracker: config.ProviderConfig{Provider: "jira"},
			Docs:    config.ProviderConfig{Provider: "confluence", Space: "DEV"},
		}},
	}
}

func newTestApp(t *testing.T, cfg *config.Config, err error, width, height int) *app {
	t.Helper()
	m := newApp(Deps{LoadConfig: func() (*config.Config, error) { return cfg, err }, Version: "v9.9.9"})
	m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	return m
}

func press(code rune, text string) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code, Text: text} }

var (
	keyQ     = press('q', "q")
	keyHelp  = press('?', "?")
	keyDown  = tea.KeyPressMsg{Code: tea.KeyDown}
	keyUp    = tea.KeyPressMsg{Code: tea.KeyUp}
	keyEnter = tea.KeyPressMsg{Code: tea.KeyEnter}
	keyEsc   = tea.KeyPressMsg{Code: tea.KeyEsc}
	keyCtrlC = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
)

// send runs msg through the app and then the messages its command returns,
// the way the Bubble Tea loop would; it reports whether the app quit.
func send(m *app, msg tea.Msg) (quit bool) {
	for msg != nil {
		var cmd tea.Cmd
		_, cmd = m.Update(msg)
		if cmd == nil {
			return false
		}
		msg = cmd()
		if _, ok := msg.(tea.QuitMsg); ok {
			return true
		}
	}
	return false
}

func view(m *app) string { return ansi.Strip(m.View().Content) }

func TestHomeShowsLogoMenuAndFrame(t *testing.T) {
	m := newTestApp(t, testConfig(), nil, 100, 30)
	out := view(m)
	for _, want := range []string{
		"██████", "draco v9.9.9", "perfil: equipo (jira → confluence)", "proyecto: pagos",
		"▸ Generar documento de despliegue", "draco g", "Proyectos", "Consolidado QA", "draco backlog scan",
		"q salir", "? ayuda",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en la pantalla:\n%s", want, out)
		}
	}
	if strings.Contains(out, "⚠") {
		t.Errorf("con la config completa no hay avisos:\n%s", out)
	}
	if lines := strings.Split(out, "\n"); len(lines) != 30 {
		t.Errorf("la pantalla ocupa %d líneas, se esperaban 30", len(lines))
	}
}

func TestNarrowTerminalShowsPlainName(t *testing.T) {
	m := newTestApp(t, testConfig(), nil, 40, 24)
	out := view(m)
	if strings.Contains(out, "█") || !strings.Contains(out, "draco") {
		t.Errorf("en una terminal angosta va el nombre sin letras grandes:\n%s", out)
	}
}

func TestNothingRenderedBeforeTheSize(t *testing.T) {
	m := newApp(Deps{LoadConfig: func() (*config.Config, error) { return testConfig(), nil }})
	if got := m.View().Content; got != "" {
		t.Errorf("sin tamaño no se dibuja nada, se obtuvo %q", got)
	}
	if !m.View().AltScreen {
		t.Error("la TUI usa la pantalla alternativa")
	}
}

func TestMenuMovesAndPointsToTheCommand(t *testing.T) {
	m := newTestApp(t, testConfig(), nil, 100, 30)
	send(m, keyUp)
	if out := view(m); !strings.Contains(out, "▸ Configuración") {
		t.Fatalf("subir desde el primero va al último:\n%s", out)
	}
	send(m, keyDown)
	send(m, keyDown)
	send(m, keyEnter)
	if out := view(m); !strings.Contains(out, "«Proyectos» todavía no está en el modo visual: usa draco project") {
		t.Errorf("una función sin pantalla indica su comando:\n%s", out)
	}
	send(m, keyDown)
	if out := view(m); strings.Contains(out, "todavía no está") {
		t.Errorf("moverse borra el aviso:\n%s", out)
	}
}

func TestQuitKeys(t *testing.T) {
	for name, k := range map[string]tea.KeyPressMsg{"q": keyQ, "ctrl+c": keyCtrlC} {
		if !send(newTestApp(t, testConfig(), nil, 100, 30), k) {
			t.Errorf("%s sale desde la pantalla principal", name)
		}
	}
	if send(newTestApp(t, testConfig(), nil, 100, 30), keyEsc) {
		t.Error("esc en la pantalla principal no sale")
	}
}

func TestHelpToggle(t *testing.T) {
	m := newTestApp(t, testConfig(), nil, 100, 30)
	short := view(m)
	send(m, keyHelp)
	if full := view(m); full == short || !strings.Contains(full, "subir") {
		t.Errorf("? muestra la ayuda completa:\n%s", full)
	}
}

type fakeScreen struct{ got []tea.Msg }

func (f *fakeScreen) update(msg tea.Msg) (screen, tea.Cmd) { f.got = append(f.got, msg); return f, nil }
func (f *fakeScreen) view(int, int) string                 { return "pantalla de prueba" }
func (f *fakeScreen) keys() []key.Binding                  { return nil }

func TestStackOpensAndGoesBack(t *testing.T) {
	m := newTestApp(t, testConfig(), nil, 100, 30)
	f := &fakeScreen{}
	send(m, openMsg{f})
	out := view(m)
	if !strings.Contains(out, "pantalla de prueba") || !strings.Contains(out, "esc volver") {
		t.Fatalf("la pantalla abierta se dibuja dentro del marco, con esc:\n%s", out)
	}
	if send(m, keyQ) {
		t.Error("q solo sale desde la pantalla principal")
	}
	if len(f.got) != 1 {
		t.Errorf("las teclas van a la pantalla de arriba: %v", f.got)
	}
	send(m, keyEsc)
	if out := view(m); !strings.Contains(out, "Generar documento") || strings.Contains(out, "esc volver") {
		t.Errorf("esc vuelve a la principal:\n%s", out)
	}
}

func TestConfigProblemsAreShown(t *testing.T) {
	cases := []struct {
		name string
		cfg  *config.Config
		err  error
		want string
	}{
		{"sin config", &config.Config{}, nil, "corre draco init"},
		{"config inválida", nil, errors.New("yaml: línea 3"), "No se pudo leer la configuración: yaml: línea 3"},
		{"varios perfiles", &config.Config{Profiles: map[string]*config.Profile{"a": {}, "b": {}}}, nil, "ninguno por defecto"},
	}
	for _, c := range cases {
		out := view(newTestApp(t, c.cfg, c.err, 120, 30))
		if !strings.Contains(out, "⚠") || !strings.Contains(out, c.want) {
			t.Errorf("%s: falta el aviso %q:\n%s", c.name, c.want, out)
		}
	}

	local := &config.Config{DefaultProfile: "local", Profiles: map[string]*config.Profile{"local": {
		Tracker: config.ProviderConfig{Provider: "jira"}, Docs: config.ProviderConfig{Provider: "markdown", Path: "docs"},
	}}}
	if out := view(newTestApp(t, local, nil, 120, 30)); !strings.Contains(out, "corre draco init") {
		t.Errorf("Jira sin credenciales también pide draco init:\n%s", out)
	}
}
