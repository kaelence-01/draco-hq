package tui

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/history"
)

// toRemove opens the deletion of the newest document, OPS-12, from the home.
func toRemove(t *testing.T, e *historyEnv) {
	t.Helper()
	send(e.m, keyTab)
	typeText(e.m, "x")
	if e.m.top().title() != "Borrar OPS-12" {
		t.Fatalf("x abre la confirmación del elegido: %q\n%s", e.m.top().title(), view(e.m))
	}
}

func TestRemoveDeletesUnlinksAndForgetsAfterTheTaskIsTyped(t *testing.T) {
	e := newHistoryApp(t, recorded(), 160, 40).withProfile()
	toRemove(t, e)
	out := view(e.m)
	for _, want := range []string{"Borrar el documento de OPS-12", "Documento de Despliegue - OPS-12 - Pagos parciales", "https://example.test/pages/12",
		"va a la papelera de Confluence", "se quita desde la tarea OPS-12", "Escribe OPS-12 para confirmar"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q antes de confirmar:\n%s", want, out)
		}
	}

	typeText(e.m, "ops-12")
	send(e.m, keyEnter)
	out = view(e.m)
	if !reflect.DeepEqual(e.docs.deleted, []string{"12"}) || !reflect.DeepEqual(e.tracker.unlinked, []string{"OPS-12 → 12"}) {
		t.Fatalf("borra el documento y quita su enlace: %v %v", e.docs.deleted, e.tracker.unlinked)
	}
	for _, want := range []string{"✓ Documento borrado", "✓ Enlace quitado desde OPS-12", "✓ OPS-12: documento borrado", "enter volver"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en el resultado:\n%s", want, out)
		}
	}
	if entries, _ := history.Load(e.path); len(history.OfIssue(entries, "OPS-12")) != 0 || len(entries) != 3 {
		t.Errorf("sale del historial y los demás quedan: %+v", entries)
	}

	send(e.m, keyEnter)
	if out = view(e.m); len(e.m.stack) != 1 || strings.Contains(out, "OPS-12  hoy") || !strings.Contains(out, "▸ OPS-9") {
		t.Errorf("enter vuelve a la principal sin el documento, con el foco en la lista:\n%s", out)
	}
}

func TestRemoveNeedsTheTaskTyped(t *testing.T) {
	e := newHistoryApp(t, recorded(), 160, 40).withProfile()
	toRemove(t, e)
	typeText(e.m, "OPS-1")
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "no coincide con OPS-12") || len(e.docs.deleted) != 0 {
		t.Errorf("otra tarea no confirma:\n%s", out)
	}
	send(e.m, keyEsc)
	if len(e.m.stack) != 1 || len(e.docs.deleted) != 0 || len(e.tracker.unlinked) != 0 {
		t.Errorf("esc vuelve sin borrar: %d pantallas, %v %v", len(e.m.stack), e.docs.deleted, e.tracker.unlinked)
	}
}

func TestRemoveReportsALinkThatStays(t *testing.T) {
	e := newHistoryApp(t, recorded(), 160, 40).withProfile()
	e.tracker.unlinkErr = errors.New("sin permisos (403)")
	toRemove(t, e)
	typeText(e.m, "OPS-12")
	send(e.m, keyEnter)
	out := squash(view(e.m))
	for _, want := range []string{"✓ Documento borrado", "⚠ No se pudo quitar el enlace desde OPS-12: sin permisos (403)",
		"consolidado QA la dará por documentada", "⚠ OPS-12: documento borrado; el enlace sigue en la tarea"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, view(e.m))
		}
	}
}

func TestRemoveKeepsEverythingWhenTheDocumentStays(t *testing.T) {
	e := newHistoryApp(t, recorded(), 160, 40).withProfile()
	e.docs.err = errors.New("sin permisos (403)")
	toRemove(t, e)
	typeText(e.m, "OPS-12")
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "✗ sin permisos (403)") || !strings.Contains(out, "Escribe OPS-12 para confirmar") {
		t.Errorf("el error se muestra y se puede intentar de nuevo:\n%s", out)
	}
	if entries, _ := history.Load(e.path); len(e.tracker.unlinked) != 0 || len(entries) != 4 {
		t.Errorf("sin borrar, el enlace y el historial quedan: %v %d", e.tracker.unlinked, len(entries))
	}
}

func TestRemoveRefusesAnotherDestination(t *testing.T) {
	e := newHistoryApp(t, recorded(), 160, 40).withProfile()
	e.bundle.Profile.Docs.Provider = "markdown"
	toRemove(t, e)
	typeText(e.m, "OPS-12")
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "su perfil ahora publica en markdown") || len(e.docs.deleted) != 0 {
		t.Errorf("si el perfil publica en otro destino, no se borra:\n%s", view(e.m))
	}
}

func TestRemoveFromHistorialFitsNarrowTerminals(t *testing.T) {
	e := newHistoryApp(t, recorded(), 80, 24).withProfile()
	send(e.m, press('7', "7"))
	typeText(e.m, "x")
	lines := strings.Split(view(e.m), "\n")
	assertFits(t, lines, 80, 24)
	if e.m.top().title() != "Borrar OPS-12" || !strings.Contains(strings.Join(lines, "\n"), "Escribe OPS-12 para confirmar") {
		t.Errorf("x en Historial abre la confirmación:\n%s", strings.Join(lines, "\n"))
	}
	typeText(e.m, "OPS-12")
	send(e.m, keyEnter)
	send(e.m, keyEnter)
	if out := view(e.m); e.m.top().title() != "Historial" || strings.Contains(out, "OPS-12  hoy") {
		t.Errorf("vuelve a Historial sin el documento:\n%s", out)
	}
}

func TestBarOpensTheRemovalOfATask(t *testing.T) {
	e := newHistoryApp(t, recorded(), 160, 40).withProfile()
	runBar(e.m, "history rm -i ops-9")
	if titles := []string{e.m.stack[len(e.m.stack)-2].title(), e.m.top().title()}; !reflect.DeepEqual(titles, []string{"Historial", "Borrar OPS-9"}) {
		t.Fatalf("/history rm -i abre Historial y la confirmación de ese documento: %v", titles)
	}
	send(e.m, keyEsc)
	if out := view(e.m); !strings.Contains(out, "▸ OPS-9") {
		t.Errorf("esc deja Historial con el documento elegido:\n%s", out)
	}

	runBar(e.m, "history rm -i OPS-99")
	if out := view(e.m); len(e.m.stack) != 1 || !strings.Contains(out, "OPS-99 no tiene documentos en el historial") {
		t.Errorf("una tarea sin documentos no abre nada:\n%s", out)
	}
}
