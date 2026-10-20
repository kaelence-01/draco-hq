package markdown

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

var ctx = context.Background()

func newStore(t *testing.T) (*Store, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "despliegues")
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s, dir
}

func put(t *testing.T, dir, rel, content string, age time.Duration) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	when := time.Now().Add(-age)
	if err := os.Chtimes(p, when, when); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, dir, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestEmptyFolderOffersItsRootAndTakesTheFirstDocument(t *testing.T) {
	s, dir := newStore(t)
	loc, err := s.Locations(ctx, provider.LocationQuery{TitlePrefix: "Documento de Despliegue"})
	if err != nil {
		t.Fatal(err)
	}
	if len(loc.Docs) != 1 || loc.Docs[0].ID != "./" {
		t.Fatalf("una carpeta vacía debe ofrecer su raíz: %+v", loc.Docs)
	}
	ref, err := s.Create(ctx, loc.Docs[0], provider.Document{Title: "Documento de Despliegue - OPS-7 - Filtro de pagos", Markdown: "## Hola\n"})
	if err != nil {
		t.Fatal(err)
	}
	if ref.ID != "documento-de-despliegue-ops-7-filtro-de-pagos.md" {
		t.Errorf("ID = %q", ref.ID)
	}
	if got := read(t, dir, ref.ID); got != "# Documento de Despliegue - OPS-7 - Filtro de pagos\n\n## Hola\n" {
		t.Errorf("contenido = %q", got)
	}
}

func TestCreateNextToADocumentAndAvoidsOverwriting(t *testing.T) {
	s, dir := newStore(t)
	put(t, dir, "2026-10/previo.md", "# Documento de Despliegue - OPS-3 - Exportar\n\nx\n", time.Hour)
	doc := provider.Document{Title: "Revisión de Facturación", Markdown: "texto"}
	first, err := s.Create(ctx, provider.DocRef{ID: "2026-10/previo.md"}, doc)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Create(ctx, provider.DocRef{ID: "2026-10/"}, doc)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != "2026-10/revision-de-facturacion.md" || second.ID != "2026-10/revision-de-facturacion-2.md" {
		t.Errorf("IDs = %q, %q", first.ID, second.ID)
	}
}

func TestFindAndSearch(t *testing.T) {
	s, dir := newStore(t)
	put(t, dir, "a.md", "# Documento de Despliegue - OPS-70 - Otro\n\nnada\n", time.Hour)
	put(t, dir, "b.md", "# Documento de Despliegue - OPS-7 - Filtro de pagos\n\nmenciona Rango.php\n", 2*time.Hour)
	put(t, dir, "notas.txt", "# no es markdown", 0)

	if r, _ := s.FindByTitle(ctx, "Documento de Despliegue - OPS-7 - Filtro de pagos"); r == nil || r.ID != "b.md" {
		t.Errorf("título exacto: %+v", r)
	}
	if r, _ := s.FindByTitleKey(ctx, "Documento de Despliegue", "OPS-7"); r == nil || r.ID != "b.md" {
		t.Errorf("OPS-7 no debe confundirse con OPS-70: %+v", r)
	}
	if r, _ := s.FindByTitle(ctx, "Inexistente"); r != nil {
		t.Errorf("no encontrado debe ser nil: %+v", r)
	}
	if rs, _ := s.Search(ctx, "rango.php"); len(rs) != 1 || rs[0].ID != "b.md" {
		t.Errorf("búsqueda en el contenido: %+v", rs)
	}
	if rs, _ := s.SearchTitle(ctx, "documento de despliegue"); len(rs) != 2 || rs[0].ID != "a.md" {
		t.Errorf("búsqueda por título, más nuevo primero: %+v", rs)
	}
}

func TestLocationsAreFoldersNewestFirst(t *testing.T) {
	s, dir := newStore(t)
	put(t, dir, "2026-09/x.md", "# Documento de Despliegue - OPS-1 - a\n", 48*time.Hour)
	put(t, dir, "2026-10/y.md", "# Documento de Despliegue - OPS-2 - b\n", time.Hour)
	put(t, dir, "otros/z.md", "# Notas de la reunión\n", 0)
	loc, err := s.Locations(ctx, provider.LocationQuery{TitlePrefix: "Documento de Despliegue", AuthorID: "ignorado"})
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, d := range loc.Docs {
		ids = append(ids, d.ID)
	}
	if !reflect.DeepEqual(ids, []string{"2026-10/", "2026-09/", "./"}) || loc.TeamWide {
		t.Errorf("ubicaciones = %v (TeamWide %v)", ids, loc.TeamWide)
	}
}

const published = `# Documento de Despliegue - OPS-7 - Filtro de pagos

## Arquitecturas e interfaces

| viejo |
|---|

## A considerar:

- [x] Respaldar base de datos
- [ ] Paso agregado a mano

` + "```" + `
## esto no es un título
` + "```" + `

### Nota del equipo

sigue dentro de la sección
`

func TestUpdateKeepsHandEditedSections(t *testing.T) {
	s, dir := newStore(t)
	put(t, dir, "doc.md", published, time.Hour)
	next := "## Arquitecturas e interfaces\n\n| nuevo |\n|---|\n\n## A considerar:\n\n- [ ] Paso de la plantilla\n"
	_, err := s.Update(ctx, provider.DocRef{ID: "doc.md"}, provider.Document{Title: "Documento de Despliegue - OPS-7 - Filtro de pagos (v2)", Markdown: next, Preserve: []string{"A considerar"}})
	if err != nil {
		t.Fatal(err)
	}
	got := read(t, dir, "doc.md")
	for _, want := range []string{"# Documento de Despliegue - OPS-7 - Filtro de pagos (v2)\n", "| nuevo |", "- [x] Respaldar base de datos", "Paso agregado a mano", "## esto no es un título", "sigue dentro de la sección"} {
		if !strings.Contains(got, want) {
			t.Errorf("falta %q en:\n%s", want, got)
		}
	}
	for _, gone := range []string{"| viejo |", "Paso de la plantilla"} {
		if strings.Contains(got, gone) {
			t.Errorf("sobra %q en:\n%s", gone, got)
		}
	}
}

func TestUpdateRefusesWhenTheFileCannotBeRead(t *testing.T) {
	s, dir := newStore(t)
	if _, err := s.Update(ctx, provider.DocRef{ID: "falta.md"}, provider.Document{Title: "x", Markdown: "y"}); err == nil || !strings.Contains(err.Error(), "no se sobrescribe") {
		t.Errorf("sin leer el actual no se escribe: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "falta.md")); err == nil {
		t.Error("no debía crearse el archivo")
	}
}

func TestSections(t *testing.T) {
	s, dir := newStore(t)
	put(t, dir, "qa.md", "# Consolidado\n\n## Resumen\n\n### Observación\n\n### Consideraciones\n\n- \n", 0)
	found, err := s.Sections(ctx, provider.DocRef{ID: "qa.md"}, []string{"Observación", "Consideraciones", "A considerar"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(found, []string{"Observación", "Consideraciones"}) {
		t.Errorf("una sección vacía existe; una ausente no: %v", found)
	}
	if _, err := s.Sections(ctx, provider.DocRef{ID: "falta.md"}, nil); err == nil {
		t.Error("un archivo que no se puede leer es un error")
	}
}

func TestNothingOutsideTheFolderIsTouched(t *testing.T) {
	s, dir := newStore(t)
	outside := filepath.Join(filepath.Dir(dir), "fuera.md")
	put(t, filepath.Dir(dir), "fuera.md", "# Fuera\n", 0)
	if _, err := s.Update(ctx, provider.DocRef{ID: "../fuera.md"}, provider.Document{Title: "x", Markdown: "y"}); err == nil {
		t.Error("una ruta con .. debe rechazarse")
	}
	if _, err := s.Create(ctx, provider.DocRef{ID: "../"}, provider.Document{Title: "x", Markdown: "y"}); err == nil {
		t.Error("crear fuera de la carpeta debe rechazarse")
	}
	if got, _ := os.ReadFile(outside); string(got) != "# Fuera\n" {
		t.Errorf("se modificó un archivo fuera del destino: %q", got)
	}
}

func TestExport(t *testing.T) {
	s, dir := newStore(t)
	put(t, dir, "doc.md", "# Título\n\n## Sección\n\ntexto\n", 0)
	e, err := s.Export(ctx, provider.DocRef{ID: "doc.md"})
	if err != nil {
		t.Fatal(err)
	}
	if e.Title != "Título" || e.Source != Source || e.Text != "## Sección\n\ntexto\n" || e.UpdatedDate == "" {
		t.Errorf("exportado = %+v", e)
	}
}

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"Documento de Despliegue - OPS-7 - Año / Módulo": "documento-de-despliegue-ops-7-ano-modulo",
		"¿?¡!":                  "documento",
		strings.Repeat("a", 90): strings.Repeat("a", 80),
	}
	for in, want := range cases {
		if got := slug(in); got != want {
			t.Errorf("slug(%q) = %q, esperaba %q", in, got, want)
		}
	}
}
