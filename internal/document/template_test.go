package document

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func templateDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestBuiltinTemplatesManifests(t *testing.T) {
	d := MustBuiltin("deploy")
	if d.Manifest.Prefix != "Documento de Despliegue" || len(d.Manifest.Preserve) != 1 || d.Manifest.Preserve[0] != "A considerar" || d.Source != "" {
		t.Errorf("deploy = %+v", d.Manifest)
	}
	q := MustBuiltin("qa")
	if q.Manifest.Prefix != "Consolidado de Pruebas QA" || strings.Join(q.Manifest.Preserve, ",") != "Observación,Consideraciones" {
		t.Errorf("qa = %+v", q.Manifest)
	}
	if _, err := LoadTemplate("inexistente"); err == nil {
		t.Error("una plantilla incluida inexistente es un error")
	}
}

func TestBlockOverrideChangesOnlyThatPart(t *testing.T) {
	dir := templateDir(t, map[string]string{
		"checklist.md.tmpl": "{{define \"checklist\"}}## A considerar:\n\n{{range .Checklist}}1. {{.}}\n{{end}}{{end}}",
	})
	tpl, err := LoadTemplate("deploy", "", filepath.Join(dir, "no-existe"), dir)
	if err != nil {
		t.Fatal(err)
	}
	if tpl.Source != dir {
		t.Errorf("Source = %q, esperaba la primera carpeta que existe", tpl.Source)
	}
	md, err := Deploy(tpl, DeployDoc{IssueURL: "https://example.test/browse/OPS-7", Checklist: []string{"Paso uno"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "1. Paso uno") || strings.Contains(md, "- [ ]") {
		t.Errorf("el bloque checklist debía reemplazarse:\n%s", md)
	}
	if !strings.Contains(md, "| **Tarea(s)** | <https://example.test/browse/OPS-7> |") {
		t.Errorf("el resto del documento debía quedar igual:\n%s", md)
	}
}

func TestMainAndManifestOverride(t *testing.T) {
	dir := templateDir(t, map[string]string{
		"main.md.tmpl":  "# Despliegue {{.Issue}}\n",
		"template.yaml": "title: \"Despliegue - {{.IssueKey}}\"\nprefix: \"Despliegue\"\n",
	})
	tpl, err := LoadTemplate("deploy", dir)
	if err != nil {
		t.Fatal(err)
	}
	md, _ := Deploy(tpl, DeployDoc{IssueURL: "https://example.test/browse/OPS-7"})
	title, err := DeployTitle(tpl, "OPS-7", "x")
	if err != nil {
		t.Fatal(err)
	}
	if md != "# Despliegue <https://example.test/browse/OPS-7>\n" || title != "Despliegue - OPS-7" {
		t.Errorf("md = %q, title = %q", md, title)
	}
	if len(tpl.Manifest.Preserve) != 1 {
		t.Errorf("un campo que el manifiesto no trae se hereda: %+v", tpl.Manifest)
	}
}

func TestTitleMustKeepThePrefix(t *testing.T) {
	dir := templateDir(t, map[string]string{"template.yaml": "title: \"Deploy {{.IssueKey}}\"\n"})
	tpl, err := LoadTemplate("deploy", dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DeployTitle(tpl, "OPS-7", "x"); err == nil || !strings.Contains(err.Error(), "prefijo") {
		t.Errorf("un título sin el prefijo rompe las búsquedas: %v", err)
	}
}

func TestBrokenTemplatesAreReported(t *testing.T) {
	if _, err := LoadTemplate("deploy", templateDir(t, map[string]string{"main.md.tmpl": "{{range .Sides}"})); err == nil {
		t.Error("una plantilla con error de sintaxis debe fallar al cargar")
	}
	tpl, err := LoadTemplate("deploy", templateDir(t, map[string]string{"main.md.tmpl": "{{.NoExiste}}"}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Deploy(tpl, DeployDoc{}); err == nil {
		t.Error("un campo inexistente debe fallar al generar, no publicar un documento roto")
	}
	file := filepath.Join(t.TempDir(), "archivo")
	_ = os.WriteFile(file, nil, 0644)
	if _, err := LoadTemplate("deploy", file); err == nil {
		t.Error("una ruta que no es carpeta es un error")
	}
}

func TestCustomTemplateCannotBreakTables(t *testing.T) {
	dir := templateDir(t, map[string]string{
		"main.md.tmpl": "| Archivo |\n|---|\n{{range .Sides}}{{range .Rows}}| {{join .Files \" \"}} |\n{{end}}{{end}}",
	})
	tpl, err := LoadTemplate("deploy", dir)
	if err != nil {
		t.Fatal(err)
	}
	md, _ := Deploy(tpl, DeployDoc{BackendRepo: "api", BackendFiles: map[string][]string{"app": {"a|b.php"}}})
	for _, n := range cells(t, md) {
		if n != 1 {
			t.Fatalf("los valores llegan escapados aunque la plantilla no escape:\n%s", md)
		}
	}
}

func TestFechaEnEspanol(t *testing.T) {
	if got := fecha(time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)); got != "05 ene 2026" {
		t.Errorf("fecha = %q", got)
	}
}

func TestDirsOrder(t *testing.T) {
	got := Dirs("qa", "/explicita", "/repos/pagos-api", "/home/yo/.config/draco/templates")
	want := []string{"/explicita", filepath.Join("/repos/pagos-api", ".draco", "templates", "qa"), filepath.Join("/home/yo/.config/draco/templates", "qa")}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("Dirs = %v, want %v", got, want)
	}
	if got := Dirs("deploy", "", "", ""); len(got) != 2 || got[1] != filepath.Join(".draco", "templates", "deploy") {
		t.Errorf("sin repo ni usuario: %v", got)
	}
}

func TestBuiltinFilesRoundTrip(t *testing.T) {
	for _, kind := range Kinds {
		files, err := BuiltinFiles(kind)
		if err != nil || files["template.yaml"] == nil || files["main.md.tmpl"] == nil {
			t.Fatalf("%s: %v %v", kind, err, files)
		}
		dir := t.TempDir()
		for n, b := range files {
			if err := os.WriteFile(filepath.Join(dir, n), b, 0644); err != nil {
				t.Fatal(err)
			}
		}
		copied, err := LoadTemplate(kind, dir)
		if err != nil || copied.Manifest.Prefix != MustBuiltin(kind).Manifest.Prefix {
			t.Errorf("%s: una copia sin editar se carga igual que la incluida: %v", kind, err)
		}
	}
	if _, err := BuiltinFiles("otro"); err == nil {
		t.Error("un tipo desconocido es error")
	}
}

func TestMissingPreserved(t *testing.T) {
	q := MustBuiltin("qa")
	md := "## Resumen\n\n### Observación\n\n```\n### Consideraciones\n```\n"
	if got := q.MissingPreserved(md); len(got) != 1 || got[0] != "Consideraciones" {
		t.Errorf("un título dentro de un bloque de código no cuenta: %v", got)
	}
	if got := MustBuiltin("deploy").MissingPreserved("## A considerar:\n"); got != nil {
		t.Errorf("los dos puntos finales no importan: %v", got)
	}
}
