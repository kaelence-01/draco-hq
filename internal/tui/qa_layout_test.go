package tui

import (
	"context"
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

// qaUnverified has one task whose checks pass except that its document link
// cannot be looked up.
type qaUnverified struct{ *qaFake }

func (qaUnverified) ReviewTasks(context.Context, provider.QAScope) ([]provider.QAIssue, error) {
	return []provider.QAIssue{{Key: "OPS-9", Summary: "Exportar a CSV", PRMerged: true}}, nil
}

func (qaUnverified) AssignedTasks(context.Context, provider.QAScope, string) ([]provider.QAIssue, error) {
	return nil, nil
}

func (qaUnverified) HasDocumentLink(context.Context, string, []string) (bool, error) {
	return false, os.ErrPermission
}

func (e *qaEnv) resize(width, height int) {
	e.m.Update(tea.WindowSizeMsg{Width: width, Height: height})
}

func TestQAEvaluateFillsAWideTerminal(t *testing.T) {
	e := newQAApp(t, qaTracker("qa@example.test"))
	e.evaluate("", "")
	e.resize(160, 40)
	lines := strings.Split(view(e.m), "\n")
	assertFits(t, lines, 160, 40)
	out := strings.Join(lines, "\n")
	for _, want := range []string{
		"✓ Sprint o Kanban  ›  Evaluar  ›  Vista previa  ›  Ubicación  ›  Publicar",
		"2 tareas", "✓ 1 completa", "✗ 1 con fallas", "? 0 sin verificar", "· 0 por evaluar",
		"TAREA", "ERR", "DEV", "DOC", "PR", "RESUMEN", "ESTADO", "completa", "con fallas",
		"CONSOLIDADO", "Consolidado de Pruebas QA - 07/09 al 21/09/2026", "Errores de código", "con errores registrados", "no se pudo verificar",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	if at := col(lineWith(t, lines, "Errores de código"), "Errores de código"); at > 8 {
		t.Errorf("el detalle empieza a la izquierda: columna %d", at)
	}
	if at := col(lineWith(t, lines, "CONSOLIDADO"), "CONSOLIDADO"); at < 90 {
		t.Errorf("el consolidado va a la derecha del detalle: columna %d", at)
	}
	if at := col(lineWith(t, lines, "RESUMEN"), "ESTADO"); at < 120 {
		t.Errorf("la tabla usa todo el ancho: el estado empieza en la columna %d", at)
	}
	if !strings.Contains(bottomEdge(lines), "╰") {
		t.Errorf("las cajas llegan hasta el pie:\n%s", out)
	}
}

func TestQADetailFollowsTheCursor(t *testing.T) {
	e := newQAApp(t, qaTracker("qa@example.test"))
	e.evaluate("", "")
	if out := squash(view(e.m)); !strings.Contains(out, "OPS-8 Monto máximo") || !strings.Contains(out, "✗ Errores de código con errores registrados") ||
		!strings.Contains(out, "✓ Devoluciones ninguna") || !strings.Contains(out, "✗ Pull request sin aprobar") {
		t.Errorf("el detalle empieza en la última tarea evaluada, con cada comprobación:\n%s", out)
	}
	send(e.m, keyUp)
	out := squash(view(e.m))
	for _, want := range []string{"OPS-7 Filtro de pagos", "Revisión QA OPS-20", "✓ Errores de código ninguno", "✓ Documento enlazado", "✓ Pull request aprobado", "Observaciones: ninguna"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	send(e.m, keyDown)
	if out := view(e.m); !strings.Contains(out, "▸ OPS-8") {
		t.Errorf("↓ vuelve a la otra tarea:\n%s", out)
	}
}

func TestQAFailedLookupCountsAsUnverifiedNotAsFailed(t *testing.T) {
	e := newQAApp(t, qaUnverified{qaTracker("qa@example.test")})
	e.evaluate("", "")
	out := squash(view(e.m))
	for _, want := range []string{"1 tarea", "✓ 0 completas", "✗ 0 con fallas", "? 1 sin verificar", "· 0 por evaluar", "OPS-9 ✓ ✓ ? ✓ Exportar a CSV sin verificar", "? Documento no se pudo verificar", "⚠ OPS-9: no se pudo verificar el documento de despliegue"} {
		if !strings.Contains(out, want) {
			t.Errorf("una consulta fallida es evidencia que falta, no una falla: falta %q\n%s", want, view(e.m))
		}
	}
}

func TestQACountersAndStateWhileEvaluating(t *testing.T) {
	q := qaTracker("qa@example.test")
	q.wait = true
	e := newQAApp(t, q)
	e.evaluate("", "")
	e.resize(160, 40)
	out := squash(view(e.m))
	for _, want := range []string{"✓ Sprint o Kanban › Evaluar 1/2 › Vista previa", "2 tareas", "✓ 1 completa", "✗ 0 con fallas", "· 1 por evaluar", "OPS-7 ✓ ✓ ✓ ✓ Filtro de pagos completa", "OPS-8 · · · · Monto máximo", "evaluando…", "Evaluando OPS-8 (2 de 2)…"} {
		if !strings.Contains(strings.ReplaceAll(out, "  ", " "), want) && !strings.Contains(out, want) {
			t.Errorf("falta %q mientras evalúa:\n%s", want, view(e.m))
		}
	}
	send(e.m, keyDown)
	if out := view(e.m); !strings.Contains(out, "▸ OPS-7") {
		t.Errorf("mientras evalúa, el cursor sigue a la tarea evaluada y no se mueve:\n%s", out)
	}
	send(e.m, keyEsc)
	<-q.canceled
}

func TestQAPendingTaskHasNoDetailYet(t *testing.T) {
	q := qaTracker("qa@example.test")
	q.wait = true
	e := newQAApp(t, q)
	e.evaluate("", "")
	e.resize(160, 40)
	e.m.top().(*qaProgress).cursor = 1
	if out := view(e.m); !strings.Contains(out, "OPS-8 Monto máximo") || !strings.Contains(out, "Todavía sin evaluar.") || strings.Contains(out, "con errores registrados") {
		t.Errorf("una tarea sin evaluar no muestra comprobaciones que no se hicieron:\n%s", out)
	}
	send(e.m, keyEsc)
	<-q.canceled
}

func TestQAEvaluateNarrowKeepsCountersAndFits(t *testing.T) {
	e := newQAApp(t, qaTracker("qa@example.test"))
	e.evaluate("", "")
	e.resize(80, 24)
	lines := strings.Split(view(e.m), "\n")
	assertFits(t, lines, 80, 24)
	out := strings.Join(lines, "\n")
	for _, want := range []string{"2 tareas", "✓ 1 completa", "✗ 1 con fallas", "? 0 sin verificar", "Err Dev Doc PR", "OPS-8", "enter abre la vista previa"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "╭") || strings.Contains(out, "CONSOLIDADO") {
		t.Errorf("en 80 columnas no hay cajas:\n%s", out)
	}
}

func TestQAWithoutRoomForTheBoxesFallsBackToTheList(t *testing.T) {
	e := newQAApp(t, qaTracker("qa@example.test"))
	e.evaluate("", "")
	e.resize(110, 17)
	lines := strings.Split(view(e.m), "\n")
	assertFits(t, lines, 110, 17)
	out := strings.Join(lines, "\n")
	if !strings.Contains(out, "OPS-8") || !strings.Contains(out, "2 tareas") || strings.Contains(out, "CONSOLIDADO") {
		t.Errorf("con poca altura, la lista sigue entera y las cajas no se dibujan cortadas:\n%s", out)
	}
}

func TestQAKeysKeepEveryKey(t *testing.T) {
	e := newQAApp(t, qaTracker("qa@example.test"))
	e.evaluate("", "")
	lines := strings.Split(view(e.m), "\n")
	short := lines[len(lines)-1]
	if !strings.Contains(short, "↑↓ elegir") || !strings.Contains(short, "enter vista previa") || !strings.Contains(short, "? más") {
		t.Errorf("el pie compacto: %q", short)
	}
	send(e.m, keyHelp)
	if full := squash(view(e.m)); !strings.Contains(full, "↑/k subir") || !strings.Contains(full, "↓/j bajar") {
		t.Errorf("la ayuda completa lleva las demás:\n%s", full)
	}
}

// col is the column, counted in characters, where sub starts in line.
func col(line, sub string) int {
	i := strings.Index(line, sub)
	if i < 0 {
		return -1
	}
	return len([]rune(line[:i]))
}
