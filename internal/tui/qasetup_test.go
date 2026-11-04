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
	return newSetupAppWith(t, savedConfig(), trk)
}

func newSetupAppWith(t *testing.T, cfg *config.Config, trk provider.Tracker) *settingsEnv {
	t.Helper()
	return newSettingsApp(t, cfg, func(d *Deps) {
		d.OpenProfile = func(*config.Config, profile.Options) (*profile.Bundle, error) {
			return &profile.Bundle{Tracker: trk}, nil
		}
	})
}

// toQASetup opens Detectar desde Jira with enter: qa_jira is not complete.
func (e *settingsEnv) toQASetup(t *testing.T) {
	t.Helper()
	for range 3 {
		send(e.m, keyDown)
	}
	if out := view(e.m); !strings.Contains(out, "enter detectar desde Jira") || !strings.Contains(out, "e editar a mano") {
		t.Fatalf("sin qa_jira completo, enter en QA detecta desde Jira y e abre el formulario:\n%s", out)
	}
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Configuración › Detectar desde Jira") || !strings.Contains(out, "Proyecto de Jira") {
		t.Fatalf("enter abre la detección y pide el proyecto:\n%s", out)
	}
}

func TestQASetupShowsWhatItProposes(t *testing.T) {
	e := newSetupApp(t, &setupTracker{fakeTracker: &fakeTracker{}})
	e.toQASetup(t)
	e.fill("OPS")
	out := view(e.m)
	for _, want := range []string{"Proyecto OPS", "Sugerido por draco", "Entra a QA con", "Ya llegó a QA (2)", "Testing, Finalizado", "Los demás estados", "Por hacer, Aprobado",
		"Errores de codificación", "falta elegirlo", "Desarrollo", "enter elegir los campos", "e cambiar"} {
		if !strings.Contains(out, want) {
			t.Fatalf("falta %q: el resumen dice qué se guardará y qué falta:\n%s", want, out)
		}
	}
	if strings.Contains(out, "101") || strings.Contains(out, "customfield_") {
		t.Errorf("el resumen no muestra ids:\n%s", out)
	}
}

func TestQASetupEnterAsksOnlyTheMissingFields(t *testing.T) {
	e := newSetupApp(t, &setupTracker{fakeTracker: &fakeTracker{}})
	e.toQASetup(t)
	e.fill("OPS")
	send(e.m, keyEnter)
	out := view(e.m)
	if strings.Contains(out, "Estados de una tarea que ya llegó a QA") || !strings.Contains(out, "Campo de errores de codificación") {
		t.Fatalf("enter acepta los estados y pide solo los campos:\n%s", out)
	}
	send(e.m, keyEnter) // Errores
	send(e.m, keyDown)
	send(e.m, keyEnter) // Reprocesos
	send(e.m, keyEnter) // Desarrollo
	q := e.file.cfg.QAJira
	if q == nil || !reflect.DeepEqual(q.Statuses, []string{"102", "104"}) || q.Fields.DevReturns != "customfield_3" {
		t.Fatalf("se guardan los estados propuestos y los campos elegidos: %+v", q)
	}
	if out := view(e.m); !strings.Contains(out, "QA guardado: proyecto OPS, 2") || !strings.Contains(out, "Falta el email") {
		t.Errorf("vuelve a Configuración y dice que falta el email:\n%s", out)
	}
}

func TestQASetupChangesStatusesAndFieldsByName(t *testing.T) {
	trk := &setupTracker{fakeTracker: &fakeTracker{}}
	e := newSetupApp(t, trk)
	e.toQASetup(t)
	e.fill("OPS")
	send(e.m, press('e', "e"))
	out := view(e.m)
	for _, want := range []string{"Estado con el que una tarea entra a QA", "Estados de una tarea que ya llegó a QA", "Aprobado", "Finalizado · finalizado"} {
		if !strings.Contains(out, want) {
			t.Fatalf("falta %q: e abre las listas, por nombre:\n%s", want, out)
		}
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
	if got := slices.Sorted(slices.Values(q.Statuses)); !reflect.DeepEqual(got, []string{"102", "103", "104"}) {
		t.Errorf("se guardan los ids de Testing, lo marcado y los finalizados: %q", q.Statuses)
	}
	if q.Fields != (config.QAJiraFields{CodingErrors: "customfield_2", DevReturns: "customfield_3", DevStatus: "customfield_1"}) {
		t.Errorf("se guardan los ids de los campos: %+v", q.Fields)
	}
	if e.file.cfg.AtlassianToken != "token-del-archivo" {
		t.Errorf("el token del entorno no llega al archivo: %q", e.file.cfg.AtlassianToken)
	}
}

func TestQASetupSavesWhatTheConfigHadWithEnter(t *testing.T) {
	cfg := savedConfig()
	cfg.QAEmail = "qa@example.test"
	cfg.QAJira = &config.QAJiraConfig{Project: "OPS", Statuses: []string{"Testing", "103", "104"},
		Fields: config.QAJiraFields{CodingErrors: "customfield_2", DevReturns: "customfield_3", DevStatus: "customfield_1"}}
	e := newSetupAppWith(t, cfg, &setupTracker{fakeTracker: &fakeTracker{}})
	for range 3 {
		send(e.m, keyDown)
	}
	if out := view(e.m); !strings.Contains(out, "enter abrir") || !strings.Contains(out, "d detectar desde Jira") {
		t.Fatalf("con qa_jira completo, enter abre el formulario y d detecta:\n%s", out)
	}
	send(e.m, press('d', "d"))
	send(e.m, keyEnter) // OPS, from the config
	out := view(e.m)
	if !strings.Contains(out, "Propuesto desde tu configuración") || !strings.Contains(out, "Testing, Aprobado, Finalizado") || !strings.Contains(out, "enter guardar") {
		t.Fatalf("propone lo guardado, por nombre, listo para guardar:\n%s", out)
	}
	send(e.m, keyEnter)
	if q := e.file.cfg.QAJira; q == nil || !reflect.DeepEqual(q.Statuses, []string{"102", "103", "104"}) || strings.Contains(view(e.m), "Falta el email") {
		t.Errorf("enter guarda lo propuesto, ya con ids, y no pide el email que ya está: %+v\n%s", q, view(e.m))
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
	if out := view(e.m); trk.project != "OPS" || !strings.Contains(out, "Configuración › Detectar desde Jira") || !strings.Contains(out, "Proyecto OPS") {
		t.Errorf("qa setup -j lee el proyecto de una vez, aunque no esté en qa_jira:\n%s", out)
	}
}
