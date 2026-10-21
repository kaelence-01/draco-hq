package tui

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

const oldTitle = "Documento de Despliegue - OPS-7 - Título viejo"

// toPreview marks the newest backend commit of OPS-7 and opens the preview;
// choice answers an existing document.
func (e *generateEnv) toPreview(t *testing.T, choice string) {
	t.Helper()
	e.toCommits(t, "OPS-7")
	send(e.m, keySpace)
	send(e.m, keyEnter)
	typeText(e.m, choice)
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Vista previa") {
		t.Fatalf("enter abre la vista previa:\n%s", out)
	}
}

func (e *generateEnv) existing(t *testing.T, markdown string) {
	t.Helper()
	if _, err := e.docs.Create(context.Background(), provider.DocRef{ID: "x.md"}, provider.Document{Title: oldTitle, Markdown: markdown}); err != nil {
		t.Fatal(err)
	}
}

func (e *generateEnv) published(t *testing.T, title string) string {
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

const newTitle = "Documento de Despliegue - OPS-7 - Filtro de pagos"

func TestPreviewShowsTheDocumentAndPublishesNothing(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toPreview(t, "")
	out := view(e.m)
	for _, want := range []string{newTitle, "▸ Documento", "FiltroController.php", "tab archivos/documento"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	send(e.m, tea.KeyPressMsg{Code: tea.KeyTab})
	if out := view(e.m); !strings.Contains(out, "▸ Archivos") || !strings.Contains(out, "Backend · pagos-api") {
		t.Errorf("tab muestra los archivos:\n%s", out)
	}
	send(e.m, keyEsc)
	if ref, _ := e.docs.FindByTitle(context.Background(), newTitle); ref != nil || len(e.tracker.linked) > 0 {
		t.Error("salir de la vista previa no publica nada")
	}
}

func TestCreateFromStartToEnd(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toPreview(t, "")
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Ubicación") || !strings.Contains(out, "▸ ./ (raíz de la carpeta)") {
		t.Fatalf("al crear se elige la ubicación:\n%s", out)
	}
	send(e.m, keyEnter)
	out := view(e.m)
	for _, want := range []string{"Inicio › Generar documento › Commits de OPS-7 › Preparar documento › Vista previa › Ubicación › Publicar",
		"Crear junto a «./ (raíz de la carpeta)»", "desde OPS-7 en jira", "$ draco g -i OPS-7 -b a1b2c3d -p pagos", "enter publicar"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "✓ Documento creado") || !strings.Contains(out, ".md") {
		t.Fatalf("el resultado tiene la ubicación del documento:\n%s", out)
	}
	if body := e.published(t, newTitle); !strings.Contains(body, "FiltroController.php") {
		t.Errorf("el documento tiene los archivos de los commits:\n%s", body)
	}
	if strings.Join(e.tracker.linked, "") != "OPS-7 → "+newTitle {
		t.Errorf("el documento se enlaza desde la tarea: %v", e.tracker.linked)
	}
	send(e.m, keyEsc)
	if len(e.m.stack) != 1 {
		t.Errorf("al terminar, esc vuelve al inicio: pila %d", len(e.m.stack))
	}
}

func TestPublishingCannotBeLeft(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.tracker.linkWait = make(chan struct{})
	defer close(e.tracker.linkWait)
	e.toPreview(t, "")
	send(e.m, keyEnter)
	send(e.m, keyEnter)
	send(e.m, keyEnter)
	send(e.m, keyEsc)
	if out := view(e.m); !strings.Contains(out, "Publicando…") || len(e.m.stack) != 7 || strings.Contains(out, "esc volver") {
		t.Errorf("mientras publica, esc no sale:\n%s", out)
	}
}

func TestLinkErrorIsReported(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.tracker.linkErr = errors.New("sin permiso (403)")
	e.toPreview(t, "")
	send(e.m, keyEnter)
	send(e.m, keyEnter)
	send(e.m, keyEnter)
	out := view(e.m)
	if !strings.Contains(out, "✓ Documento creado") || !strings.Contains(out, "No se pudo enlazar el documento desde OPS-7: sin permiso (403)") || !strings.Contains(out, "consolidado QA") {
		t.Errorf("un enlace fallido se reporta junto al documento creado:\n%s", out)
	}
}

func TestUpdateKeepsWhatWasWrittenByHand(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.existing(t, "# Viejo\n\n## A considerar:\n\n- [x] Respaldo hecho a mano\n")
	e.toPreview(t, "1")
	if out := view(e.m); !strings.Contains(out, "✓ Lo escrito a mano en «A considerar» se conserva") {
		t.Fatalf("la vista previa dice que se conserva lo escrito a mano:\n%s", out)
	}
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Publicar") || !strings.Contains(out, "Actualizar «"+oldTitle+"»") {
		t.Fatalf("al actualizar no se elige ubicación:\n%s", out)
	}
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "✓ Documento actualizado") {
		t.Fatalf("falta el resultado:\n%s", out)
	}
	body := e.published(t, newTitle)
	if !strings.Contains(body, "Respaldo hecho a mano") || !strings.Contains(body, "FiltroController.php") {
		t.Errorf("la actualización conserva «A considerar» y trae los archivos nuevos:\n%s", body)
	}
}

func TestUpdateThatLosesASectionAsksFirst(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.existing(t, "# Viejo\n")
	e.toPreview(t, "1")
	if out := view(e.m); !strings.Contains(out, "⚠ El documento actual no tiene «A considerar»") {
		t.Fatalf("avisa lo que no se puede conservar:\n%s", out)
	}
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Pulsa s para continuar") || strings.Contains(out, "Publicar") {
		t.Fatalf("enter solo no sigue:\n%s", out)
	}
	typeText(e.m, "s")
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Actualizar «"+oldTitle+"»") {
		t.Errorf("con s se sigue:\n%s", out)
	}
}

func TestCreateAnotherNextToTheExisting(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.existing(t, "# Viejo\n")
	e.toPreview(t, "2")
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Ubicación") {
		t.Errorf("crear otro elige ubicación:\n%s", out)
	}
}

func TestLocationsFilter(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toPreview(t, "")
	send(e.m, keyEnter)
	typeText(e.m, "/zzz")
	if out := view(e.m); !strings.Contains(out, "/ zzz") || !strings.Contains(out, "Ningún documento coincide") {
		t.Errorf("el filtro deja fuera lo que no coincide:\n%s", out)
	}
	send(e.m, keyEnter)
	send(e.m, keyEsc)
	if out := view(e.m); !strings.Contains(out, "▸ ./ (raíz de la carpeta)") || !strings.Contains(out, "Ubicación") {
		t.Errorf("esc quita el filtro sin salir:\n%s", out)
	}
}

// noLocations is a destination without previous deploy documents.
type noLocations struct{ provider.DocStore }

func (noLocations) Locations(context.Context, provider.LocationQuery) (*provider.Locations, error) {
	return &provider.Locations{}, nil
}

func TestNoLocationsIsExplained(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.store = noLocations{e.docs}
	e.toPreview(t, "")
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "no se encontraron documentos de despliegue previos") || !strings.Contains(out, "destino Markdown") {
		t.Errorf("sin ubicaciones se explica qué hacer:\n%s", out)
	}
}
