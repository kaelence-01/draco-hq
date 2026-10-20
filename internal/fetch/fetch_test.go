package fetch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

type fakeDocs struct {
	provider.DocStore

	pages     []provider.DocRef
	searchErr error
	doc       *provider.Exported
	docErr    error

	gotQuery, gotDocID string
}

func (f *fakeDocs) Search(_ context.Context, text string) ([]provider.DocRef, error) {
	f.gotQuery = text
	return f.pages, f.searchErr
}

func (f *fakeDocs) Export(_ context.Context, r provider.DocRef) (*provider.Exported, error) {
	f.gotDocID = r.ID
	return f.doc, f.docErr
}

var ctx = context.Background()

func target() *Target { return &Target{Issue: "OPS-7", SpaceKey: "DEV"} }

func TestResolve(t *testing.T) {
	cfg := &config.Config{ConfluenceSpaceKey: "GLOBAL"}

	tg, err := Resolve(cfg, Request{Issue: " ops-7 "})
	if err != nil {
		t.Fatal(err)
	}
	if tg.Issue != "OPS-7" || tg.SpaceKey != "GLOBAL" {
		t.Errorf("esperaba OPS-7 en GLOBAL, obtuve %+v", tg)
	}

	if tg, _ := Resolve(cfg, Request{Issue: "OPS-7", Space: "FLAG"}); tg.SpaceKey != "FLAG" {
		t.Errorf("el flag debe ganar, obtuve %q", tg.SpaceKey)
	}
	if tg, _ := Resolve(&config.Config{}, Request{Issue: "OPS-7"}); tg.SpaceKey != "" {
		t.Errorf("sin space se busca en todos, obtuve %q", tg.SpaceKey)
	}
	if _, err := Resolve(cfg, Request{Issue: "  "}); err == nil {
		t.Error("sin issue debe fallar")
	}
}

func TestSearch(t *testing.T) {
	docs := &fakeDocs{pages: []provider.DocRef{{ID: "1"}, {ID: "2"}}}
	s := NewSession(docs, target())
	if err := s.Search(ctx); err != nil {
		t.Fatal(err)
	}
	if docs.gotQuery != "OPS-7" || len(s.Pages) != 2 {
		t.Errorf("búsqueda mal armada: %q, %d páginas", docs.gotQuery, len(s.Pages))
	}
}

func TestSearchWithoutPagesIsErrNoPages(t *testing.T) {
	err := NewSession(&fakeDocs{}, target()).Search(ctx)
	if !errors.Is(err, ErrNoPages) || !strings.Contains(err.Error(), "OPS-7") {
		t.Errorf("esperaba ErrNoPages con el issue, obtuve %v", err)
	}
}

func TestSearchPropagatesErrors(t *testing.T) {
	err := NewSession(&fakeDocs{searchErr: errors.New("401")}, target()).Search(ctx)
	if err == nil || errors.Is(err, ErrNoPages) {
		t.Errorf("un fallo de la API no es «sin páginas»: %v", err)
	}
}

func TestExport(t *testing.T) {
	docs := &fakeDocs{doc: &provider.Exported{
		DocRef: provider.DocRef{ID: "42", Title: "Documento de Despliegue - Validación de pagos"},
		Source: "Confluence",
		Space:  "Desarrollo",
		Text:   "Pasos del despliegue",
	}}
	s := NewSession(docs, target())

	e, err := s.Export(ctx, &provider.DocRef{ID: "42"})
	if err != nil {
		t.Fatal(err)
	}
	if docs.gotDocID != "42" {
		t.Errorf("descargó el documento %q", docs.gotDocID)
	}
	if !strings.Contains(e.Text, "OPS-7 - Documento de Despliegue") || !strings.Contains(e.Text, "Pasos del despliegue") {
		t.Errorf("texto inesperado:\n%s", e.Text)
	}
	if e.FileName != "OPS-7_Documento_de_Despliegue_Validacion_de_pa.txt" {
		t.Errorf("nombre de archivo = %q", e.FileName)
	}
}

func TestExportErrors(t *testing.T) {
	s := NewSession(&fakeDocs{docErr: errors.New("404")}, target())
	if _, err := s.Export(ctx, &provider.DocRef{ID: "42"}); err == nil {
		t.Error("un fallo al descargar debe devolverse")
	}
	if _, err := s.Export(ctx, nil); err == nil {
		t.Error("sin página elegida debe fallar")
	}
}

var (
	bar80 = strings.Repeat("=", 80)
	sep80 = strings.Repeat("-", 80)
)

func TestIssueTextWithAllFields(t *testing.T) {
	d := &provider.Exported{
		DocRef:      provider.DocRef{Title: "Titulo", URL: "https://example.com/p", Version: 3},
		Source:      "Confluence",
		Space:       "Space",
		CreatedDate: "2026-10-01",
		UpdatedBy:   "Usuario",
		Text:        "Hola\n",
	}
	want := bar80 + "\nAPP-1 - Titulo\n" + bar80 + "\n" +
		"Fuente: Confluence - Space\nURL: https://example.com/p\n" +
		"Creado: 2026-10-01 | Versión: 3\nÚltima edición: Usuario\n" +
		sep80 + "\n\nHola\n\n" +
		bar80 + "\nFIN DEL DOCUMENTO\n" + bar80 + "\n"

	if got := issueText("APP-1", d); got != want {
		t.Errorf("salida distinta.\ngot:  %q\nwant: %q", got, want)
	}
}

func TestIssueTextOmitsEmptyOptionalLines(t *testing.T) {
	d := &provider.Exported{DocRef: provider.DocRef{Title: "Minimo", URL: "https://example.com/m"}, Source: "Confluence", Space: "Space"}
	want := bar80 + "\nAPP-2 - Minimo\n" + bar80 + "\n" +
		"Fuente: Confluence - Space\nURL: https://example.com/m\n" +
		sep80 + "\n\n" +
		bar80 + "\nFIN DEL DOCUMENTO\n" + bar80 + "\n"

	if got := issueText("APP-2", d); got != want {
		t.Errorf("salida distinta.\ngot:  %q\nwant: %q", got, want)
	}
}

func TestIssueTextVersionNeedsCreatedDate(t *testing.T) {
	d := &provider.Exported{DocRef: provider.DocRef{Title: "T", Version: 5}}
	if got := issueText("APP-3", d); strings.Contains(got, "Versión") {
		t.Errorf("sin fecha de creación no debe imprimir la versión: %q", got)
	}
}

func TestSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "OPS-7.txt")
	if err := (&Export{Text: "contenido"}).Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "contenido" {
		t.Errorf("archivo = %q, %v", got, err)
	}
	if err := (&Export{}).Save(filepath.Join(t.TempDir(), "no", "existe.txt")); err == nil {
		t.Error("un directorio inexistente debe dar error")
	}
}

func TestFileName(t *testing.T) {
	cases := []struct{ title, want string }{
		{"Revisión de Facturación", "OPS-7_Revision_de_Facturacion.txt"},
		{"Año -- nuevo / ¿módulo? ", "OPS-7_Ano_nuevo_modulo.txt"},
		{"¿?¡!", "OPS-7.txt"},
		{strings.Repeat("a", 39) + " b", "OPS-7_" + strings.Repeat("a", 39) + ".txt"},
	}
	for _, c := range cases {
		if got := fileName("OPS-7", c.title); got != c.want {
			t.Errorf("%q → %q, esperaba %q", c.title, got, c.want)
		}
	}
}
