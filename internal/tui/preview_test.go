package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestPreviewDrawsTheDocumentAndMToggleTheMarkdown(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toPreview(t, "")
	out := view(e.m)
	for _, want := range []string{"▸ Documento (renderizado)", "m muestra el Markdown", "Épica", "Arquitecturas e interfaces", "FiltroController.php", "Pasar backend al servidor"} {
		if !strings.Contains(out, want) {
			t.Errorf("por defecto se ve el documento dibujado: falta %q\n%s", want, out)
		}
	}
	for _, hidden := range []string{"<!--", "draco:", "|---|", "**", "## "} {
		if strings.Contains(out, hidden) {
			t.Errorf("el documento dibujado no muestra %q:\n%s", hidden, out)
		}
	}

	send(e.m, press('m', "m"))
	out = view(e.m)
	for _, want := range []string{"▸ Documento (Markdown)", "m muestra el documento renderizado", "<!-- draco:tabla", "|---|", "## Arquitecturas e interfaces"} {
		if !strings.Contains(out, want) {
			t.Errorf("m muestra el Markdown tal como se publicará: falta %q\n%s", want, out)
		}
	}
	send(e.m, press('m', "m"))
	if out := view(e.m); !strings.Contains(out, "▸ Documento (renderizado)") || strings.Contains(out, "<!--") {
		t.Errorf("m otra vez vuelve al documento dibujado:\n%s", out)
	}
}

func TestPreviewMOnlyAppliesToTheDocument(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toPreview(t, "")
	send(e.m, tea.KeyPressMsg{Code: tea.KeyTab})
	send(e.m, press('m', "m"))
	out := view(e.m)
	if !strings.Contains(out, "▸ Archivos") || !strings.Contains(out, "Documento (renderizado)") || strings.Contains(out, "m muestra") {
		t.Errorf("en la pestaña de archivos m no hace nada y no se ofrece:\n%s", out)
	}
	send(e.m, tea.KeyPressMsg{Code: tea.KeyTab})
	if out := view(e.m); !strings.Contains(out, "▸ Documento (renderizado)") || !strings.Contains(out, "FiltroController.php") {
		t.Errorf("m en archivos no cambió el modo del documento:\n%s", out)
	}
}

func TestPreviewKeysKeepMInTheFullHelp(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toPreview(t, "")
	lines := strings.Split(view(e.m), "\n")
	short := lines[len(lines)-1]
	if !strings.Contains(short, "tab archivos/documento") || !strings.Contains(short, "? más") || strings.Contains(short, "Markdown") {
		t.Errorf("el pie compacto lleva tab, desplazar y continuar, y m va en la ayuda completa: %q", short)
	}
	send(e.m, keyHelp)
	if full := squash(view(e.m)); !strings.Contains(full, "m Markdown/renderizado") {
		t.Errorf("la ayuda completa lleva m:\n%s", full)
	}
}

func TestPreviewShowsTheStepsAndWhatWillHappen(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toPreview(t, "")
	e.resize(160, 40)
	lines := strings.Split(view(e.m), "\n")
	assertFits(t, lines, 160, 40)
	title := lineWith(t, lines, newTitle)
	if !strings.Contains(title, "✓ Tarea  ›  ✓ Commits  ›  Vista previa  ›  Ubicación  ›  Publicar") {
		t.Errorf("con sitio, los pasos van en la línea del título: %q", title)
	}
	if status := lineWith(t, lines, "Se creará un documento nuevo."); !strings.HasPrefix(strings.TrimSpace(status), "✓ Se creará") {
		t.Errorf("dice qué pasará al publicar: %q", status)
	}

	e.resize(110, 34)
	lines = strings.Split(view(e.m), "\n")
	assertFits(t, lines, 110, 34)
	status := lineWith(t, lines, "Se creará un documento nuevo.")
	if strings.Contains(lineWith(t, lines, newTitle), "Publicar") || !strings.Contains(status, "Vista previa  ›  Ubicación  ›  Publicar") {
		t.Errorf("si el título no deja sitio, los pasos pasan a la línea de estado:\n%s", strings.Join(lines, "\n"))
	}

	e.resize(80, 24)
	lines = strings.Split(view(e.m), "\n")
	assertFits(t, lines, 80, 24)
	if out := strings.Join(lines, "\n"); strings.Contains(out, "Ubicación  ›") || !strings.Contains(out, "Se creará un documento nuevo.") {
		t.Errorf("en 80 columnas no hay línea de pasos:\n%s", out)
	}
}

func TestPreviewDrawsAgainWhenTheTerminalChanges(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toPreview(t, "")
	for _, size := range [][2]int{{160, 40}, {80, 24}, {110, 34}} {
		e.resize(size[0], size[1])
		lines := strings.Split(view(e.m), "\n")
		assertFits(t, lines, size[0], size[1])
		if out := strings.Join(lines, "\n"); !strings.Contains(out, "Épica") || strings.Contains(out, "<!--") {
			t.Errorf("%dx%d: el documento se dibuja para el ancho nuevo:\n%s", size[0], size[1], out)
		}
	}
}

func TestQAPreviewIsDrawnToo(t *testing.T) {
	e := newQAApp(t, qaTracker("qa@example.test"))
	e.toPreview(t)
	e.m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	out := view(e.m)
	for _, want := range []string{"▸ Documento (renderizado)", kanbanTitle, "Tabla Consolidada", "OPS-7", "OPS-8", "Se creará un consolidado nuevo.", "✓ Sprint o Kanban  ›  ✓ Evaluar  ›  Vista previa"} {
		if !strings.Contains(out, want) {
			t.Errorf("el consolidado se ve dibujado, con los pasos de QA: falta %q\n%s", want, out)
		}
	}
	for _, hidden := range []string{"<!--", "|---|", "**"} {
		if strings.Contains(out, hidden) {
			t.Errorf("el consolidado dibujado no muestra %q:\n%s", hidden, out)
		}
	}
	if strings.Contains(out, "Archivos") {
		t.Errorf("QA no tiene pestaña de archivos:\n%s", out)
	}
	send(e.m, press('m', "m"))
	if out := view(e.m); !strings.Contains(out, "<!-- draco:tabla") || !strings.Contains(out, "| Tarea |") {
		t.Errorf("m también muestra el Markdown del consolidado:\n%s", out)
	}
}
