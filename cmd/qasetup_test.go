package cmd

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

// setupTracker is a tracker that lists statuses and fields, as Jira does.
type setupTracker struct {
	fakeTracker
	project string
}

func (s *setupTracker) ProjectStatuses(_ context.Context, project string) ([]provider.Status, error) {
	s.project = project
	return []provider.Status{
		{ID: "101", Name: "Por hacer"}, {ID: "102", Name: "En curso"}, {ID: "103", Name: "Testing"},
		{ID: "104", Name: "Aprobado"}, {ID: "105", Name: "Finalizado", Done: true},
	}, nil
}

func (*setupTracker) QAFields(context.Context) ([]provider.Field, error) {
	return []provider.Field{
		{ID: "customfield_1", Name: "Desarrollo", Kind: provider.FieldPRStatus},
		{ID: "customfield_2", Name: "Errores", Kind: provider.FieldNumber},
		{ID: "customfield_3", Name: "Reprocesos", Kind: provider.FieldNumber},
	}, nil
}

func setupHome(t *testing.T, trk provider.Tracker) {
	t.Helper()
	configHome(t)
	writeConfig(t, &config.Config{AtlassianEmail: "qa@example.test", AtlassianToken: "token-del-archivo", BaseURL: "https://example.test"})
	t.Setenv("ATLASSIAN_TOKEN", "token-del-entorno")
	fakeAccount(t, trk)
}

func TestQASetupPicksByNumber(t *testing.T) {
	trk := &setupTracker{}
	setupHome(t, trk)
	// OPS; n: pick the statuses; Enter: Testing; 3,4,5; a wrong number, then 1; 2. The only PR field is taken.
	code, out, errOut := runWith("OPS\nn\n\n3,4,5\n9\n1\n2\n", "qa", "setup")
	if code != exitOK {
		t.Fatalf("código %d\n%s\n%s", code, out, errOut)
	}
	for _, want := range []string{"Proyecto OPS", "Sugerido por draco", "Ya llegó a QA (2)", "Testing, Finalizado", "(falta elegirlo)", "¿Usar estos estados?", "Estados de OPS:", "[3] * Testing", "[5] * Finalizado  (finalizado)", "[3,5]", "entre 1 y 2", "Estado del PR: Desarrollo", "QA guardado: proyecto OPS, 3 estados", "falta qa_email"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en:\n%s", want, out)
		}
	}
	cfg := readConfig(t)
	q := cfg.QAJira
	if trk.project != "OPS" || q == nil || q.Project != "OPS" || !reflect.DeepEqual(q.Statuses, []string{"103", "104", "105"}) || q.TestingStatus != "" {
		t.Fatalf("se guardan los ids de los estados elegidos: %+v", q)
	}
	if q.Fields != (config.QAJiraFields{CodingErrors: "customfield_2", DevReturns: "customfield_3", DevStatus: "customfield_1"}) {
		t.Errorf("se guardan los ids de los campos elegidos: %+v", q.Fields)
	}
	if cfg.AtlassianToken != "token-del-archivo" {
		t.Errorf("el token del entorno no llega al archivo: %q", cfg.AtlassianToken)
	}

	// Again with -j: no question for the project; what was saved is complete, Enter keeps it.
	code, out, _ = runWith("\n", "qa", "setup", "-j", "OPS")
	if code != exitOK || strings.Contains(out, "Proyecto de Jira") || strings.Contains(out, "Estados de OPS:") {
		t.Errorf("con -j no pregunta el proyecto y, completo, no muestra las listas: %d\n%s", code, out)
	}
	for _, want := range []string{"Propuesto desde tu configuración", "Testing, Aprobado, Finalizado", "Reprocesos", "¿Guardar así?"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en el resumen:\n%s", want, out)
		}
	}
	if got := readConfig(t).QAJira; !reflect.DeepEqual(got, q) {
		t.Errorf("Enter deja la config igual: %+v", got)
	}
}

func TestQASetupAcceptsTheSuggestedStatuses(t *testing.T) {
	setupHome(t, &setupTracker{})
	// OPS; Enter: these statuses; then only the two counters.
	code, out, errOut := runWith("OPS\n\n1\n2\n", "qa", "setup")
	if code != exitOK || strings.Contains(out, "Estados de OPS:") {
		t.Fatalf("aceptar los estados salta su lista: %d\n%s\n%s", code, out, errOut)
	}
	if q := readConfig(t).QAJira; q == nil || !reflect.DeepEqual(q.Statuses, []string{"103", "105"}) || q.Fields.DevReturns != "customfield_3" {
		t.Errorf("se guardan los estados sugeridos y los campos elegidos: %+v", q)
	}
}

func TestQASetupNeedsATrackerThatListsThem(t *testing.T) {
	setupHome(t, &fakeTracker{})
	if code, _, errOut := runWith("OPS\n", "qa", "setup"); code != exitError || !strings.Contains(errOut, "no soporta el consolidado QA") {
		t.Errorf("un tracker sin la capacidad: %d %s", code, errOut)
	}
}
