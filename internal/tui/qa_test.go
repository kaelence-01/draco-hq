package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/profile"
	"github.com/kaelence-01/draco-hq/internal/provider"
	"github.com/kaelence-01/draco-hq/internal/provider/markdown"
)

// qaFake has two tasks under review and the review task of OPS-7. With wait,
// checking OPS-8's document waits until its step is canceled.
type qaFake struct {
	*fakeTracker
	email string
	wait  bool
}

func (q *qaFake) Me(context.Context) (*provider.User, error) {
	return &provider.User{ID: "yo", Email: q.email}, nil
}

func (*qaFake) ReviewTasks(context.Context, provider.QAScope) ([]provider.QAIssue, error) {
	return []provider.QAIssue{
		{Key: "OPS-7", URL: "https://example.test/browse/OPS-7", Summary: "Filtro de pagos", PRMerged: true},
		{Key: "OPS-8", URL: "https://example.test/browse/OPS-8", Summary: "Monto máximo", HasCodingErrors: true},
		{Key: "OPS-20", Summary: "Revisión de Tarea - OPS-7"},
	}, nil
}

func (*qaFake) AssignedTasks(context.Context, provider.QAScope, string) ([]provider.QAIssue, error) {
	return []provider.QAIssue{{Key: "OPS-20", Summary: "Revisión de Tarea - OPS-7"}}, nil
}

func (q *qaFake) HasDocumentLink(ctx context.Context, key string, _ []string) (bool, error) {
	if key != "OPS-8" {
		return true, nil
	}
	if q.wait {
		<-ctx.Done()
		close(q.canceled)
		return false, ctx.Err()
	}
	return false, os.ErrPermission
}

func (*qaFake) Observations(context.Context, string) (string, error) {
	return "falta validar el monto", nil
}

type qaEnv struct {
	m       *app
	tracker provider.Tracker
	docs    *markdown.Store
	store   provider.DocStore // what the profile publishes to; docs unless a test replaces it
}

func newQAApp(t *testing.T, tracker provider.Tracker) *qaEnv {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))
	docs, err := markdown.New(filepath.Join(t.TempDir(), "qa"))
	if err != nil {
		t.Fatal(err)
	}
	e := &qaEnv{tracker: tracker, docs: docs, store: docs}
	cfg := &config.Config{
		QAEmail: "qa@example.test",
		QAJira: &config.QAJiraConfig{
			Project: "OPS", Statuses: []string{"10001"},
			Fields: config.QAJiraFields{CodingErrors: "customfield_1", DevReturns: "customfield_2", DevStatus: "customfield_3"},
		},
		DefaultProfile: "local",
		Profiles: map[string]*config.Profile{"local": {
			Tracker: config.ProviderConfig{Provider: "jira"}, Docs: config.ProviderConfig{Provider: "markdown", Path: "qa"},
		}},
		AtlassianEmail: "tu@example.test", AtlassianToken: "x", BaseURL: "https://example.test",
	}
	e.m = newApp(Deps{
		Version:    "v9.9.9",
		LoadConfig: func() (*config.Config, error) { return cfg, nil },
		Now:        func() time.Time { return time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC) },
		OpenProfile: func(c *config.Config, o profile.Options) (*profile.Bundle, error) {
			return &profile.Bundle{Name: "local", Profile: *c.Profiles["local"], Tracker: e.tracker, Docs: e.store}, nil
		},
	})
	e.m.Update(tea.WindowSizeMsg{Width: 110, Height: 34})
	send(e.m, keyDown)
	send(e.m, keyDown)
	send(e.m, keyEnter)
	return e
}

func qaTracker(email string) *qaFake {
	return &qaFake{fakeTracker: &fakeTracker{canceled: make(chan struct{})}, email: email}
}

// evaluate answers the form with sprint and module.
func (e *qaEnv) evaluate(sprint, module string) {
	typeText(e.m, sprint)
	send(e.m, keyEnter)
	typeText(e.m, module)
	send(e.m, keyEnter)
}

func TestQAFormValidatesTheSprint(t *testing.T) {
	e := newQAApp(t, qaTracker("qa@example.test"))
	if out := view(e.m); !strings.Contains(out, "Inicio › Consolidado QA") || !strings.Contains(out, "modo Kanban") {
		t.Fatalf("el formulario explica el modo Kanban:\n%s", out)
	}
	typeText(e.m, "doce")
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "el sprint debe ser un número positivo") {
		t.Errorf("el sprint es un número:\n%s", out)
	}
}

func TestQAKanbanEvaluatesEachTask(t *testing.T) {
	e := newQAApp(t, qaTracker("QA@example.test"))
	e.evaluate("", "")
	out := view(e.m)
	flat := squash(out)
	for _, want := range []string{
		"Inicio › Consolidado QA › Evaluar", "Kanban · 07/09 al 21/09/2026", "✓ Cuenta QA@example.test · 2 tareas para revisión · 1 tarea QA propia", "ERR DEV DOC PR",
		"OPS-7 ✓ ✓ ✓ ✓ Filtro de pagos completa", "OPS-8 ✗ ✓ ? ✗ Monto máximo con fallas",
		"⚠ OPS-8: no se pudo verificar el documento de despliegue",
		"OPS-8", "Observaciones: falta validar el monto",
		"No hay un consolidado publicado", "creará", "07/09 al 21/09/2026", "enter vista previa",
	} {
		if !strings.Contains(flat, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "OPS-20   ") {
		t.Errorf("la tarea de revisión no es una fila del reporte:\n%s", out)
	}
	send(e.m, keyUp)
	out = view(e.m)
	for _, want := range []string{"OPS-7 Filtro de pagos", "Revisión QA OPS-20", "ninguno", "enlazado", "aprobado"} {
		if !strings.Contains(squash(out), want) {
			t.Errorf("el detalle sigue al cursor y muestra la tarea de revisión: falta %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "Observaciones: falta validar") || strings.Contains(out, "no se pudo verificar") {
		t.Errorf("el detalle es el de OPS-7, no el de OPS-8:\n%s", out)
	}
}

func TestQASprintFindsTheExistingReport(t *testing.T) {
	e := newQAApp(t, qaTracker("qa@example.test"))
	title := "Consolidado de Pruebas QA - Pagos - Sprint 12"
	if _, err := e.docs.Create(context.Background(), provider.DocRef{ID: "x.md"}, provider.Document{Title: title, Markdown: "## Observación\n\nA mano\n"}); err != nil {
		t.Fatal(err)
	}
	e.evaluate("12", "Pagos")
	out := view(e.m)
	for _, want := range []string{"Sprint 12 · Pagos", "Ya existe", "actualizará", "conservando"} {
		if !strings.Contains(squash(out), want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
}

func TestQAModuleIsRequiredInSprintMode(t *testing.T) {
	e := newQAApp(t, qaTracker("qa@example.test"))
	e.evaluate("12", "")
	if out := view(e.m); !strings.Contains(out, "falta el módulo") || !strings.Contains(out, "Sprint") {
		t.Errorf("el módulo es obligatorio con sprint:\n%s", out)
	}
}

func TestQAExplainsAnotherAccount(t *testing.T) {
	e := newQAApp(t, qaTracker("dev@example.test"))
	e.evaluate("", "")
	out := view(e.m)
	for _, want := range []string{"✗ Verificando la cuenta de QA", "Cuenta autenticada: dev@example.test", "qa_email del config: qa@example.test", "ajusta qa_email en"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
}

func TestQAProfileWithoutQA(t *testing.T) {
	e := newQAApp(t, &fakeTracker{})
	e.evaluate("", "")
	if out := view(e.m); !strings.Contains(out, "no soporta el consolidado QA: usa un perfil con tracker jira") {
		t.Errorf("un tracker sin QA se explica en el formulario:\n%s", out)
	}
}

func TestQARowsArriveLiveAndEscCancels(t *testing.T) {
	q := qaTracker("qa@example.test")
	q.wait = true
	e := newQAApp(t, q)
	e.evaluate("", "")
	out := view(e.m)
	if !strings.Contains(squash(out), "OPS-7 ✓ ✓ ✓ ✓") || !strings.Contains(out, "Evaluando OPS-8 (2 de 2)") {
		t.Fatalf("cada fila aparece al evaluarse, sin esperar a las demás:\n%s", out)
	}
	send(e.m, keyEsc)
	<-q.canceled
	if out := view(e.m); !strings.Contains(out, "Módulo") || strings.Contains(out, "Evaluar") {
		t.Errorf("esc cancela y vuelve al formulario:\n%s", out)
	}
}

func TestQAFitsIn80Columns(t *testing.T) {
	e := newQAApp(t, qaTracker("qa@example.test"))
	e.evaluate("", "")
	e.m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	out := view(e.m)
	for _, l := range strings.Split(out, "\n") {
		if w := ansi.StringWidth(l); w > 80 {
			t.Errorf("línea de %d columnas: %q", w, l)
		}
	}
	if !strings.Contains(out, "OPS-8") || !strings.Contains(out, "enter abre la vista previa") {
		t.Errorf("en 80 columnas cabe la tabla y cómo seguir:\n%s", out)
	}
}

// toPreview evaluates in Kanban mode and opens the preview.
func (e *qaEnv) toPreview(t *testing.T) {
	t.Helper()
	e.evaluate("", "")
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "› Evaluar › Vista previa") {
		t.Fatalf("enter abre la vista previa:\n%s", out)
	}
}

func (e *qaEnv) published(t *testing.T, title string) string {
	t.Helper()
	ref, err := e.docs.FindByTitle(context.Background(), title)
	if err != nil || ref == nil {
		t.Fatalf("no se publicó %q: %v", title, err)
	}
	data, err := os.ReadFile(ref.URL)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

const kanbanTitle = "Consolidado de Pruebas QA - 07/09 al 21/09/2026"

func TestQAPublishesANewReport(t *testing.T) {
	e := newQAApp(t, qaTracker("qa@example.test"))
	e.toPreview(t)
	out := view(e.m)
	if !strings.Contains(out, kanbanTitle) || !strings.Contains(out, "OPS-7") || strings.Contains(out, "tab archivos/documento") {
		t.Errorf("la vista previa muestra el consolidado, sin pestaña de archivos:\n%s", out)
	}
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "› Ubicación") || !strings.Contains(out, "▸ ./ (raíz de la carpeta)") {
		t.Fatalf("al crear se elige la ubicación:\n%s", out)
	}
	send(e.m, keyEnter)
	out = view(e.m)
	for _, want := range []string{"Título   " + kanbanTitle, "Destino  markdown · Crear junto a «./ (raíz de la carpeta)»", "$ draco qa"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Enlace") {
		t.Errorf("el consolidado no se enlaza desde una tarea:\n%s", out)
	}
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "✓ Consolidado creado") {
		t.Fatalf("resultado:\n%s", out)
	}
	if doc := e.published(t, kanbanTitle); !strings.Contains(doc, "OPS-7") || !strings.Contains(doc, "| ? |") {
		t.Errorf("el consolidado publicado trae las tareas evaluadas:\n%s", doc)
	}
	send(e.m, keyEnter)
	if out := view(e.m); strings.Contains(headerLine(out), "›") {
		t.Errorf("al terminar se vuelve a la principal:\n%s", out)
	}
}

func TestQAUpdatesKeepingWhatQAWrote(t *testing.T) {
	e := newQAApp(t, qaTracker("qa@example.test"))
	if _, err := e.docs.Create(context.Background(), provider.DocRef{ID: "x.md"}, provider.Document{Title: kanbanTitle,
		Markdown: "### Observación\n\nNota escrita por QA\n\n### Consideraciones\n\nRevisar en producción\n"}); err != nil {
		t.Fatal(err)
	}
	e.toPreview(t)
	if out := view(e.m); !strings.Contains(out, "✓ Lo escrito a mano en «Observación», «Consideraciones» se conserva.") {
		t.Errorf("la vista previa dice qué se conserva:\n%s", out)
	}
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Actualizar «"+kanbanTitle+"»") {
		t.Fatalf("al actualizar no se elige ubicación:\n%s", out)
	}
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "✓ Consolidado actualizado") {
		t.Fatalf("resultado:\n%s", out)
	}
	doc := e.published(t, kanbanTitle)
	for _, want := range []string{"Nota escrita por QA", "Revisar en producción", "OPS-8"} {
		if !strings.Contains(doc, want) {
			t.Errorf("falta %q en el consolidado actualizado:\n%s", want, doc)
		}
	}
}

func TestQAAsksBeforeLosingASection(t *testing.T) {
	e := newQAApp(t, qaTracker("qa@example.test"))
	if _, err := e.docs.Create(context.Background(), provider.DocRef{ID: "x.md"}, provider.Document{Title: kanbanTitle,
		Markdown: "### Observación\n\nNota escrita por QA\n"}); err != nil {
		t.Fatal(err)
	}
	e.toPreview(t)
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "no tiene «Consideraciones»") || !strings.Contains(out, "Pulsa s para continuar") {
		t.Fatalf("sin la sección, enter no sigue:\n%s", out)
	}
	send(e.m, press('s', "s"))
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "› Publicar") {
		t.Errorf("con s se sigue a publicar:\n%s", out)
	}
}

// noReports is a destination with no previous report nor locations.
type noReports struct{ provider.DocStore }

func (noReports) SearchTitle(context.Context, string) ([]provider.DocRef, error) { return nil, nil }
func (noReports) Locations(context.Context, provider.LocationQuery) (*provider.Locations, error) {
	return &provider.Locations{}, nil
}

func TestQAWithoutAPreviousReport(t *testing.T) {
	e := newQAApp(t, qaTracker("qa@example.test"))
	e.store = noReports{e.docs}
	e.toPreview(t)
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "no se encontró ningún consolidado QA previo") || !strings.Contains(out, "(por ejemplo") || !strings.Contains(out, "«Consolidado de Pruebas QA - 07/09") {
		t.Errorf("sin consolidado previo se explica con el título que falta:\n%s", out)
	}
}
