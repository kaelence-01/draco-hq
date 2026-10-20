package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

// qaTracker is the fake tracker with the QA capability.
type qaTracker struct {
	fakeTracker
	email string
}

func (q *qaTracker) Me(context.Context) (*provider.User, error) {
	return &provider.User{ID: "yo", Email: q.email}, nil
}

func (*qaTracker) ReviewTasks(context.Context, provider.QAScope) ([]provider.QAIssue, error) {
	return []provider.QAIssue{
		{Key: "OPS-7", URL: "https://example.test/browse/OPS-7", PRMerged: true},
		{Key: "OPS-8", URL: "https://example.test/browse/OPS-8", HasCodingErrors: true},
		{Key: "OPS-20", Summary: "Revisión de Tarea - OPS-7"},
	}, nil
}

func (*qaTracker) AssignedTasks(context.Context, provider.QAScope, string) ([]provider.QAIssue, error) {
	return []provider.QAIssue{{Key: "OPS-20", URL: "https://example.test/browse/OPS-20", Summary: "Revisión de Tarea - OPS-7"}}, nil
}

func (*qaTracker) HasDocumentLink(_ context.Context, key, _ string) (bool, error) {
	if key == "OPS-8" {
		return false, os.ErrPermission
	}
	return true, nil
}

func (*qaTracker) Observations(context.Context, string) (string, error) {
	return "falta validar el monto", nil
}

func qaConfig() *config.Config {
	return &config.Config{
		QAEmail: "qa@example.test",
		QAJira: &config.QAJiraConfig{
			Project: "OPS", Statuses: []string{"10001"},
			Fields: config.QAJiraFields{CodingErrors: "customfield_1", DevReturns: "customfield_2", DevStatus: "customfield_3"},
		},
	}
}

func fixedNow(t *testing.T) {
	prev := now
	t.Cleanup(func() { now = prev })
	now = func() time.Time { return time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC) }
}

func TestQAKanbanPublishes(t *testing.T) {
	fixedNow(t)
	dir := fakeEnvWith(t, qaConfig(), &qaTracker{email: "QA@example.test"})

	// Enter: Kanban; [1] the folder; Enter: confirm.
	code, out, errOut := runWith("\n1\n\n", "qa")
	if code != exitOK {
		t.Fatalf("código %d\n%s\n%s", code, out, errOut)
	}
	for _, want := range []string{"Período: 07/09 al 21/09/2026", "2 tareas para revisión", "1 tareas QA propias", "OPS-8: no se pudo verificar", "Consolidado creado"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en:\n%s", want, out)
		}
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.md"))
	if len(files) != 1 {
		t.Fatalf("esperaba un consolidado, hay %v", files)
	}
	doc, _ := os.ReadFile(files[0])
	if !strings.Contains(string(doc), "# Consolidado de Pruebas QA - 07/09 al 21/09/2026") || !strings.Contains(string(doc), "| ? |") {
		t.Errorf("consolidado:\n%s", doc)
	}
}

func TestQASprintDryRun(t *testing.T) {
	fixedNow(t)
	dir := fakeEnvWith(t, qaConfig(), &qaTracker{email: "qa@example.test"})
	code, out, _ := run("qa", "-s", "12", "-m", "Pagos", "--dry-run")
	if code != exitOK || !strings.Contains(out, "Módulo: Pagos") || !strings.Contains(out, "Consolidado de Pruebas QA - Pagos - Sprint 12") || !strings.Contains(out, "no se publicó nada") {
		t.Fatalf("código %d\n%s", code, out)
	}
	if files, _ := filepath.Glob(filepath.Join(dir, "*.md")); len(files) != 0 {
		t.Errorf("--dry-run no debe escribir: %v", files)
	}
}

func TestQAErrors(t *testing.T) {
	fixedNow(t)
	fakeEnvWith(t, qaConfig(), &qaTracker{email: "dev@example.test"})
	code, _, errOut := run("qa", "-s", "12", "-m", "Pagos")
	if code != exitError || !strings.Contains(errOut, "dev@example.test") || !strings.Contains(errOut, "qa@example.test") {
		t.Errorf("otra cuenta: %d %s", code, errOut)
	}
	if code, _, errOut := run("qa", "-s", "doce"); code != exitUsage || !strings.Contains(errOut, "número positivo") {
		t.Errorf("sprint no numérico: %d %s", code, errOut)
	}

	fakeEnvWith(t, qaConfig(), &fakeTracker{})
	if code, _, errOut := run("qa", "-s", "12", "-m", "Pagos"); code != exitError || !strings.Contains(errOut, "no soporta el consolidado QA") {
		t.Errorf("un tracker sin QA: %d %s", code, errOut)
	}
}
