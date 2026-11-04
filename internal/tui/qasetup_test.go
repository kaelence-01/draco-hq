package tui

import (
	"context"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/profile"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

// setupTracker lists statuses and fields, as Jira does.
type setupTracker struct {
	*fakeTracker
	project string
}

func (s *setupTracker) ProjectStatuses(_ context.Context, project string) ([]provider.Status, error) {
	s.project = project
	return []provider.Status{
		{ID: "101", Name: "Por hacer"}, {ID: "102", Name: "Testing"}, {ID: "103", Name: "Aprobado"}, {ID: "104", Name: "Finalizado", Done: true},
	}, nil
}

func (*setupTracker) QAFields(context.Context) ([]provider.Field, error) {
	return []provider.Field{
		{ID: "customfield_1", Name: "Desarrollo", Kind: provider.FieldPRStatus},
		{ID: "customfield_2", Name: "Errores", Kind: provider.FieldNumber},
		{ID: "customfield_3", Name: "Reprocesos", Kind: provider.FieldNumber},
	}, nil
}

func newSetupApp(t *testing.T, trk provider.Tracker) *settingsEnv {
	t.Helper()
	return newSettingsApp(t, savedConfig(), func(d *Deps) {
		d.OpenProfile = func(*config.Config, profile.Options) (*profile.Bundle, error) {
			return &profile.Bundle{Tracker: trk}, nil
		}
	})
}

func (e *settingsEnv) toQASetup(t *testing.T) {
	t.Helper()
	for range 3 {
		send(e.m, keyDown)
	}
	if out := view(e.m); !strings.Contains(out, "d detectar desde Jira") {
		t.Fatalf("en QA el pie ofrece detectar desde Jira:\n%s", out)
	}
	send(e.m, press('d', "d"))
	if out := view(e.m); !strings.Contains(out, "Configuración › Detectar desde Jira") || !strings.Contains(out, "Proyecto de Jira") {
		t.Fatalf("d abre la detección y pide el proyecto:\n%s", out)
	}
}

func TestQASetupPicksStatusesAndFieldsByName(t *testing.T) {
	trk := &setupTracker{fakeTracker: &fakeTracker{}}
	e := newSetupApp(t, trk)
	e.toQASetup(t)
	e.fill("OPS")
	out := view(e.m)
	for _, want := range []string{"Estado con el que una tarea entra a QA", "Testing", "Estados de una tarea que ya llegó a QA", "Aprobado", "Finalizado · finalizado"} {
		if !strings.Contains(out, want) {
			t.Fatalf("falta %q: los estados de OPS por nombre:\n%s", want, out)
		}
	}
	if strings.Contains(out, "101") {
		t.Errorf("los ids no se muestran:\n%s", out)
	}
	send(e.m, keyEnter)        // Testing, the suggestion
	send(e.m, keyDown)         // the cursor starts on the first marked one
	send(e.m, press(' ', " ")) // also Aprobado
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Errores  (customfield_2)") || !strings.Contains(out, "Desarrollo  (customfield_1)") {
		t.Fatalf("los campos se eligen por nombre:\n%s", out)
	}
	send(e.m, keyEnter) // Errores
	send(e.m, keyDown)
	send(e.m, keyEnter) // Reprocesos
	send(e.m, keyEnter) // Desarrollo

	q := e.file.cfg.QAJira
	if trk.project != "OPS" || q == nil || q.Project != "OPS" || q.TestingStatus != "" {
		t.Fatalf("se guarda qa_jira de OPS: %q %+v", trk.project, q)
	}
	got := slices.Sorted(slices.Values(q.Statuses))
	if !reflect.DeepEqual(got, []string{"102", "103", "104"}) {
		t.Errorf("se guardan los ids de Testing, lo marcado y los finalizados: %q", q.Statuses)
	}
	if q.Fields != (config.QAJiraFields{CodingErrors: "customfield_2", DevReturns: "customfield_3", DevStatus: "customfield_1"}) {
		t.Errorf("se guardan los ids de los campos: %+v", q.Fields)
	}
	if e.file.cfg.AtlassianToken != "token-del-archivo" {
		t.Errorf("el token del entorno no llega al archivo: %q", e.file.cfg.AtlassianToken)
	}
	if out := view(e.m); len(e.m.stack) != 2 || !strings.Contains(out, "QA guardado: proyecto OPS, 3 estados") {
		t.Errorf("vuelve a Configuración con lo guardado:\n%s", out)
	}
}

func TestQASetupExplainsATrackerThatCannot(t *testing.T) {
	e := newSetupApp(t, &fakeTracker{})
	e.toQASetup(t)
	e.fill("OPS")
	if out := view(e.m); !strings.Contains(out, "no soporta el consolidado QA") || !strings.Contains(out, "Proyecto de Jira") || e.file.saves != 0 {
		t.Errorf("el error queda en el formulario del proyecto, sin guardar:\n%s", out)
	}
}

func TestBarOpensQASetup(t *testing.T) {
	trk := &setupTracker{fakeTracker: &fakeTracker{}}
	e := newSetupApp(t, trk)
	runBar(e.m, "qa setup -j OPS")
	if out := view(e.m); trk.project != "OPS" || !strings.Contains(out, "Configuración › Detectar desde Jira") || !strings.Contains(out, "Estados de una tarea que ya llegó a QA") {
		t.Errorf("qa setup -j lee el proyecto de una vez, aunque no esté en qa_jira:\n%s", out)
	}
}
