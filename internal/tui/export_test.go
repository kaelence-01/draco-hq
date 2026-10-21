package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/profile"
	"github.com/kaelence-01/draco-hq/internal/provider"
	"github.com/kaelence-01/draco-hq/internal/provider/markdown"
)

// waitingDocs blocks every search until its step is canceled.
type waitingDocs struct {
	provider.DocStore
	canceled chan struct{}
}

func (w waitingDocs) Search(ctx context.Context, _ string) ([]provider.DocRef, error) {
	<-ctx.Done()
	close(w.canceled)
	return nil, ctx.Err()
}

type exportEnv struct {
	m      *app
	dir    string // the Markdown folder searched
	cwd    string
	store  provider.DocStore
	opened []profile.Options
}

const deployDoc = "Documento de Despliegue - OPS-7 - Filtro"

func newExportApp(t *testing.T, docs ...string) *exportEnv {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	e := &exportEnv{dir: filepath.Join(t.TempDir(), "despliegues"), cwd: t.TempDir()}
	t.Chdir(e.cwd)
	store, err := markdown.New(e.dir)
	if err != nil {
		t.Fatal(err)
	}
	e.store = store
	for i, title := range docs {
		ref := provider.DocRef{ID: "x.md"}
		if _, err := store.Create(context.Background(), ref, provider.Document{Title: title, Markdown: "## A considerar:\n\n- [x] Respaldo de OPS-7 " + strings.Repeat("·", i) + "\n"}); err != nil {
			t.Fatal(err)
		}
	}
	cfg := &config.Config{
		DefaultProfile: "local",
		Profiles: map[string]*config.Profile{"local": {
			Tracker: config.ProviderConfig{Provider: "jira"}, Docs: config.ProviderConfig{Provider: "markdown", Path: e.dir},
		}},
		AtlassianEmail: "tu@example.test", AtlassianToken: "x", BaseURL: "https://example.test",
	}
	e.m = newApp(Deps{
		Version:    "v9.9.9",
		LoadConfig: func() (*config.Config, error) { return cfg, nil },
		OpenProfile: func(c *config.Config, o profile.Options) (*profile.Bundle, error) {
			e.opened = append(e.opened, o)
			return &profile.Bundle{Name: "local", Profile: *c.Profiles["local"], Docs: e.store}, nil
		},
	})
	e.m.Update(tea.WindowSizeMsg{Width: 200, Height: 32})
	for range 3 {
		send(e.m, keyDown)
	}
	send(e.m, keyEnter)
	return e
}

// search fills the form with issue and the default scope.
func (e *exportEnv) search(t *testing.T, issue string) {
	t.Helper()
	typeText(e.m, issue)
	send(e.m, keyEnter)
	send(e.m, keyEnter)
}

func (e *exportEnv) defaultPath() string {
	return filepath.Join(e.cwd, "OPS-7_Documento_de_Despliegue_OPS_7_Filtro.txt")
}

func TestExportFormAsksTaskAndScope(t *testing.T) {
	e := newExportApp(t)
	out := view(e.m)
	for _, want := range []string{"Inicio › Exportar página", "Tarea", "Space o carpeta", "Vacío: el del perfil"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "falta la clave de la tarea") {
		t.Errorf("la tarea es obligatoria:\n%s", out)
	}
}

func TestExportFromStartToEnd(t *testing.T) {
	e := newExportApp(t, deployDoc, "Consolidado de Pruebas QA - Pagos - Sprint 12")
	e.search(t, "ops-7")
	if len(e.opened) != 1 || !e.opened[0].AnyScope || e.opened[0].Scope != "" || e.opened[0].Name != "local" {
		t.Errorf("el perfil se abre sin exigir space, como draco f: %+v", e.opened)
	}
	out := view(e.m)
	for _, want := range []string{"Inicio › Exportar página › Documentos de OPS-7", "2 documentos mencionan OPS-7 en markdown · " + e.dir, deployDoc, "Consolidado de Pruebas QA"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}

	send(e.m, press('/', "/"))
	typeText(e.m, "despliegue")
	send(e.m, keyEnter)
	if out := view(e.m); strings.Contains(out, "Consolidado de Pruebas QA") {
		t.Errorf("el filtro deja solo lo que coincide:\n%s", out)
	}
	send(e.m, keyEnter)
	out = view(e.m)
	for _, want := range []string{"› Exportar", deployDoc, "caracteres", "- [x] Respaldo", "FIN DEL DOCUMENTO", "Guardar en", "OPS-7_Documento_de_Despliegue", "enter guardar"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}

	send(e.m, keyEnter)
	data, err := os.ReadFile(e.defaultPath())
	if err != nil {
		t.Fatalf("se guarda en la carpeta actual con el nombre sugerido: %v\n%s", err, view(e.m))
	}
	for _, want := range []string{"OPS-7 - " + deployDoc, "Fuente: Markdown local", "- [x] Respaldo", "FIN DEL DOCUMENTO"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("falta %q en el archivo:\n%s", want, data)
		}
	}
	out = view(e.m)
	if !strings.Contains(out, "✓ Exportado") || !strings.Contains(out, "$ draco f -i OPS-7 -o "+e.defaultPath()) {
		t.Errorf("el resultado da la ruta y el comando equivalente:\n%s", out)
	}
	send(e.m, keyEsc)
	if out := view(e.m); strings.Contains(out, "›") || !strings.Contains(out, "Exportar página") {
		t.Errorf("al terminar, esc vuelve a la principal:\n%s", out)
	}
}

func TestExportWithOneDocumentSkipsTheList(t *testing.T) {
	e := newExportApp(t, deployDoc)
	e.search(t, "OPS-7")
	if out := view(e.m); !strings.Contains(out, "Inicio › Exportar página › Documentos de OPS-7 › Exportar") || !strings.Contains(out, "Guardar en") {
		t.Errorf("con un solo documento se exporta sin preguntar:\n%s", out)
	}
}

func TestExportToAnotherPath(t *testing.T) {
	e := newExportApp(t, deployDoc)
	e.search(t, "OPS-7")
	path := filepath.Join(t.TempDir(), "mi despliegue?.txt")
	send(e.m, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	typeText(e.m, path)
	send(e.m, keyEnter)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("se guarda en la ruta escrita, con su ? (no abre la ayuda): %v\n%s", err, view(e.m))
	}
	if out := view(e.m); !strings.Contains(out, `-o "`+path+`"`) {
		t.Errorf("una ruta con espacios va entre comillas en el comando:\n%s", out)
	}
	if _, err := os.Stat(e.defaultPath()); err == nil {
		t.Error("no se guarda también en la ruta sugerida")
	}
}

func TestExportAsksBeforeReplacing(t *testing.T) {
	e := newExportApp(t, deployDoc)
	if err := os.WriteFile(e.defaultPath(), []byte("viejo"), 0644); err != nil {
		t.Fatal(err)
	}
	e.search(t, "OPS-7")
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "⚠ Ya existe") || !strings.Contains(out, "s reemplazar") {
		t.Fatalf("pide confirmar antes de reemplazar:\n%s", out)
	}
	send(e.m, keyEsc)
	if out := view(e.m); !strings.Contains(out, "Guardar en") || !strings.Contains(out, "› Exportar") {
		t.Fatalf("esc vuelve a la ruta sin salir de la pantalla:\n%s", out)
	}
	if data, _ := os.ReadFile(e.defaultPath()); string(data) != "viejo" {
		t.Fatal("sin confirmar no se reemplaza")
	}
	send(e.m, keyEnter)
	send(e.m, press('s', "s"))
	if data, _ := os.ReadFile(e.defaultPath()); !strings.Contains(string(data), "FIN DEL DOCUMENTO") {
		t.Errorf("s reemplaza el archivo:\n%s", data)
	}
}

func TestExportWithoutDocuments(t *testing.T) {
	e := newExportApp(t, deployDoc)
	e.search(t, "OPS-9")
	out := view(e.m)
	if !strings.Contains(out, "no se encontraron documentos que mencionen el issue: OPS-9") || !strings.Contains(out, "se buscó en markdown") {
		t.Errorf("explica la búsqueda vacía y dónde se buscó:\n%s", out)
	}
}

func TestExportEscCancelsTheSearch(t *testing.T) {
	e := newExportApp(t)
	w := waitingDocs{DocStore: e.store, canceled: make(chan struct{})}
	e.store = w
	e.search(t, "OPS-7")
	if out := view(e.m); !strings.Contains(out, "Buscando documentos que mencionen OPS-7") {
		t.Fatalf("la búsqueda corre con su spinner:\n%s", out)
	}
	send(e.m, keyEsc)
	<-w.canceled
	if out := view(e.m); !strings.Contains(out, "Space o carpeta") {
		t.Errorf("esc cancela y vuelve al formulario:\n%s", out)
	}
}

func TestExportFitsIn80Columns(t *testing.T) {
	e := newExportApp(t, deployDoc)
	e.search(t, "OPS-7")
	e.m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	out := view(e.m)
	for _, l := range strings.Split(out, "\n") {
		if w := ansi.StringWidth(l); w > 80 {
			t.Errorf("línea de %d columnas: %q", w, l)
		}
	}
	if !strings.Contains(out, "Filtro.txt") {
		t.Errorf("la ruta pasa a la línea siguiente, no se corta:\n%s", out)
	}
}
