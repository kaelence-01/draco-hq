package cmd

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kaelence-01/draco-hq/internal/history"
)

func historyEntries(t *testing.T) []history.Entry {
	t.Helper()
	path, _ := historyPath()
	entries, err := history.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func TestPublishingRecordsTheDocumentOnce(t *testing.T) {
	fakeEnv(t, &fakeTracker{})
	if code, out, errOut := runWith("1\n\n", "g", "-i", "OPS-7", "-b", "a1b2c3d"); code != exitOK {
		t.Fatalf("código %d\n%s\n%s", code, out, errOut)
	}
	entries := historyEntries(t)
	if len(entries) != 1 || entries[0].Issue != "OPS-7" || entries[0].Updated || entries[0].Dest != "markdown" || entries[0].Profile != "prueba" {
		t.Fatalf("historial tras crear: %+v", entries)
	}

	if code, out, errOut := runWith("1\n\n", "g", "-i", "OPS-7", "-b", "a1b2c3d"); code != exitOK {
		t.Fatalf("código %d\n%s\n%s", code, out, errOut)
	}
	if entries = historyEntries(t); len(entries) != 1 || !entries[0].Updated {
		t.Fatalf("actualizar no agrega otra entrada, la marca actualizada: %+v", entries)
	}

	code, out, _ := run("history")
	for _, want := range []string{"OPS-7", "actualizado", "Documento de Despliegue - OPS-7 - Filtro de pagos", entries[0].URL} {
		if code != exitOK || !strings.Contains(out, want) {
			t.Errorf("falta %q en draco history:\n%s", want, out)
		}
	}
}

func TestDryRunRecordsNothing(t *testing.T) {
	fakeEnv(t, &fakeTracker{})
	if code, _, _ := run("g", "-i", "OPS-7", "-b", "a1b2c3d", "--dry-run"); code != exitOK {
		t.Fatal("el --dry-run falló")
	}
	if entries := historyEntries(t); len(entries) != 0 {
		t.Errorf("--dry-run no publica, no va al historial: %+v", entries)
	}
}

func TestHistoryRemembersAFailedLink(t *testing.T) {
	fakeEnv(t, &fakeTracker{linkErr: errors.New("sin permiso")})
	runWith("1\n\n", "g", "-i", "OPS-7", "-b", "a1b2c3d")
	if entries := historyEntries(t); len(entries) != 1 || !entries[0].LinkFailed {
		t.Fatalf("el enlace fallido se guarda: %+v", entries)
	}
	if _, out, _ := run("history"); !strings.Contains(out, "sin enlace desde la tarea") {
		t.Errorf("draco history avisa del enlace fallido:\n%s", out)
	}
}

func TestADamagedHistoryDoesNotStopPublishing(t *testing.T) {
	fakeEnv(t, &fakeTracker{})
	path, _ := historyPath()
	if err := os.WriteFile(path, []byte("{no es json"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runWith("1\n\n", "g", "-i", "OPS-7", "-b", "a1b2c3d")
	if code != exitOK || !strings.Contains(out, "Documento creado") || !strings.Contains(out, "no se guardó en el historial") {
		t.Errorf("el documento se publica y el historial se avisa: código %d\n%s", code, out)
	}
	if code, _, errOut := run("history"); code != exitError || !strings.Contains(errOut, "dañado") {
		t.Errorf("draco history explica el historial dañado: código %d\n%s", code, errOut)
	}
}

func TestHistoryFiltersAndLimits(t *testing.T) {
	fakeEnv(t, &fakeTracker{})
	path, _ := historyPath()
	at := time.Date(2026, 11, 9, 10, 42, 0, 0, time.Local)
	for i, e := range []history.Entry{
		{Issue: "OPS-3", Title: "Documento OPS-3", URL: "https://example.test/3", DocID: "3", Project: "facturacion"},
		{Issue: "OPS-9", Title: "Documento OPS-9", URL: "https://example.test/9", DocID: "9", Project: "pagos"},
		{Issue: "OPS-12", Title: "Documento OPS-12", URL: "https://example.test/12", DocID: "12", Project: "pagos", Updated: true},
	} {
		e.Dest, e.At = "confluence", at.Add(time.Duration(i)*time.Hour)
		if err := history.Record(path, e); err != nil {
			t.Fatal(err)
		}
	}

	_, out, _ := run("history", "-p", "pagos")
	if !strings.Contains(out, "OPS-12") || !strings.Contains(out, "OPS-9") || strings.Contains(out, "OPS-3") || strings.Contains(out, "  pagos") {
		t.Errorf("-p pagos muestra solo los de pagos, sin repetir el proyecto:\n%s", out)
	}
	if strings.Index(out, "OPS-12") > strings.Index(out, "OPS-9") {
		t.Errorf("el más reciente va primero:\n%s", out)
	}
	if !strings.Contains(out, "2026-11-09 11:42") {
		t.Errorf("falta la fecha:\n%s", out)
	}

	_, out, _ = run("history", "-n", "1")
	if !strings.Contains(out, "OPS-12") || strings.Contains(out, "OPS-9") || !strings.Contains(out, "1 de 3. Más con -n 3.") {
		t.Errorf("-n 1 muestra uno y dice cuántos hay:\n%s", out)
	}

	if _, out, _ = run("history", "-p", "inventario"); !strings.Contains(out, "No hay documentos de inventario") {
		t.Errorf("un proyecto sin documentos se dice:\n%s", out)
	}
	if code, _, errOut := run("history", "-n", "0"); code != exitUsage || !strings.Contains(errOut, "mayor que cero") {
		t.Errorf("-n 0 es un error de uso: código %d\n%s", code, errOut)
	}
}

func TestEmptyHistory(t *testing.T) {
	fakeEnv(t, &fakeTracker{})
	if code, out, _ := run("history"); code != exitOK || !strings.Contains(out, "Todavía no hay documentos") {
		t.Errorf("historial vacío: código %d\n%s", code, out)
	}
}
