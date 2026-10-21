package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/profile"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

// accountTracker answers Me with user, or with err.
type accountTracker struct {
	*fakeTracker
	user *provider.User
	err  error
}

func (a accountTracker) Me(context.Context) (*provider.User, error) { return a.user, a.err }

type settingsEnv struct {
	m       *app
	file    *fakeFile
	tracker accountTracker
	opened  []profile.Options
	openErr error
}

// newSettingsApp opens Configuración over file; LoadConfig adds a token, as
// ATLASSIAN_TOKEN would.
func newSettingsApp(t *testing.T, file *config.Config) *settingsEnv {
	t.Helper()
	e := &settingsEnv{file: &fakeFile{cfg: file}, tracker: accountTracker{fakeTracker: &fakeTracker{}, user: &provider.User{Name: "Ana Pérez", Email: "tu@example.test"}}}
	e.m = newApp(Deps{
		Version: "v9.9.9",
		LoadConfig: func() (*config.Config, error) {
			c := clone(t, e.file.cfg)
			c.AtlassianToken = "token-del-entorno"
			return c, nil
		},
		LoadFile:   func() (*config.Config, error) { return clone(t, e.file.cfg), nil },
		SaveConfig: func(c *config.Config) error { e.file.cfg = clone(t, c); e.file.saves++; return nil },
		OpenProfile: func(c *config.Config, o profile.Options) (*profile.Bundle, error) {
			e.opened = append(e.opened, o)
			if e.openErr != nil {
				return nil, e.openErr
			}
			return &profile.Bundle{Tracker: e.tracker}, nil
		},
	})
	e.m.Update(tea.WindowSizeMsg{Width: 110, Height: 34})
	send(e.m, keyUp)
	send(e.m, keyEnter)
	return e
}

func savedConfig() *config.Config {
	return &config.Config{
		AtlassianEmail: "tu@example.test", AtlassianToken: "token-del-archivo", BaseURL: "https://empresa.example.test",
		DefaultProfile: "equipo",
		Profiles: map[string]*config.Profile{
			"equipo":  {Tracker: config.ProviderConfig{Provider: "jira"}, Docs: config.ProviderConfig{Provider: "confluence", Space: "DEV"}},
			"externo": {Tracker: config.ProviderConfig{Provider: "jira"}, Docs: config.ProviderConfig{Provider: "markdown", Path: "./docs"}, Template: "./plantillas"},
		},
	}
}

func (e *settingsEnv) fill(values ...string) {
	for _, v := range values {
		typeText(e.m, v)
		send(e.m, keyEnter)
	}
}

func TestSettingsShowsEachSection(t *testing.T) {
	e := newSettingsApp(t, savedConfig())
	out := view(e.m)
	for _, want := range []string{"Inicio › Configuración", "▸ Conexión con Atlassian", "✓ tu@example.test en https://empresa.example.test", "Perfiles", "✓ equipo: jira → confluence (space DEV)"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	if len(e.m.stack) != 2 {
		t.Error("con credenciales y perfiles no arranca el asistente")
	}
}

func TestWizardFromAnEmptyConfig(t *testing.T) {
	e := newSettingsApp(t, &config.Config{})
	out := view(e.m)
	if !strings.Contains(out, "Inicio › Configuración › Conexión") || !strings.Contains(out, "ATLASSIAN_TOKEN está definida en el entorno") {
		t.Fatalf("sin configuración, el asistente empieza por la conexión y avisa del token del entorno:\n%s", out)
	}
	e.tracker.user = &provider.User{Name: "Ana Pérez", Email: "ana@example.test"}
	typeText(e.m, "tu@example.test")
	send(e.m, keyEnter)
	typeText(e.m, "secreto-123")
	if strings.Contains(view(e.m), "secreto-123") {
		t.Fatal("el token no se muestra al escribirlo")
	}
	send(e.m, keyEnter)
	e.fill("https://empresa.example.test/")

	if e.file.cfg.AtlassianToken != "secreto-123" || e.file.cfg.BaseURL != "https://empresa.example.test" {
		t.Fatalf("se guarda el token escrito, no el del entorno, y la URL sin / final: %q %q", e.file.cfg.AtlassianToken, e.file.cfg.BaseURL)
	}
	out = view(e.m)
	if strings.Contains(out, "secreto-123") || !strings.Contains(out, "Autenticado como Ana Pérez (ana@example.test)") || !strings.Contains(out, "no es el de la cuenta autenticada") {
		t.Fatalf("la verificación muestra la cuenta y que el email no coincide:\n%s", out)
	}
	if len(e.opened) != 1 || !e.opened[0].AnyScope {
		t.Errorf("la verificación abre el perfil sin exigir space: %+v", e.opened)
	}
	typeText(e.m, "s")
	if e.file.cfg.AtlassianEmail != "ana@example.test" || !strings.Contains(view(e.m), "Email corregido a ana@example.test") {
		t.Fatalf("s usa el email de la cuenta: %q\n%s", e.file.cfg.AtlassianEmail, view(e.m))
	}

	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Primer perfil") || !strings.Contains(out, "equipo") {
		t.Fatalf("enter sigue con el primer perfil:\n%s", out)
	}
	send(e.m, keyEnter)
	send(e.m, keyDown)
	send(e.m, keyEnter)
	e.fill("./docs/despliegues")

	got := e.file.cfg.Profiles["equipo"]
	if got == nil || got.Docs.Provider != "markdown" || got.Docs.Path != "./docs/despliegues" || e.file.cfg.DefaultProfile != "equipo" {
		t.Fatalf("el primer perfil se guarda por defecto: %+v, %q", got, e.file.cfg.DefaultProfile)
	}
	if _, ok := e.file.cfg.Profiles[config.LegacyProfile]; ok {
		t.Error("el asistente, como draco init, no guarda el perfil implícito")
	}
	out = view(e.m)
	if len(e.m.stack) != 2 || !strings.Contains(out, "✓ Perfil guardado: equipo (jira → markdown)") || !strings.Contains(out, "p crea el primer proyecto") {
		t.Fatalf("vuelve a Configuración y ofrece un proyecto:\n%s", out)
	}
	typeText(e.m, "p")
	if out := view(e.m); !strings.Contains(out, "Proyectos › Nuevo proyecto") {
		t.Errorf("p abre el formulario de Proyectos:\n%s", out)
	}
}

func TestConnectionKeepsTheTokenWhenEmpty(t *testing.T) {
	e := newSettingsApp(t, savedConfig())
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Vacío conserva el actual") || !strings.Contains(out, "tu@example.test") || strings.Contains(out, "token-del-archivo") {
		t.Fatalf("el formulario parte del archivo y no muestra el token:\n%s", out)
	}
	e.fill("", "", "")
	if e.file.cfg.AtlassianToken != "token-del-archivo" || e.file.saves != 1 {
		t.Fatalf("un token vacío conserva el guardado, y nunca el del entorno: %q", e.file.cfg.AtlassianToken)
	}
	out := view(e.m)
	if !strings.Contains(out, "Autenticado como Ana Pérez") || strings.Contains(out, "no es el de la cuenta") || !strings.Contains(out, "enter vuelve a Configuración") {
		t.Fatalf("con perfiles, la verificación termina volviendo a Configuración:\n%s", out)
	}
	send(e.m, keyEnter)
	if len(e.m.stack) != 2 {
		t.Errorf("enter vuelve a Configuración: pila %d", len(e.m.stack))
	}
}

func TestConnectionNeedsHTTPS(t *testing.T) {
	e := newSettingsApp(t, savedConfig())
	send(e.m, keyEnter)
	send(e.m, keyEnter)
	send(e.m, keyEnter)
	for range len("https://empresa.example.test") {
		send(e.m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	e.fill("http://empresa.example.test")
	if out := view(e.m); e.file.saves != 0 || !strings.Contains(out, "https://") {
		t.Errorf("una URL sin https no se guarda: %d escrituras\n%s", e.file.saves, out)
	}
}

func TestFailedCheckKeepsWhatWasSaved(t *testing.T) {
	e := newSettingsApp(t, savedConfig())
	e.tracker.err = errors.New("401 no autorizado")
	send(e.m, keyEnter)
	e.fill("", "token-nuevo", "")
	out := view(e.m)
	if e.file.cfg.AtlassianToken != "token-nuevo" || !strings.Contains(out, "No se pudo validar las credenciales: 401 no autorizado") || !strings.Contains(out, "Lo guardado se conserva") {
		t.Fatalf("un error de verificación se explica sin deshacer lo guardado:\n%s", out)
	}
	send(e.m, keyEnter)
	if len(e.m.stack) != 2 {
		t.Errorf("enter vuelve a Configuración: pila %d", len(e.m.stack))
	}
}

func (e *settingsEnv) toProfiles(t *testing.T) {
	t.Helper()
	send(e.m, keyDown)
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Configuración › Perfiles") {
		t.Fatalf("enter en Perfiles abre la lista:\n%s", out)
	}
}

func TestProfilesListDefaultAndRemove(t *testing.T) {
	e := newSettingsApp(t, savedConfig())
	e.toProfiles(t)
	out := view(e.m)
	for _, want := range []string{"▸ equipo", "por defecto", "externo", "confluence", "DEV"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	send(e.m, keyDown)
	typeText(e.m, "d")
	if e.file.cfg.DefaultProfile != "externo" || !strings.Contains(view(e.m), "perfil: externo") {
		t.Fatalf("d guarda el perfil por defecto y la cabecera lo muestra: %q\n%s", e.file.cfg.DefaultProfile, view(e.m))
	}
	typeText(e.m, "x")
	if out := view(e.m); !strings.Contains(out, "¿Eliminar el perfil «externo»?") {
		t.Fatalf("x pide confirmación:\n%s", out)
	}
	typeText(e.m, "s")
	if _, ok := e.file.cfg.Profiles["externo"]; ok || e.file.cfg.DefaultProfile != "" {
		t.Errorf("s elimina el perfil y deja de ser el por defecto: %v %q", e.file.cfg.ProfileNames(), e.file.cfg.DefaultProfile)
	}
	if e.file.cfg.AtlassianToken != "token-del-archivo" {
		t.Errorf("guardar no cambia el token del archivo: %q", e.file.cfg.AtlassianToken)
	}
}

func TestNewProfileKeepsTheImplicitOne(t *testing.T) {
	cfg := savedConfig()
	cfg.Profiles, cfg.DefaultProfile, cfg.ConfluenceSpaceKey = nil, "", "DEV"
	e := newSettingsApp(t, cfg)
	e.toProfiles(t)
	if out := view(e.m); !strings.Contains(out, "Sin perfiles") || !strings.Contains(out, "'atlassian'") || !strings.Contains(out, "(space DEV)") {
		t.Fatalf("sin perfiles, la lista explica el implícito:\n%s", out)
	}
	typeText(e.m, "n")
	typeText(e.m, "externo")
	send(e.m, keyEnter)
	send(e.m, keyDown)
	send(e.m, keyEnter)
	e.fill("./docs", "", "")

	implicit := e.file.cfg.Profiles[config.LegacyProfile]
	got := e.file.cfg.Profiles["externo"]
	if implicit == nil || implicit.Docs.Space != "DEV" || got == nil || got.Docs.Path != "./docs" || e.file.cfg.DefaultProfile != config.LegacyProfile {
		t.Fatalf("el primer perfil también guarda el implícito, que sigue por defecto: %v %+v %q", e.file.cfg.ProfileNames(), got, e.file.cfg.DefaultProfile)
	}
	if out := view(e.m); !strings.Contains(out, "También se guardó 'atlassian'") {
		t.Errorf("avisa que guardó el implícito:\n%s", out)
	}
}

func TestEditProfileKeepsItsName(t *testing.T) {
	e := newSettingsApp(t, savedConfig())
	e.toProfiles(t)
	send(e.m, keyDown)
	typeText(e.m, "e")
	if out := view(e.m); !strings.Contains(out, "Editar externo") {
		t.Fatalf("e abre el formulario del perfil:\n%s", out)
	}
	send(e.m, keyUp)
	send(e.m, keyEnter)
	e.fill("OPS", "", "")
	got := e.file.cfg.Profiles["externo"]
	if got == nil || got.Docs.Provider != "confluence" || got.Docs.Space != "OPS" || got.Template != "./plantillas" {
		t.Errorf("editar cambia el destino y conserva la plantilla: %+v", got)
	}
}

func TestNewProfileRejectsAnExistingName(t *testing.T) {
	e := newSettingsApp(t, savedConfig())
	e.toProfiles(t)
	typeText(e.m, "nequipo")
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "ya existe") || e.file.saves != 0 {
		t.Errorf("un nombre existente no avanza:\n%s", out)
	}
}
