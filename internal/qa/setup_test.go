package qa

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

type fakeSetup struct {
	statuses   []provider.Status
	fields     []provider.Field
	err        error
	gotProject string
}

func (f *fakeSetup) ProjectStatuses(_ context.Context, project string) ([]provider.Status, error) {
	f.gotProject = project
	return f.statuses, f.err
}

func (f *fakeSetup) QAFields(context.Context) ([]provider.Field, error) { return f.fields, nil }

func jiraForSetup() *fakeSetup {
	return &fakeSetup{
		statuses: []provider.Status{
			{ID: "1", Name: "Por hacer"}, {ID: "2", Name: "En curso"}, {ID: "3", Name: "Testing"},
			{ID: "4", Name: "Aprobado"}, {ID: "5", Name: "Finalizado", Done: true}, {ID: "6", Name: "Descartado", Done: true},
		},
		fields: []provider.Field{
			{ID: "customfield_1", Name: "Desarrollo", Kind: provider.FieldPRStatus},
			{ID: "customfield_2", Name: "Errores", Kind: provider.FieldNumber},
			{ID: "customfield_3", Name: "Reprocesos", Kind: provider.FieldNumber},
		},
	}
}

func readSetup(t *testing.T, src *fakeSetup, project string) *Setup {
	t.Helper()
	s := NewSetup(src)
	if err := s.Read(context.Background(), project); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSetupReadNeedsAProjectWithStatuses(t *testing.T) {
	if err := NewSetup(jiraForSetup()).Read(context.Background(), " "); err == nil {
		t.Error("sin clave de proyecto no se consulta")
	}
	err := NewSetup(&fakeSetup{}).Read(context.Background(), "OPS")
	if !errors.Is(err, ErrNoStatuses) || !strings.Contains(err.Error(), "'OPS'") {
		t.Errorf("un proyecto sin estados se explica: %v", err)
	}
	src := jiraForSetup()
	src.err = errors.New("recurso no encontrado (404)")
	if err := NewSetup(src).Read(context.Background(), "XYZ"); err == nil || src.gotProject != "XYZ" {
		t.Errorf("el error de Jira llega tal cual: %v", err)
	}
}

func TestSetupProposesFromScratch(t *testing.T) {
	s := readSetup(t, jiraForSetup(), "OPS")
	got := s.Propose(nil)
	want := Choice{Testing: "Testing", Statuses: []string{"3", "5", "6"}, DevStatus: "customfield_1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sin config: Testing, los finalizados y el campo de desarrollo; los contadores sin elegir:\n%+v", got)
	}
	if n := len(s.FieldsOf(provider.FieldNumber)); n != 2 {
		t.Errorf("los contadores se eligen entre los campos numéricos: %d", n)
	}
}

func TestSetupProposesWhatTheConfigHas(t *testing.T) {
	s := readSetup(t, jiraForSetup(), "OPS")
	current := &config.QAJiraConfig{
		Project: "OPS", Statuses: []string{"Aprobado", "3", "99"}, TestingStatus: "testing",
		Fields: config.QAJiraFields{CodingErrors: "customfield_2", DevReturns: "customfield_3", DevStatus: "customfield_9"},
	}
	got := s.Propose(current)
	want := Choice{Testing: "Testing", Statuses: []string{"4", "3"}, CodingErrors: "customfield_2", DevReturns: "customfield_3", DevStatus: "customfield_1"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("lo de la config que sigue en Jira, por id o por nombre; lo que ya no existe se descarta:\n%+v", got)
	}
}

func TestSetupApplyWritesTheProject(t *testing.T) {
	s := readSetup(t, jiraForSetup(), "OPS")
	c := Choice{Testing: "Testing", Statuses: []string{"3", "4", "5"}, CodingErrors: "customfield_2", DevReturns: "customfield_3", DevStatus: "customfield_1"}
	q, err := s.Apply(nil, c)
	if err != nil {
		t.Fatal(err)
	}
	if q.Project != "OPS" || !reflect.DeepEqual(q.Statuses, []string{"3", "4", "5"}) || q.TestingStatus != "" || q.Fields.DevStatus != "customfield_1" {
		t.Errorf("sin qa_jira se crea con el proyecto por defecto: %+v", q)
	}

	current := &config.QAJiraConfig{Project: "PAG", Statuses: []string{"8"}, ReviewPrefixes: []string{"Validación"},
		Fields: config.QAJiraFields{CodingErrors: "customfield_7", DevReturns: "customfield_8", DevStatus: "customfield_9"}}
	q, err = s.Apply(current, c)
	if err != nil {
		t.Fatal(err)
	}
	if q.Project != "PAG" || q.ReviewPrefixes[0] != "Validación" || !reflect.DeepEqual(q.Projects["OPS"].Statuses, []string{"3", "4", "5"}) {
		t.Errorf("con otro por defecto, OPS se agrega en projects y lo demás se conserva: %+v", q)
	}
	if current.Projects != nil || current.Fields.DevStatus != "customfield_9" {
		t.Errorf("Apply no modifica la config que recibe: %+v", current)
	}

	if _, err := s.Apply(nil, Choice{Testing: "Testing"}); err == nil {
		t.Error("sin estados no se guarda")
	}
	if _, err := s.Apply(nil, Choice{Testing: "Testing", Statuses: []string{"3"}}); err == nil || !strings.Contains(err.Error(), "fields.coding_errors") {
		t.Errorf("sin los contadores, Validate lo dice: %v", err)
	}
}
