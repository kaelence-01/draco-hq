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
		{Key: "OPS-7", Summary: "Filtro de pagos", PRMerged: true},
		{Key: "OPS-8", Summary: "Monto máximo", HasCodingErrors: true},
		{Key: "OPS-20", Summary: "Revisión de Tarea - OPS-7"},
	}, nil
}

func (*qaFake) AssignedTasks(context.Context, provider.QAScope, string) ([]provider.QAIssue, error) {
	return []provider.QAIssue{{Key: "OPS-20", Summary: "Revisión de Tarea - OPS-7"}}, nil
}

func (q *qaFake) HasDocumentLink(ctx context.Context, key, _ string) (bool, error) {
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
}

func newQAApp(t *testing.T, tracker provider.Tracker) *qaEnv {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", os.Getenv("HOME"))
	docs, err := markdown.New(filepath.Join(t.TempDir(), "qa"))
	if err != nil {
		t.Fatal(err)
	}
	e := &qaEnv{tracker: tracker, docs: docs}
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
			return &profile.Bundle{Name: "local", Profile: *c.Profiles["local"], Tracker: e.tracker, Docs: e.docs}, nil
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
	for _, want := range []string{
		"Inicio › Consolidado QA › Evaluar", "Kanban · 07/09 al 21/09/2026", "✓ Cuenta QA@example.test · 2 tareas para revisión · 1 tarea QA propia", "Err Dev Doc PR",
		"OPS-7   ✓   ✓   ✓   ✓   Filtro de pagos", "OPS-8   ✗   ✓   ?   ✗   Monto máximo",
		"⚠ OPS-8: no se pudo verificar el documento de despliegue",
		"OPS-8", "Observaciones: falta validar el monto",
		"se creará «Consolidado de Pruebas QA - 07/09 al 21/09/2026»", "$ draco qa",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "OPS-20   ") {
		t.Errorf("la tarea de revisión no es una fila del reporte:\n%s", out)
	}
	send(e.m, keyUp)
	if out := view(e.m); !strings.Contains(out, "OPS-7 · revisión OPS-20") {
		t.Errorf("el detalle sigue al cursor y muestra la tarea de revisión:\n%s", out)
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
	for _, want := range []string{"Sprint 12 · Pagos", "Ya existe «" + title + "»: se actualizará conservando lo escrito a mano", "$ draco qa -s 12 -m Pagos"} {
		if !strings.Contains(out, want) {
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
	if !strings.Contains(out, "OPS-7   ✓   ✓   ✓   ✓") || !strings.Contains(out, "Evaluando OPS-8 (2 de 2)") {
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
	if !strings.Contains(out, "OPS-8") || !strings.Contains(out, "draco qa") {
		t.Errorf("en 80 columnas cabe la tabla y el comando:\n%s", out)
	}
}
