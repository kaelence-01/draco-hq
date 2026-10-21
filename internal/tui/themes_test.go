package tui

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// colorCode is an SGR sequence that sets a foreground or background color.
var colorCode = regexp.MustCompile(`\x1b\[(?:[0-9]+;)*(?:3[0-8]|4[0-8]|9[0-7]|10[0-7])(?:;[0-9]+)*m`)

func resetTheme(t *testing.T) { t.Cleanup(func() { applyTheme(defaultTheme) }) }

func (e *settingsEnv) toThemes(t *testing.T) {
	t.Helper()
	for range 4 {
		send(e.m, keyDown)
	}
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Configuración › Tema") || !strings.Contains(out, "▸ "+theme) {
		t.Fatalf("enter en Tema abre la lista, con el actual elegido:\n%s", out)
	}
}

func TestThemeIsTriedSavedAndApplied(t *testing.T) {
	resetTheme(t)
	e := newSettingsApp(t, savedConfig())
	if out := view(e.m); !strings.Contains(out, "✓ ansi (por defecto)") {
		t.Fatalf("la sección muestra el tema:\n%s", out)
	}
	e.toThemes(t)
	send(e.m, keyDown)
	if theme != "alto-contraste" || !strings.Contains(view(e.m), "▸ alto-contraste") {
		t.Fatalf("moverse prueba el tema en toda la pantalla: %q", theme)
	}
	send(e.m, keyDown)
	send(e.m, keyEnter)
	if e.file.cfg.Theme != "mono" || theme != "mono" || len(e.m.stack) != 2 {
		t.Fatalf("enter guarda el tema y vuelve a Configuración: %q %q", e.file.cfg.Theme, theme)
	}
	if out := view(e.m); !strings.Contains(out, "✓ Tema guardado: mono") || !strings.Contains(out, "✓ mono") {
		t.Errorf("Configuración muestra el tema guardado:\n%s", out)
	}
	if raw := e.m.View().Content; colorCode.MatchString(raw) {
		t.Errorf("mono no emite color: %q", colorCode.FindString(raw))
	}
	if e.file.cfg.AtlassianToken != "token-del-archivo" {
		t.Errorf("guardar el tema no toca el token del archivo: %q", e.file.cfg.AtlassianToken)
	}
}

func TestLeavingThePickerPutsTheSavedThemeBack(t *testing.T) {
	resetTheme(t)
	e := newSettingsApp(t, savedConfig())
	e.toThemes(t)
	send(e.m, keyDown)
	send(e.m, keyEsc)
	if theme != "ansi" || e.file.saves != 0 {
		t.Errorf("esc vuelve al tema guardado sin guardar nada: %q, %d escrituras", theme, e.file.saves)
	}
}

func TestDefaultThemeIsSavedEmpty(t *testing.T) {
	resetTheme(t)
	cfg := savedConfig()
	cfg.Theme = "mono"
	e := newSettingsApp(t, cfg)
	if theme != "mono" {
		t.Fatalf("el tema de la config se aplica al abrir: %q", theme)
	}
	for range 4 {
		send(e.m, keyDown)
	}
	send(e.m, keyEnter)
	send(e.m, keyUp)
	send(e.m, keyUp)
	send(e.m, keyEnter)
	if e.file.saves != 1 || e.file.cfg.Theme != "" || theme != "ansi" {
		t.Errorf("elegir ansi lo guarda vacío, el valor por defecto: %q %q", e.file.cfg.Theme, theme)
	}
}

func TestEveryThemeFitsIn80Columns(t *testing.T) {
	resetTheme(t)
	for _, th := range themes {
		cfg := savedConfig()
		cfg.Theme = th.name
		e := newSettingsApp(t, cfg)
		e.m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		e.toThemes(t)
		for _, out := range []string{view(e.m), ansi.Strip(e.m.View().Content)} {
			for _, l := range strings.Split(out, "\n") {
				if w := ansi.StringWidth(l); w > 80 {
					t.Errorf("%s: una línea mide %d columnas: %q", th.name, w, l)
				}
			}
		}
		if raw := e.m.View().Content; th.name != "mono" && !colorCode.MatchString(raw) {
			t.Errorf("%s usa colores", th.name)
		}
	}
}

func TestUnknownThemeFallsBackToTheDefault(t *testing.T) {
	resetTheme(t)
	cfg := savedConfig()
	cfg.Theme = "neon"
	e := newSettingsApp(t, cfg)
	if out := view(e.m); theme != "ansi" || !strings.Contains(out, "⚠ «neon» no existe: se usa ansi") {
		t.Errorf("un tema desconocido usa el de por defecto y se avisa: %q\n%s", theme, out)
	}
}
