package fetch

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/atlassian"
	"github.com/kaelence-01/draco-hq/internal/config"
)

type fakeAPI struct {
	pages      []atlassian.Page
	searchErr  error
	content    *atlassian.PageContent
	contentErr error

	gotQuery, gotSpace, gotPageID string
}

func (f *fakeAPI) FindPagesByText(query, spaceKey string) ([]atlassian.Page, error) {
	f.gotQuery, f.gotSpace = query, spaceKey
	return f.pages, f.searchErr
}

func (f *fakeAPI) GetPageWithContent(id string) (*atlassian.PageContent, error) {
	f.gotPageID = id
	return f.content, f.contentErr
}

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
	api := &fakeAPI{pages: []atlassian.Page{{ID: "1"}, {ID: "2"}}}
	s := NewSession(api, target())
	if err := s.Search(); err != nil {
		t.Fatal(err)
	}
	if api.gotQuery != "OPS-7" || api.gotSpace != "DEV" || len(s.Pages) != 2 {
		t.Errorf("búsqueda mal armada: %q %q, %d páginas", api.gotQuery, api.gotSpace, len(s.Pages))
	}
}

func TestSearchWithoutPagesIsErrNoPages(t *testing.T) {
	err := NewSession(&fakeAPI{}, target()).Search()
	if !errors.Is(err, ErrNoPages) || !strings.Contains(err.Error(), "OPS-7") {
		t.Errorf("esperaba ErrNoPages con el issue, obtuve %v", err)
	}
}

func TestSearchPropagatesErrors(t *testing.T) {
	err := NewSession(&fakeAPI{searchErr: errors.New("401")}, target()).Search()
	if err == nil || errors.Is(err, ErrNoPages) {
		t.Errorf("un fallo de la API no es «sin páginas»: %v", err)
	}
}

func TestExport(t *testing.T) {
	api := &fakeAPI{content: &atlassian.PageContent{
		ID:          "42",
		Title:       "Documento de Despliegue - Validación de pagos",
		SpaceName:   "Desarrollo",
		StorageBody: "<p>Pasos del despliegue</p>",
	}}
	s := NewSession(api, target())

	e, err := s.Export(&atlassian.Page{ID: "42"})
	if err != nil {
		t.Fatal(err)
	}
	if api.gotPageID != "42" {
		t.Errorf("descargó la página %q", api.gotPageID)
	}
	if !strings.Contains(e.Text, "OPS-7 - Documento de Despliegue") || !strings.Contains(e.Text, "Pasos del despliegue") {
		t.Errorf("texto inesperado:\n%s", e.Text)
	}
	if e.FileName != "OPS-7_Documento_de_Despliegue_Validacion_de_pa.txt" {
		t.Errorf("nombre de archivo = %q", e.FileName)
	}
}

func TestExportErrors(t *testing.T) {
	s := NewSession(&fakeAPI{contentErr: errors.New("404")}, target())
	if _, err := s.Export(&atlassian.Page{ID: "42"}); err == nil {
		t.Error("un fallo al descargar debe devolverse")
	}
	if _, err := s.Export(nil); err == nil {
		t.Error("sin página elegida debe fallar")
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
