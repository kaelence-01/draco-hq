package tui

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/profile"
	"github.com/kaelence-01/draco-hq/internal/provider"
	"github.com/kaelence-01/draco-hq/internal/provider/markdown"
)

// fakeTracker knows OPS-7; OPS-9 waits until its step is canceled. With
// linkWait, linking waits until it is closed.
type fakeTracker struct {
	canceled chan struct{}
	linked   []string
	linkErr  error
	linkWait chan struct{}
}

func (t *fakeTracker) Issue(ctx context.Context, key string) (*provider.Issue, error) {
	switch key {
	case "OPS-7":
		return &provider.Issue{Key: "OPS-7", Summary: "Filtro de pagos", URL: "https://example.test/browse/OPS-7"}, nil
	case "OPS-9":
		<-ctx.Done()
		close(t.canceled)
		return nil, ctx.Err()
	}
	return nil, errors.New("recurso no encontrado (404)")
}
func (*fakeTracker) Me(context.Context) (*provider.User, error) { return &provider.User{ID: "yo"}, nil }
func (t *fakeTracker) LinkDocument(_ context.Context, key string, d provider.DocRef) error {
	if t.linkWait != nil {
		<-t.linkWait
	}
	t.linked = append(t.linked, key+" → "+d.Title)
	return t.linkErr
}

// fakeGit has a backend and a frontend clone; the hash "malo" does not exist.
type fakeGit struct{ read map[string][]string }

func (fakeGit) CurrentBranch(_ context.Context, dir string) (string, error) {
	return map[string]string{"/code/pagos-api": "develop", "/code/pagos-web": "filtro-fechas"}[dir], nil
}

func (fakeGit) RecentCommits(_ context.Context, dir string, _ int) ([]provider.Commit, error) {
	day := time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)
	if dir == "/code/pagos-web" {
		return []provider.Commit{{ShortHash: "e4f5a6b", Subject: "Filtro de fechas en la lista", Author: "Sofía", Date: day}}, nil
	}
	return []provider.Commit{
		{ShortHash: "a1b2c3d", Subject: "Agrega filtro por rango de fechas", Author: "Ana", Date: day},
		{ShortHash: "9f8e7d6", Subject: "Valida rango máximo", Author: "Ana", Date: day},
		{ShortHash: "5c4b3a2", Subject: "Índices para la tabla de pagos", Author: "Luis", Date: day},
	}, nil
}

func (g fakeGit) ChangedFiles(_ context.Context, dir string, hashes []string) ([]string, error) {
	g.read[dir] = hashes
	if slices.Contains(hashes, "malo") {
		return nil, errors.New("el commit malo no existe en el repo local")
	}
	if dir == "/code/pagos-web" {
		return []string{"src/app/pagos/filtro.component.ts"}, nil
	}
	return []string{"app/Http/Pagos/FiltroController.php", "routes/api.php"}, nil
}

type generateEnv struct {
	m       *app
	tracker *fakeTracker
	git     fakeGit
	docs    *markdown.Store
	store   provider.DocStore // what the profile publishes to; docs unless a test replaces it
	opened  []profile.Options
}

func newGenerateApp(t *testing.T, openErr error) *generateEnv {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Chdir(t.TempDir())
	docs, err := markdown.New(filepath.Join(t.TempDir(), "despliegues"))
	if err != nil {
		t.Fatal(err)
	}
	e := &generateEnv{tracker: &fakeTracker{canceled: make(chan struct{})}, git: fakeGit{read: map[string][]string{}}, docs: docs, store: docs}
	cfg := &config.Config{
		DefaultProject: "pagos",
		Projects: map[string]*config.ProjectConfig{
			"pagos": {BackendPath: "/code/pagos-api", BackendRepo: "pagos-api", FrontendPath: "/code/pagos-web", FrontendRepo: "pagos-web",
				VCSHost: "https://github.com", VCSOrg: "mi-org", ConfluenceSpaceKey: "PAG"},
			"facturacion": {BackendPath: "/code/facturacion-api", BackendRepo: "facturacion-api"},
		},
		DefaultProfile: "local",
		Profiles: map[string]*config.Profile{"local": {
			Tracker: config.ProviderConfig{Provider: "jira"}, Docs: config.ProviderConfig{Provider: "markdown", Path: "despliegues"},
		}},
		AtlassianEmail: "tu@example.test", AtlassianToken: "x", BaseURL: "https://example.test",
	}
	e.m = newApp(Deps{
		Version:    "v9.9.9",
		LoadConfig: func() (*config.Config, error) { return cfg, nil },
		OpenProfile: func(c *config.Config, o profile.Options) (*profile.Bundle, error) {
			e.opened = append(e.opened, o)
			if openErr != nil {
				return nil, openErr
			}
			return &profile.Bundle{Name: "local", Profile: *c.Profiles["local"], Tracker: e.tracker, Docs: e.store, Changes: e.git}, nil
		},
	})
	e.m.Update(tea.WindowSizeMsg{Width: 110, Height: 34})
	send(e.m, keyEnter)
	return e
}

var keySpace = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}

// toCommits fills the task and keeps the active project.
func (e *generateEnv) toCommits(t *testing.T, issue string) {
	t.Helper()
	typeText(e.m, issue)
	send(e.m, keyEnter)
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Commits de "+issue) {
		t.Fatalf("el formulario lleva a los commits:\n%s", out)
	}
}

func TestGenerateFormAsksTaskAndProject(t *testing.T) {
	e := newGenerateApp(t, nil)
	out := view(e.m)
	for _, want := range []string{"Inicio › Generar documento", "Tarea", "Proyecto", "pagos"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "falta la clave de la tarea") {
		t.Errorf("la tarea es obligatoria:\n%s", out)
	}
}

func TestGenerateOpensTheProfileWithTheProjectSpace(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toCommits(t, "OPS-7")
	if len(e.opened) != 1 || e.opened[0].ProjectSpace != "PAG" || e.opened[0].Scope != "" || e.opened[0].Name != "" {
		t.Errorf("el perfil se abre con el space del proyecto: %+v", e.opened)
	}
}

func TestGenerateUsesTheProjectProfile(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.m.st.cfg.Projects["pagos"].Profile = "local"
	e.toCommits(t, "OPS-7")
	if len(e.opened) != 1 || e.opened[0].Name != "local" {
		t.Errorf("el perfil del proyecto se usa al generar: %+v", e.opened)
	}
}

func TestGenerateExplainsAProfileError(t *testing.T) {
	e := newGenerateApp(t, profile.ErrNoSpace)
	typeText(e.m, "OPS-7")
	send(e.m, keyEnter)
	send(e.m, keyEnter)
	out := view(e.m)
	if !strings.Contains(out, "space key no configurado") || !strings.Contains(out, "ponlo en el proyecto") || !strings.Contains(out, "Tarea") {
		t.Errorf("el error se explica y el formulario sigue:\n%s", out)
	}
}

func TestCommitsListsEachSideWithItsBranch(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toCommits(t, "OPS-7")
	out := view(e.m)
	for _, want := range []string{"▸ Backend", "Frontend", "pagos-api", "rama develop", "a1b2c3d", "Agrega filtro por rango de fechas", "Ana · 2026-10-19"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	send(e.m, tea.KeyPressMsg{Code: tea.KeyTab})
	if out := view(e.m); !strings.Contains(out, "rama filtro-fechas") || !strings.Contains(out, "e4f5a6b") {
		t.Errorf("tab cambia al frontend:\n%s", out)
	}
}

func TestCommitsNeedsOne(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toCommits(t, "OPS-7")
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Marca al menos un commit") {
		t.Errorf("sin commits no se sigue:\n%s", out)
	}
}

func TestGenerateReadsTheMarkedCommitsInListOrder(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toCommits(t, "OPS-7")
	send(e.m, keyDown)
	send(e.m, keySpace)
	send(e.m, keyUp)
	send(e.m, keySpace)
	typeText(e.m, "hf1e2d3c")
	send(e.m, keyEnter)
	send(e.m, tea.KeyPressMsg{Code: tea.KeyTab})
	send(e.m, keySpace)
	if out := view(e.m); !strings.Contains(out, "Backend (3)") || !strings.Contains(out, "Frontend (1)") {
		t.Fatalf("las pestañas cuentan lo marcado:\n%s", out)
	}
	send(e.m, keyEnter)

	if got := e.git.read["/code/pagos-api"]; strings.Join(got, ",") != "a1b2c3d,9f8e7d6,f1e2d3c" {
		t.Errorf("los marcados van en el orden de la lista y el manual al final: %v", got)
	}
	out := view(e.m)
	for _, want := range []string{
		"Inicio › Generar documento › Commits de OPS-7 › Preparar documento", "✓ OPS-7 — Filtro de pagos", "se creará uno nuevo",
		"Backend · pagos-api", "2 archivos", "app/Http/Pagos/ FiltroController.php", "Frontend · pagos-web",
		"1 archivo", "enter abre la vista previa",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
}

func TestGenerateKeepsGoingWhenOneSideFails(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toCommits(t, "OPS-7")
	send(e.m, keySpace)
	send(e.m, tea.KeyPressMsg{Code: tea.KeyTab})
	typeText(e.m, "hmalo")
	send(e.m, keyEnter)
	send(e.m, keyEnter)
	out := view(e.m)
	if !strings.Contains(out, "✗ Frontend [malo]") || !strings.Contains(out, "Uno de los lados falló") || !strings.Contains(out, "Backend · pagos-api") {
		t.Errorf("un lado que falla se avisa y se sigue con el otro:\n%s", out)
	}
}

func TestGenerateShowsATrackerError(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toCommits(t, "OPS-404")
	send(e.m, keySpace)
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "✗ Buscando la tarea OPS-404") || !strings.Contains(out, "recurso no encontrado (404)") {
		t.Errorf("el error del paso se muestra:\n%s", out)
	}
}

func TestLongErrorsWrap(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toCommits(t, "OPS-404")
	send(e.m, keySpace)
	send(e.m, keyEnter)
	e.m.Update(tea.WindowSizeMsg{Width: 24, Height: 30})
	if out := view(e.m); !strings.Contains(out, "(404)") {
		t.Errorf("un error largo pasa a la línea siguiente, no se corta:\n%s", out)
	}
}

func TestGenerateAsksAboutAnExistingDocument(t *testing.T) {
	for _, c := range []struct {
		key    string
		read   bool
		update bool
	}{{"1", true, true}, {"2", true, false}, {"3", false, false}} {
		e := newGenerateApp(t, nil)
		if _, err := e.docs.Create(context.Background(), provider.DocRef{ID: "x.md"}, provider.Document{Title: "Documento de Despliegue - OPS-7 - Título viejo", Markdown: "# x"}); err != nil {
			t.Fatal(err)
		}
		e.toCommits(t, "OPS-7")
		send(e.m, keySpace)
		send(e.m, keyEnter)
		if out := view(e.m); !strings.Contains(out, "Ya existe: Documento de Despliegue - OPS-7 - Título viejo") || !strings.Contains(out, "1 actualizar") {
			t.Fatalf("pregunta qué hacer con el documento existente:\n%s", out)
		}
		typeText(e.m, c.key)
		_, read := e.git.read["/code/pagos-api"]
		if read != c.read {
			t.Errorf("%s: archivos leídos = %v", c.key, read)
		}
		if out := view(e.m); strings.Contains(out, "se actualizará el documento existente") != c.update {
			t.Errorf("%s: actualizar = %v:\n%s", c.key, c.update, out)
		}
		if !c.read && !strings.Contains(view(e.m), "▸ Backend") {
			t.Errorf("3 cancela y vuelve a los commits:\n%s", view(e.m))
		}
	}
}

func TestEscCancelsARunningStep(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toCommits(t, "OPS-9")
	send(e.m, keySpace)
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Buscando la tarea OPS-9…") {
		t.Fatalf("el paso está en curso:\n%s", out)
	}
	p := e.m.top().(*progress)
	send(e.m, keyEsc)
	select {
	case <-e.tracker.canceled:
	case <-time.After(time.Second):
		t.Fatal("esc cancela el paso en curso")
	}
	if out := view(e.m); !strings.Contains(out, "▸ Backend") {
		t.Errorf("esc vuelve a los commits:\n%s", out)
	}

	send(e.m, stepDoneMsg{owner: p, step: stepFetch})
	send(e.m, keyEnter)
	next := e.m.top().(*progress)
	send(e.m, stepDoneMsg{owner: p, step: stepFetch})
	if next.step != stepFetch {
		t.Errorf("el resultado de un paso cancelado se descarta: paso %d", next.step)
	}
}

func TestBackFromCommitsKeepsTheForm(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toCommits(t, "OPS-7")
	send(e.m, keyEsc)
	out := view(e.m)
	if !strings.Contains(out, "OPS-7") || !strings.Contains(out, "Tarea") {
		t.Fatalf("al volver, el formulario conserva la tarea:\n%s", out)
	}
	send(e.m, keyEnter)
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Commits de OPS-7") {
		t.Errorf("y se puede seguir de nuevo:\n%s", out)
	}
}
