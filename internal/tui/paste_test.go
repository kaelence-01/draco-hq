package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func paste(s string) tea.PasteMsg { return tea.PasteMsg{Content: s} }

func TestOneLine(t *testing.T) {
	for in, want := range map[string]string{
		"a1b2c3d":                         "a1b2c3d",
		"  pagos  api \n":                 "pagos  api",
		"g -i OPS-7\r\n-b a1b2c3d\n\n":    "g -i OPS-7 -b a1b2c3d",
		"\n \n":                           "",
		"C:\\Mis  despliegues\\OPS-7.txt": "C:\\Mis  despliegues\\OPS-7.txt",
	} {
		if got := oneLine(in); got != want {
			t.Errorf("oneLine(%q) = %q, se esperaba %q", in, got, want)
		}
	}
}

func TestPasteAHashByHand(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toCommits(t, "OPS-7")
	send(e.m, paste("f1e2d3c"))
	if out := view(e.m); strings.Contains(out, "f1e2d3c") {
		t.Fatalf("sin escribir un hash, pegar no agrega nada:\n%s", out)
	}
	typeText(e.m, "/")
	send(e.m, paste("f1e2d3c Arregla el total de pagos\n"))
	if out := view(e.m); !strings.Contains(out, "/ f1e2d3c█") {
		t.Fatalf("se pega el hash de una línea de git log --oneline:\n%s", out)
	}
	send(e.m, keyEnter)
	send(e.m, keyEnter)
	if got := e.git.read["/code/pagos-api"]; strings.Join(got, ",") != "f1e2d3c" {
		t.Errorf("el hash pegado va al documento: %v", got)
	}
}

func TestPasteInTheFilter(t *testing.T) {
	e := newGenerateApp(t, nil)
	e.toPreview(t, "")
	send(e.m, keyEnter)
	send(e.m, paste("zzz"))
	if out := view(e.m); strings.Contains(out, "zzz") {
		t.Fatalf("sin / no hay filtro donde pegar:\n%s", out)
	}
	typeText(e.m, "/")
	send(e.m, paste("zz\nz"))
	if out := view(e.m); !strings.Contains(out, "/ zz z") || !strings.Contains(out, "Ningún documento coincide") {
		t.Errorf("lo pegado filtra, en una línea:\n%s", out)
	}
}

func TestPasteInTheBar(t *testing.T) {
	m := newBarApp(t)
	send(m, paste("help"))
	if out := view(m); strings.Contains(out, "/help█") {
		t.Fatalf("con la barra cerrada, pegar no la abre:\n%s", out)
	}
	typeText(m, "/")
	send(m, paste("g --pro\n"))
	if out := view(m); !strings.Contains(out, "/g --pro█") || !strings.Contains(out, "--profile") {
		t.Errorf("lo pegado se escribe en la barra y sugiere:\n%s", out)
	}
}

func TestPasteTheTaskAndThePathToExport(t *testing.T) {
	e := newExportApp(t, deployDoc)
	send(e.m, paste("OPS-7"))
	send(e.m, keyEnter)
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Guardar en") {
		t.Fatalf("la tarea pegada en el formulario busca el documento:\n%s", out)
	}
	path := filepath.Join(t.TempDir(), "OPS-7 pegado.txt")
	send(e.m, tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	send(e.m, paste(path+"\n"))
	send(e.m, keyEnter)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("se guarda en la ruta pegada: %v\n%s", err, view(e.m))
	}
}
