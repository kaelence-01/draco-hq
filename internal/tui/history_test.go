package tui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/history"
	"github.com/kaelence-01/draco-hq/internal/profile"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

var (
	keyTab   = tea.KeyPressMsg{Code: tea.KeyTab}
	keyRight = tea.KeyPressMsg{Code: tea.KeyRight}
	testNow  = time.Date(2026, 11, 9, 12, 0, 0, 0, time.Local)
)

type historyEnv struct {
	m       *app
	path    string
	opened  []string
	openErr error

	// bundle is what opening a profile gives; nil fails as without network.
	bundle  *profile.Bundle
	docs    *fakeDocs
	tracker *fakeTracker
}

// fakeDocs is a destination that only deletes.
type fakeDocs struct {
	provider.DocStore
	deleted []string
	err     error
}

func (d *fakeDocs) Delete(_ context.Context, r provider.DocRef) error {
	d.deleted = append(d.deleted, r.ID)
	return d.err
}

// withProfile makes opening a profile give a Confluence destination and a
// tracker that record what is deleted and unlinked.
func (e *historyEnv) withProfile() *historyEnv {
	e.docs, e.tracker = &fakeDocs{}, &fakeTracker{}
	e.bundle = &profile.Bundle{Name: "equipo", Profile: config.Profile{Tracker: config.ProviderConfig{Provider: "jira"}, Docs: config.ProviderConfig{Provider: "confluence"}},
		Tracker: e.tracker, Docs: e.docs}
	return e
}

// recorded is a history with a document of each kind, the newest first:
// updated today, without its link yesterday, and from a removed project.
func recorded() []history.Entry {
	return []history.Entry{
		{Issue: "OPS-3", Title: "Documento de Despliegue - OPS-3 - Inventario", URL: "https://example.test/pages/3", DocID: "3", Dest: "confluence",
			Project: "inventario", Profile: "equipo", At: time.Date(2025, 12, 1, 9, 0, 0, 0, time.Local)},
		{Issue: "OPS-7", Title: "Documento de Despliegue - OPS-7 - Filtro de pagos", URL: "https://example.test/pages/7", DocID: "7", Dest: "confluence",
			Project: "pagos", Profile: "equipo", At: time.Date(2026, 11, 4, 9, 0, 0, 0, time.Local)},
		{Issue: "OPS-9", Title: "Documento de Despliegue - OPS-9 - Notas de crédito", URL: "https://example.test/pages/9", DocID: "9", Dest: "confluence",
			Project: "facturacion", Profile: "equipo", At: testNow.Add(-20*time.Hour - 30*time.Minute), LinkFailed: true},
		{Issue: "OPS-12", Title: "Documento de Despliegue - OPS-12 - Pagos parciales", URL: "https://example.test/pages/12", DocID: "12", Dest: "confluence",
			Project: "pagos", Profile: "equipo", At: testNow.Add(-time.Hour), Updated: true},
	}
}

func newHistoryApp(t *testing.T, entries []history.Entry, width, height int) *historyEnv {
	t.Helper()
	e := &historyEnv{path: filepath.Join(t.TempDir(), "history.json")}
	for _, x := range entries {
		if err := history.Record(e.path, x); err != nil {
			t.Fatal(err)
		}
	}
	cfg := testConfig()
	cfg.Projects["facturacion"] = &config.ProjectConfig{BackendRepo: "facturacion-api"}
	e.m = newApp(Deps{
		Version:     "v9.9.9",
		LoadConfig:  func() (*config.Config, error) { return cfg, nil },
		Now:         func() time.Time { return testNow },
		HistoryPath: func() (string, error) { return e.path, nil },
		OpenURL: func(u string) error {
			e.opened = append(e.opened, u)
			return e.openErr
		},
		OpenProfile: func(_ *config.Config, o profile.Options) (*profile.Bundle, error) {
			if e.bundle == nil {
				return nil, errors.New("sin red en el test")
			}
			if o.Name != "equipo" || !o.AnyScope {
				t.Errorf("se abre el perfil del documento, sin exigir el space: %+v", o)
			}
			return e.bundle, nil
		},
	})
	e.m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	view(e.m) // the loop draws after the size, and the home learns what fits
	return e
}

func lineIndex(t *testing.T, lines []string, want string) int {
	t.Helper()
	for i, l := range lines {
		if strings.Contains(l, want) {
			return i
		}
	}
	t.Fatalf("ninguna línea tiene %q:\n%s", want, strings.Join(lines, "\n"))
	return -1
}

func TestHomeShowsTheRecentDocumentsUnderTheMenu(t *testing.T) {
	e := newHistoryApp(t, recorded(), 160, 40)
	lines := strings.Split(view(e.m), "\n")
	assertFits(t, lines, 160, 40)
	if lineIndex(t, lines, "RECIENTES") < lineIndex(t, lines, "Historial") {
		t.Error("los recientes van bajo el menú")
	}
	for _, want := range []string{"[Todos]", "facturacion", "inventario", "pagos", "tab recientes"} {
		lineIndex(t, lines, want)
	}
	for _, want := range []string{"OPS-12  hoy 11:00   actualizado pagos", "OPS-9   ayer 15:30  creado      ⚠ sin enlace", "OPS-7   04/11", "OPS-3   01/12/2025"} {
		if !strings.Contains(squash(view(e.m)), squash(want)) {
			t.Errorf("falta la fila %q:\n%s", want, view(e.m))
		}
	}
	if lineIndex(t, lines, "OPS-12") > lineIndex(t, lines, "OPS-9") {
		t.Error("el más reciente va primero")
	}
}

func TestTabGivesTheKeysToTheRecentDocuments(t *testing.T) {
	e := newHistoryApp(t, recorded(), 160, 40)
	send(e.m, keyTab)
	out := view(e.m)
	for _, want := range []string{"▸ OPS-12", "Actualizado el 09/11/2026 11:00", "Documento de Despliegue - OPS-12 - Pagos parciales",
		"https://example.test/pages/12", "$ draco history", "enter abrir", "u actualizar", "←→ proyecto", "tab menú", "e lo exporta"} {
		if !strings.Contains(out, want) {
			t.Errorf("con el foco en los recientes falta %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Lee la tarea") {
		t.Errorf("el panel muestra el documento, no la función del menú:\n%s", out)
	}
	send(e.m, keyHelp)
	if out = view(e.m); !strings.Contains(squash(out), "e exportar") {
		t.Errorf("la ayuda completa tiene e:\n%s", out)
	}
	send(e.m, keyHelp)

	send(e.m, keyDown)
	if out = view(e.m); !strings.Contains(squash(out), "La tarea no enlaza este documento") {
		t.Errorf("el detalle avisa del enlace fallido:\n%s", out)
	}

	send(e.m, keyRight)
	out = view(e.m)
	if !strings.Contains(out, "[facturacion]") || !strings.Contains(out, "$ draco history -p facturacion") || strings.Contains(out, "OPS-12") {
		t.Errorf("→ filtra por el siguiente proyecto:\n%s", out)
	}
	if !strings.Contains(out, "OPS-9") || !strings.Contains(out, "Notas de crédito") {
		t.Errorf("filtrado, la fila muestra el título en vez del proyecto:\n%s", out)
	}

	send(e.m, keyTab)
	if out = view(e.m); !strings.Contains(out, "Lee la tarea") || strings.Contains(out, "▸ OPS-9") {
		t.Errorf("tab vuelve al menú y a su panel:\n%s", out)
	}
}

func TestEnterOpensTheDocument(t *testing.T) {
	e := newHistoryApp(t, recorded(), 160, 40)
	send(e.m, keyTab)
	send(e.m, keyEnter)
	if len(e.opened) != 1 || e.opened[0] != "https://example.test/pages/12" || len(e.m.stack) != 1 {
		t.Fatalf("enter abre la dirección del documento sin salir de la principal: %v", e.opened)
	}
	if out := view(e.m); !strings.Contains(out, "✓ Abriendo OPS-12: https://example.test/pages/12") {
		t.Errorf("el pie dice qué se abrió:\n%s", out)
	}

	e.openErr = errors.New("no se encontró con qué abrirlo")
	send(e.m, keyEnter)
	if out := squash(view(e.m)); !strings.Contains(out, "⚠ No se pudo abrir OPS-12: no se encontró con qué abrirlo. Está en https://example.test/pages/12") {
		t.Errorf("si no se puede abrir, se dice y se da la dirección:\n%s", out)
	}
}

func TestUpdateOpensGenerarWithTheTask(t *testing.T) {
	e := newHistoryApp(t, recorded(), 160, 40)
	send(e.m, keyTab)
	typeText(e.m, "u")
	out := view(e.m)
	if e.m.top().title() != "Generar documento" || !strings.Contains(out, "OPS-12") || !strings.Contains(out, "sin red en el test") {
		t.Fatalf("u abre Generar con la tarea y su proyecto, y sigue hasta abrir el perfil:\n%s", out)
	}
	if g := e.m.top().(*generate); g.project != "pagos" {
		t.Errorf("el proyecto es el del documento: %q", g.project)
	}

	send(e.m, homeMsg{})
	for range 3 {
		send(e.m, keyDown)
	}
	typeText(e.m, "u")
	if out = view(e.m); !strings.Contains(out, "El proyecto «inventario» ya no está en la configuración: elige otro") || !strings.Contains(out, "OPS-3") {
		t.Errorf("un proyecto que ya no existe se avisa en el formulario:\n%s", out)
	}
}

func TestExportOpensWithTheTask(t *testing.T) {
	e := newHistoryApp(t, recorded(), 160, 40)
	send(e.m, keyTab)
	typeText(e.m, "e")
	if out := view(e.m); e.m.top().title() != "Exportar página" || !strings.Contains(out, "OPS-12") {
		t.Errorf("e abre Exportar con la tarea:\n%s", out)
	}
}

func TestWithoutRoomTheHomeKeepsTheMenu(t *testing.T) {
	e := newHistoryApp(t, recorded(), 100, 22)
	out := view(e.m)
	if strings.Contains(out, "RECIENTES") || strings.Contains(out, "tab recientes") {
		t.Errorf("sin alto para tres documentos, la principal es la de siempre:\n%s", out)
	}
	send(e.m, keyTab)
	send(e.m, keyEnter)
	if e.m.top().title() != "Generar documento" {
		t.Errorf("sin la caja, tab no le da las teclas y enter abre el menú: %q", e.m.top().title())
	}
	if e := newHistoryApp(t, recorded(), 80, 40); strings.Contains(view(e.m), "RECIENTES") {
		t.Error("en 80 columnas no hay caja: se llega por Historial")
	}
}

func TestEmptyAndDamagedHistory(t *testing.T) {
	e := newHistoryApp(t, nil, 160, 40)
	if out := view(e.m); !strings.Contains(out, "Todavía no hay documentos: se agregan al publicar") || !strings.Contains(out, "Generar o con draco g.") {
		t.Errorf("sin documentos, la caja dice cómo llegan:\n%s", view(e.m))
	}

	if err := os.WriteFile(e.path, []byte("{no es json"), 0o600); err != nil {
		t.Fatal(err)
	}
	send(e.m, press('7', "7"))
	if out := squash(view(e.m)); !strings.Contains(out, "⚠ el historial está dañado") {
		t.Errorf("un historial dañado se avisa:\n%s", view(e.m))
	}
}

func TestHistoryScreen(t *testing.T) {
	e := newHistoryApp(t, recorded(), 110, 34)
	send(e.m, press('7', "7"))
	lines := strings.Split(view(e.m), "\n")
	assertFits(t, lines, 110, 34)
	if e.m.top().title() != "Historial" {
		t.Fatalf("7 abre Historial: %q", e.m.top().title())
	}
	out := strings.Join(lines, "\n")
	for _, want := range []string{"Inicio › Historial", "DOCUMENTOS PUBLICADOS", "[Todos]", "▸ OPS-12", "OPS-3", "Actualizado el 09/11/2026 11:00", "$ draco history"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q:\n%s", want, out)
		}
	}
	if at := strings.Index(lineWith(t, lines, "Actualizado el"), "Actualizado"); at < 50 {
		t.Errorf("el detalle va a la derecha de la lista: columna %d", at)
	}

	for _, size := range [][2]int{{80, 24}, {60, 20}} {
		e := newHistoryApp(t, recorded(), size[0], size[1])
		send(e.m, press('7', "7"))
		lines := strings.Split(view(e.m), "\n")
		assertFits(t, lines, size[0], size[1])
		out := strings.Join(lines, "\n")
		if !strings.Contains(out, "▸ OPS-12") || !strings.Contains(out, "Actualizado el") {
			t.Errorf("en %dx%d, la lista y bajo ella el detalle:\n%s", size[0], size[1], out)
		}
	}
}

func TestBarOpensTheHistoryOfAProject(t *testing.T) {
	e := newHistoryApp(t, recorded(), 160, 40)
	runBar(e.m, "history -p facturacion")
	out := view(e.m)
	if e.m.top().title() != "Historial" || !strings.Contains(out, "[facturacion]") || strings.Contains(out, "OPS-12") {
		t.Errorf("/history -p abre Historial en ese proyecto:\n%s", out)
	}
	runBar(e.m, "history -p inventario")
	if out = view(e.m); !strings.Contains(out, "[inventario]") {
		t.Errorf("un proyecto que ya no está en la config sigue en el historial:\n%s", out)
	}
	runBar(e.m, "history -n 30")
	if out = view(e.m); len(e.m.stack) != 1 || !strings.Contains(out, "--limit no está en el modo visual") {
		t.Errorf("-n es del modo comando:\n%s", out)
	}
}

func TestPublishingRecordsInTheHistory(t *testing.T) {
	e := newGenerateApp(t, nil)
	path := filepath.Join(t.TempDir(), "history.json")
	e.m.st.deps.HistoryPath = func() (string, error) { return path, nil }
	e.toPreview(t, "")
	send(e.m, keyEnter)
	send(e.m, keyEnter)
	send(e.m, keyEnter)
	entries, err := history.Load(path)
	if err != nil || len(entries) != 1 || entries[0].Issue != "OPS-7" || entries[0].Project != "pagos" || entries[0].Profile != "local" || entries[0].Dest != "markdown" {
		t.Fatalf("publicar deja el documento en el historial: %+v %v", entries, err)
	}
	send(e.m, keyEnter)
	if out := view(e.m); len(e.m.stack) != 1 || !strings.Contains(out, "RECIENTES") || !strings.Contains(out, "OPS-7") {
		t.Errorf("al volver al inicio, el documento está en los recientes:\n%s", out)
	}
}

func TestAHistoryThatCannotBeSavedIsReported(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.m.st.deps.HistoryPath = func() (string, error) { return "", errors.New("sin carpeta de configuración") }
	e.toPreview(t, "")
	send(e.m, keyEnter)
	send(e.m, keyEnter)
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "✓ Documento creado") || !strings.Contains(out, "⚠ No se guardó en el historial: sin carpeta de configuración") {
		t.Errorf("el documento se publica y el historial se avisa:\n%s", out)
	}
}

func TestWhen(t *testing.T) {
	for at, want := range map[time.Time]string{
		testNow.Add(-time.Hour):                          "hoy 11:00",
		time.Date(2026, 11, 9, 0, 0, 0, 0, time.Local):   "hoy 00:00",
		time.Date(2026, 11, 8, 23, 59, 0, 0, time.Local): "ayer 23:59",
		time.Date(2026, 11, 7, 23, 59, 0, 0, time.Local): "07/11",
		time.Date(2025, 12, 31, 10, 0, 0, 0, time.Local): "31/12/2025",
	} {
		if got := when(at, testNow); got != want {
			t.Errorf("when(%v) = %q, se esperaba %q", at, got, want)
		}
	}
}
