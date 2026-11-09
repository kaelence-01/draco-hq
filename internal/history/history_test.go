package history

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kaelence-01/draco-hq/internal/deploy"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

func entry(issue, id, project string) Entry {
	return Entry{Issue: issue, Title: "Documento de Despliegue - " + issue, URL: "https://example.test/pages/" + id, DocID: id, Dest: "confluence", Project: project}
}

func issues(entries []Entry) []string {
	var out []string
	for _, e := range entries {
		out = append(out, e.Issue)
	}
	return out
}

func TestLoadWithoutFileIsEmpty(t *testing.T) {
	entries, err := Load(filepath.Join(t.TempDir(), "history.json"))
	if err != nil || entries != nil {
		t.Fatalf("sin archivo el historial está vacío: %v %v", entries, err)
	}
}

func TestRecordKeepsTheNewestFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "draco", "history.json")
	for _, e := range []Entry{entry("OPS-7", "1", "pagos"), entry("OPS-12", "2", "facturacion")} {
		if err := Record(path, e); err != nil {
			t.Fatal(err)
		}
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"OPS-12", "OPS-7"}; !reflect.DeepEqual(issues(got), want) {
		t.Errorf("orden %v, se esperaba %v", issues(got), want)
	}
	if info, _ := os.Stat(path); info.Mode().Perm()&0o077 != 0 && os.PathSeparator == '/' {
		t.Errorf("el historial es solo del usuario: permisos %v", info.Mode().Perm())
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(path), "history-*")); len(left) > 0 {
		t.Errorf("quedaron archivos temporales: %v", left)
	}
}

func TestRecordReplacesTheSameDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	created := entry("OPS-7", "1", "pagos")
	updated := created
	updated.Updated = true
	for _, e := range []Entry{created, entry("OPS-12", "2", "pagos"), updated} {
		if err := Record(path, e); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := Load(path)
	if len(got) != 2 || got[0].Issue != "OPS-7" || !got[0].Updated {
		t.Errorf("un documento actualizado sube al primer lugar una sola vez: %+v", got)
	}
}

func TestDocumentsOfTwoDestinationsAreDifferent(t *testing.T) {
	md := entry("OPS-7", "1", "pagos")
	md.Dest = "markdown"
	if got := Add([]Entry{entry("OPS-7", "1", "pagos")}, md); len(got) != 2 {
		t.Errorf("el mismo id en otro destino es otro documento: %+v", got)
	}
}

func TestAddKeepsTheLimit(t *testing.T) {
	var entries []Entry
	for i := range Limit + 5 {
		entries = Add(entries, entry("OPS-"+strconv.Itoa(i), strconv.Itoa(i), "pagos"))
	}
	if len(entries) != Limit || entries[0].Issue != "OPS-"+strconv.Itoa(Limit+4) {
		t.Errorf("se esperaban las %d más recientes: %d, la primera %s", Limit, len(entries), entries[0].Issue)
	}
}

func TestADamagedHistoryIsNotOverwritten(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	if err := os.WriteFile(path, []byte("{no es json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "dañado") {
		t.Errorf("un historial dañado se informa: %v", err)
	}
	if err := Record(path, entry("OPS-7", "1", "pagos")); err == nil {
		t.Error("no se guarda sobre un historial que no se pudo leer")
	}
	if b, _ := os.ReadFile(path); string(b) != "{no es json" {
		t.Errorf("el historial dañado cambió: %q", b)
	}
}

func TestOfAndProjects(t *testing.T) {
	entries := []Entry{entry("OPS-7", "1", "pagos"), entry("OPS-12", "2", "facturacion"), entry("OPS-9", "3", "pagos"), entry("OPS-3", "4", "")}
	if got := issues(Of(entries, "pagos")); !reflect.DeepEqual(got, []string{"OPS-7", "OPS-9"}) {
		t.Errorf("pagos: %v", got)
	}
	if got := Of(entries, ""); len(got) != 4 {
		t.Errorf("sin proyecto son todos: %v", issues(got))
	}
	if got := Projects(entries); !reflect.DeepEqual(got, []string{"facturacion", "pagos"}) {
		t.Errorf("proyectos: %v", got)
	}
}

func TestDeployEntry(t *testing.T) {
	at := time.Date(2026, 11, 9, 10, 42, 0, 0, time.UTC)
	target := &deploy.Target{Issue: "OPS-7", ProjectName: "pagos"}
	draft := &deploy.Draft{Title: "Documento de Despliegue - OPS-7 - Filtro de pagos", NotPreserved: []string{"A considerar"}}
	res := &deploy.Result{Page: &provider.DocRef{ID: "42", URL: "https://example.test/pages/42"}, Updated: true, LinkErr: errors.New("sin permiso")}
	got := Deploy(target, draft, res, "equipo", "confluence", at)
	want := Entry{Issue: "OPS-7", Title: draft.Title, URL: "https://example.test/pages/42", DocID: "42", Dest: "confluence", Project: "pagos",
		Profile: "equipo", Updated: true, At: at, LinkFailed: true, NotPreserved: []string{"A considerar"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("entrada\n%+v\nse esperaba\n%+v", got, want)
	}
}
