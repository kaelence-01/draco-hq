package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"gopkg.in/yaml.v3"

	"github.com/kaelence-01/draco-hq/internal/config"
)

// fakeFile is the config file in memory; LoadConfig adds a token, as
// ATLASSIAN_TOKEN would.
type fakeFile struct {
	cfg   *config.Config
	saves int
}

func clone(t *testing.T, c *config.Config) *config.Config {
	t.Helper()
	data, err := yaml.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	out := &config.Config{}
	if err := yaml.Unmarshal(data, out); err != nil {
		t.Fatal(err)
	}
	return out
}

func newProjectsApp(t *testing.T, change ...func(*config.Config)) (*app, *fakeFile) {
	t.Helper()
	f := &fakeFile{cfg: &config.Config{
		AtlassianEmail: "tu@example.test", BaseURL: "https://example.test",
		DefaultProject: "pagos",
		Projects: map[string]*config.ProjectConfig{
			"pagos": {BackendPath: "~/code/pagos-api", BackendRepo: "pagos-api", VCSOrg: "mi-org",
				DeployChecklist: []string{"Ejecutar migraciones"}, Backlog: &config.BacklogConfig{Backend: &config.BacklogRepoConfig{MaxLines: 600}}},
			"facturacion": {BackendPath: "~/code/facturacion-api", BackendRepo: "facturacion-api"},
		},
		DefaultProfile: "equipo",
		Profiles: map[string]*config.Profile{"equipo": {
			Tracker: config.ProviderConfig{Provider: "jira"}, Docs: config.ProviderConfig{Provider: "confluence", Space: "DEV"},
		}},
	}}
	for _, c := range change {
		c(f.cfg)
	}
	m := newApp(Deps{
		Version: "v9.9.9",
		LoadConfig: func() (*config.Config, error) {
			c := clone(t, f.cfg)
			c.AtlassianToken = "token-del-entorno"
			return c, nil
		},
		LoadFile:   func() (*config.Config, error) { return clone(t, f.cfg), nil },
		SaveConfig: func(c *config.Config) error { f.cfg = clone(t, c); f.saves++; return nil },
	})
	m.Update(tea.WindowSizeMsg{Width: 110, Height: 34})
	send(m, keyDown)
	send(m, keyEnter)
	return m, f
}

func assertNoToken(t *testing.T, f *fakeFile) {
	t.Helper()
	if f.cfg.AtlassianToken != "" {
		t.Errorf("lo guardado no trae el token del entorno: %q", f.cfg.AtlassianToken)
	}
}

func TestProjectsListAndDetail(t *testing.T) {
	m, _ := newProjectsApp(t)
	out := view(m)
	for _, want := range []string{"Inicio › Proyectos", "facturacion", "▸ pagos", "activo", "por defecto", "~/code/pagos-api", "mi-org", "Ejecutar migraciones", "n nuevo"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
}

func TestProjectsFooterKeepsEveryKeyInTheFullHelp(t *testing.T) {
	m, _ := newProjectsApp(t)
	lines := strings.Split(view(m), "\n")
	short := lines[len(lines)-1]
	for _, want := range []string{"enter usar en esta sesión", "n nuevo", "e editar", "/ filtrar", "? más"} {
		if !strings.Contains(short, want) {
			t.Errorf("el pie lleva %q: %q", want, short)
		}
	}
	if strings.Contains(short, "eliminar") {
		t.Errorf("eliminar solo está en la ayuda completa: %q", short)
	}
	send(m, keyHelp)
	full := view(m)
	for _, want := range []string{"d por defecto", "x eliminar", "n nuevo", "esc volver"} {
		if !strings.Contains(squash(full), want) {
			t.Errorf("la ayuda completa lleva %q:\n%s", want, full)
		}
	}
}

func TestProjectBringsItsProfile(t *testing.T) {
	m, f := newProjectsApp(t, func(c *config.Config) {
		c.Profiles["servicios"] = &config.Profile{Tracker: config.ProviderConfig{Provider: "jira"}, Docs: config.ProviderConfig{Provider: "confluence", Space: "SRV"}}
		c.Projects["facturacion"].Profile = "servicios"
	})
	if out := view(m); !strings.Contains(out, "perfil: equipo") {
		t.Fatalf("sin perfil propio, el proyecto usa el por defecto:\n%s", out)
	}
	send(m, keyUp)
	send(m, keyEnter)
	out := view(m)
	for _, want := range []string{"perfil: servicios", "proyecto: facturacion", "con el perfil servicios"} {
		if !strings.Contains(out, want) {
			t.Errorf("elegir el proyecto trae su perfil: falta %q\n%s", want, out)
		}
	}

	send(m, keyDown)
	typeText(m, "e")
	for range 7 {
		send(m, keyEnter)
	}
	if out := view(m); !strings.Contains(out, "Perfil del proyecto") || !strings.Contains(out, "El perfil por defecto") {
		t.Fatalf("con varios perfiles, el formulario pregunta el del proyecto:\n%s", out)
	}
	send(m, keyDown)
	send(m, keyDown)
	for range 3 {
		send(m, keyEnter)
	}
	if got := f.cfg.Projects["pagos"].Profile; got != "servicios" {
		t.Fatalf("se guarda el perfil elegido: %q\n%s", got, view(m))
	}
	if out := view(m); !strings.Contains(out, "perfil") || !strings.Contains(out, "servicios") {
		t.Errorf("el detalle muestra el perfil:\n%s", out)
	}
}

func TestActiveProjectIsNotSaved(t *testing.T) {
	m, f := newProjectsApp(t)
	send(m, keyUp)
	send(m, keyEnter)
	out := view(m)
	if !strings.Contains(out, "proyecto: facturacion") || !strings.Contains(out, "la configuración no cambia") {
		t.Errorf("enter cambia el proyecto de la sesión:\n%s", out)
	}
	if f.saves != 0 {
		t.Errorf("el proyecto activo no se guarda: %d escrituras", f.saves)
	}
	send(m, keyEsc)
	if out := view(m); !strings.Contains(out, "proyecto: facturacion") {
		t.Errorf("el proyecto activo sigue al volver a la principal:\n%s", out)
	}
}

func TestDefaultProjectIsSaved(t *testing.T) {
	m, f := newProjectsApp(t)
	send(m, keyUp)
	typeText(m, "d")
	if f.cfg.DefaultProject != "facturacion" || f.saves != 1 {
		t.Errorf("d guarda el proyecto por defecto: %q, %d escrituras", f.cfg.DefaultProject, f.saves)
	}
	assertNoToken(t, f)
	if out := view(m); !strings.Contains(out, "✓ Proyecto por defecto: facturacion") {
		t.Errorf("falta la confirmación:\n%s", out)
	}
}

func TestFilterCapturesTheKeys(t *testing.T) {
	m, _ := newProjectsApp(t)
	typeText(m, "/fa?")
	out := view(m)
	if !strings.Contains(out, "/ fa?") || !strings.Contains(out, "Ningún proyecto coincide") || m.help.ShowAll {
		t.Errorf("mientras se filtra, ? es parte del filtro:\n%s", out)
	}
	send(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	send(m, keyEnter)
	out = view(m)
	if !strings.Contains(out, "filtro: fa") || !strings.Contains(out, "▸ facturacion") || strings.Contains(out, "▸ pagos") {
		t.Errorf("enter deja el filtro aplicado:\n%s", out)
	}
	send(m, keyEsc)
	if out := view(m); !strings.Contains(out, "Inicio › Proyectos") || strings.Contains(out, "filtro:") {
		t.Errorf("esc primero quita el filtro:\n%s", out)
	}
	send(m, keyEsc)
	if len(m.stack) != 1 {
		t.Error("sin filtro, esc vuelve a la principal")
	}
}

func TestRemoveAsksFirst(t *testing.T) {
	m, f := newProjectsApp(t)
	typeText(m, "x")
	if out := view(m); !strings.Contains(out, "¿Eliminar el proyecto «pagos»?") {
		t.Fatalf("x pide confirmación:\n%s", out)
	}
	send(m, keyEsc)
	if f.saves != 0 || len(m.stack) != 2 {
		t.Fatalf("esc cancela el borrado sin cerrar la pantalla: %d escrituras", f.saves)
	}
	typeText(m, "xs")
	if _, ok := f.cfg.Projects["pagos"]; ok || f.cfg.DefaultProject != "" {
		t.Errorf("s borra el proyecto y deja de ser el por defecto: %v, %q", f.cfg.ProjectNames(), f.cfg.DefaultProject)
	}
	assertNoToken(t, f)
	out := view(m)
	if !strings.Contains(out, "Era el proyecto por defecto") || strings.Contains(out, "proyecto: pagos") {
		t.Errorf("avisa que era el por defecto y pagos deja de ser el activo:\n%s", out)
	}
}

func TestCreateProject(t *testing.T) {
	m, f := newProjectsApp(t)
	typeText(m, "n")
	if out := view(m); !strings.Contains(out, "Inicio › Proyectos › Nuevo proyecto") || !strings.Contains(out, "Nombre") {
		t.Fatalf("n abre el formulario:\n%s", out)
	}
	typeText(m, "reportes?")
	send(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	for _, v := range []string{"", "~/code/reportes-api", "", "", "", "https://github.com", "mi-org", ""} {
		typeText(m, v)
		send(m, keyEnter)
	}
	if out := view(m); !strings.Contains(out, "Backlog técnico del backend") || strings.Contains(out, "frontend") {
		t.Fatalf("sigue la página opcional del backlog, solo de los repos con ruta:\n%s", out)
	}
	send(m, keyEnter)
	send(m, keyEnter)
	got, ok := f.cfg.Projects["reportes"]
	if !ok || len(m.stack) != 2 {
		t.Fatalf("el proyecto se guarda y vuelve a la lista: %v\n%s", f.cfg.ProjectNames(), view(m))
	}
	if got.BackendRepo != "reportes-api" || got.VCSHost != "https://github.com" || got.FrontendRepo != "" || got.Backlog != nil {
		t.Errorf("el repo sale de la carpeta y lo vacío queda vacío, backlog incluido: %+v", got)
	}
	if f.cfg.DefaultProject != "pagos" {
		t.Errorf("con un por defecto, el nuevo no lo reemplaza: %q", f.cfg.DefaultProject)
	}
	assertNoToken(t, f)
	if out := view(m); !strings.Contains(out, "✓ Proyecto guardado: reportes") {
		t.Errorf("falta la confirmación:\n%s", out)
	}
}

func TestProjectBacklogPatterns(t *testing.T) {
	m, f := newProjectsApp(t)
	typeText(m, "n")
	for _, v := range []string{"reportes", "/repos/reportes-api", "", "", "", "", "", "", "app/[Services/*.php"} {
		typeText(m, v)
		send(m, keyEnter)
	}
	if out := view(m); !strings.Contains(out, "patrón de clases inválido") || f.cfg.Projects["reportes"] != nil {
		t.Fatalf("un patrón mal formado no avanza ni se guarda:\n%s", out)
	}
	for range len("[Services/*.php") {
		send(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	typeText(m, "**/Services/**/*.php, app/Models/*.php")
	send(m, keyEnter)
	typeText(m, "app/Http/{modulo}/**")
	send(m, keyEnter)

	got := f.cfg.Projects["reportes"].BacklogFor("backend")
	if got == nil || len(got.ClassGlobs) != 2 || got.ClassGlobs[0] != "app/**/Services/**/*.php" || len(got.Modules) != 1 {
		t.Fatalf("las listas se guardan separadas por comas: %+v", got)
	}
	if f.cfg.Projects["reportes"].BacklogFor("frontend") != nil {
		t.Error("sin ruta de frontend no hay backlog de frontend")
	}
	typeText(m, "/reportes")
	if out := view(m); !strings.Contains(out, "backlog backend") || !strings.Contains(out, "módulos: app/Http/{modulo}/**") {
		t.Errorf("el detalle muestra el backlog:\n%s", out)
	}
}

func TestFormRejectsAnExistingName(t *testing.T) {
	m, f := newProjectsApp(t)
	typeText(m, "npagos")
	send(m, keyEnter)
	if out := view(m); !strings.Contains(out, "ya existe") || f.saves != 0 {
		t.Errorf("un nombre existente no avanza:\n%s", out)
	}
}

func TestFormWithoutPathsIsNotSaved(t *testing.T) {
	m, f := newProjectsApp(t)
	typeText(m, "nvacio")
	for range 8 {
		send(m, keyEnter)
	}
	if f.saves != 0 || !strings.Contains(view(m), "El proyecto no se guardó: sin rutas de backend ni de frontend") {
		t.Errorf("sin rutas no se guarda: %d escrituras\n%s", f.saves, view(m))
	}
}

func TestEscClosesTheFormWithoutSaving(t *testing.T) {
	m, f := newProjectsApp(t)
	typeText(m, "nreportes")
	send(m, keyEsc)
	if f.saves != 0 || len(m.stack) != 2 {
		t.Errorf("esc cierra el formulario sin guardar: %d escrituras, pila %d", f.saves, len(m.stack))
	}
}

func TestEditKeepsWhatTheFormDoesNotShow(t *testing.T) {
	m, f := newProjectsApp(t)
	typeText(m, "e")
	if out := view(m); !strings.Contains(out, "Editar pagos") || !strings.Contains(out, "~/code/pagos-api") {
		t.Fatalf("e abre el formulario con los valores del proyecto:\n%s", out)
	}
	send(m, keyEnter)
	send(m, keyEnter)
	typeText(m, "~/code/pagos-web")
	for range 5 + 2 + 2 {
		send(m, keyEnter)
	}
	got := f.cfg.Projects["pagos"]
	if got == nil || got.FrontendRepo != "pagos-web" || got.BackendRepo != "pagos-api" {
		t.Fatalf("la edición se guarda: %+v", got)
	}
	if len(got.DeployChecklist) != 1 || got.Backlog == nil || got.Backlog.Backend.MaxLines != 600 {
		t.Errorf("editar conserva checklist y backlog: %+v", got)
	}
	assertNoToken(t, f)
}
