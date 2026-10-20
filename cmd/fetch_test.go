package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeDoc(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestFetchExportsTheOnlyDocument(t *testing.T) {
	dir := fakeEnv(t, &fakeTracker{})
	writeDoc(t, dir, "ops-7.md", "# Documento de Despliegue - OPS-7 - Filtro\n\n## A considerar:\n\n- [x] Respaldo\n")

	code, out, errOut := run("f", "-i", "ops-7")
	if code != exitOK || !strings.Contains(out, "Encontrado: Documento de Despliegue - OPS-7 - Filtro") {
		t.Fatalf("código %d\n%s\n%s", code, out, errOut)
	}
	cwd, _ := os.Getwd()
	txt, err := os.ReadFile(filepath.Join(cwd, "OPS-7_Documento_de_Despliegue_OPS_7_Filtro.txt"))
	if err != nil {
		t.Fatalf("sin diálogo se guarda en la carpeta actual: %v\n%s", err, out)
	}
	for _, want := range []string{"OPS-7 - Documento de Despliegue - OPS-7 - Filtro", "Fuente: Markdown local", "- [x] Respaldo", "FIN DEL DOCUMENTO"} {
		if !strings.Contains(string(txt), want) {
			t.Errorf("falta %q en:\n%s", want, txt)
		}
	}
}

func TestFetchChoosesAmongSeveralAndHonorsOutput(t *testing.T) {
	dir := fakeEnv(t, &fakeTracker{})
	writeDoc(t, dir, "a.md", "# Documento de Despliegue - OPS-7 - Primero\n\nOPS-7\n")
	writeDoc(t, dir, "b.md", "# Consolidado de Pruebas QA - Pagos - Sprint 12\n\nmenciona OPS-7\n")
	out := filepath.Join(t.TempDir(), "elegido.txt")

	code, stdout, errOut := runWith("2\n", "f", "-i", "OPS-7", "-o", out)
	if code != exitOK || !strings.Contains(stdout, "[2]") || !strings.Contains(stdout, "Elegido:") {
		t.Fatalf("código %d\n%s\n%s", code, stdout, errOut)
	}
	if _, err := os.Stat(out); err != nil {
		t.Errorf("-o debe respetarse: %v", err)
	}
	if code, _, errOut := runWith("9\n", "f", "-i", "OPS-7"); code != exitError || !strings.Contains(errOut, "opción inválida") {
		t.Errorf("opción fuera de rango: %d %s", code, errOut)
	}
}

func TestFetchWithoutDocuments(t *testing.T) {
	fakeEnv(t, &fakeTracker{})
	if code, _, errOut := run("f", "-i", "OPS-9"); code != exitError || !strings.Contains(errOut, "no se encontraron documentos") {
		t.Errorf("%d %s", code, errOut)
	}
	if code, _, _ := run("f"); code != exitUsage {
		t.Errorf("sin -i es un error de uso: %d", code)
	}
}

func TestWSLPath(t *testing.T) {
	if got := wslPath(`C:\Users\ana\Desktop\x.txt`); got != "/mnt/c/Users/ana/Desktop/x.txt" {
		t.Errorf("wslPath = %q", got)
	}
	if got := wslPath(""); got != "" {
		t.Errorf("sin elección queda vacío: %q", got)
	}
}
