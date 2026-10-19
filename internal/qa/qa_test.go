package qa

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kaelence-01/draco-hq/internal/atlassian"
	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/document"
)

// fakeAPI records what the workflow asks Atlassian and answers from its fields.
type fakeAPI struct {
	me         *atlassian.JiraUser
	meErr      error
	review     []atlassian.QAIssue
	reviewErr  error
	assigned   []atlassian.QAIssue
	docLinks   map[string]bool
	docErr     map[string]error
	novedad    map[string]string
	novedadErr error
	byTitle    *atlassian.Page
	current    *atlassian.PageADF
	currentErr error
	qaPages    []atlassian.Page
	pages      map[string]*atlassian.Page

	gotSchema     atlassian.QASchema
	gotSprintName string
	gotModule     string
	gotSince      string
	gotQAEmail    string
	gotPagesFor   *string
	commentsRead  []string
	created       struct{ spaceID, parentID, title string }
	updated       struct {
		id      string
		version int
	}
}

func (f *fakeAPI) Whoami() (*atlassian.JiraUser, error) { return f.me, f.meErr }

func (f *fakeAPI) GetQATasksForReview(s atlassian.QASchema, sprintName, module string) ([]atlassian.QAIssue, error) {
	f.gotSchema, f.gotSprintName, f.gotModule = s, sprintName, module
	return f.review, f.reviewErr
}

func (f *fakeAPI) GetQATasksAsAssignee(_ atlassian.QASchema, sprintName, qaEmail string) ([]atlassian.QAIssue, error) {
	f.gotQAEmail = qaEmail
	return f.assigned, nil
}

func (f *fakeAPI) GetQATasksForReviewKanban(s atlassian.QASchema, since string) ([]atlassian.QAIssue, error) {
	f.gotSchema, f.gotSince = s, since
	return f.review, f.reviewErr
}

func (f *fakeAPI) GetQATasksAsAssigneeKanban(_ atlassian.QASchema, _, qaEmail string) ([]atlassian.QAIssue, error) {
	f.gotQAEmail = qaEmail
	return f.assigned, nil
}

func (f *fakeAPI) HasDeployDocLink(key string) (bool, error) { return f.docLinks[key], f.docErr[key] }

func (f *fakeAPI) GetNovedadComment(key string) (string, error) {
	f.commentsRead = append(f.commentsRead, key)
	return f.novedad[key], f.novedadErr
}

func (f *fakeAPI) FindPageByTitle(string, string) (*atlassian.Page, error) { return f.byTitle, nil }
func (f *fakeAPI) GetPageADF(string) (*atlassian.PageADF, error)           { return f.current, f.currentErr }
func (f *fakeAPI) GetPage(id string) (*atlassian.Page, error)              { return f.pages[id], nil }

func (f *fakeAPI) FindQAPagesForModule(module, _ string) ([]atlassian.Page, error) {
	f.gotPagesFor = &module
	return f.qaPages, nil
}

func (f *fakeAPI) CreatePage(spaceID, parentID, title string, _ map[string]any) (*atlassian.Page, error) {
	f.created.spaceID, f.created.parentID, f.created.title = spaceID, parentID, title
	return &atlassian.Page{ID: "new", WebURL: "https://example.test/new"}, nil
}

func (f *fakeAPI) UpdatePage(id, _ string, version int, _ map[string]any) (*atlassian.Page, error) {
	f.updated.id, f.updated.version = id, version
	return &atlassian.Page{ID: id, WebURL: "https://example.test/" + id}, nil
}

// monday is 2026-09-21; ten business days earlier is 2026-09-07.
var monday = time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)

func validConfig() *config.Config {
	return &config.Config{
		QAEmail:            "qa@example.test",
		ConfluenceSpaceKey: "DEV",
		QAReport:           &config.QAReportConfig{QA: "Equipo QA"},
		QAJira: &config.QAJiraConfig{
			Project:  "OPS",
			Statuses: []string{"10001", "En Revisión"},
			Fields: config.QAJiraFields{
				CodingErrors: "customfield_1",
				DevReturns:   "customfield_2",
				DevStatus:    "customfield_3",
			},
		},
	}
}

func sprintTarget(t *testing.T) *Target {
	t.Helper()
	tg, err := Resolve(validConfig(), Request{Sprint: 12, Module: "Pagos"}, monday)
	if err != nil {
		t.Fatal(err)
	}
	return tg
}

// --- Resolve ---

func TestResolveSprintMode(t *testing.T) {
	tg := sprintTarget(t)
	if tg.Kanban {
		t.Fatal("con sprint no debe ser Kanban")
	}
	if tg.SprintName != "Pagos_Sprint 12" {
		t.Errorf("SprintName = %q", tg.SprintName)
	}
	if tg.Title != document.BuildQATitle("Pagos", 12) {
		t.Errorf("Title = %q", tg.Title)
	}
	if tg.Schema.TestingStatus != config.DefaultTestingStatus {
		t.Errorf("sin testing_status debe usarse el default, obtuve %q", tg.Schema.TestingStatus)
	}
	if tg.Schema.FieldDevStatus != "customfield_3" || tg.Schema.Project != "OPS" {
		t.Errorf("schema mal convertido: %+v", tg.Schema)
	}
	if len(tg.Warnings) != 0 {
		t.Errorf("no esperaba warnings: %v", tg.Warnings)
	}
}

func TestResolveSprintNameFromConfig(t *testing.T) {
	cfg := validConfig()
	cfg.QAJira.SprintName = "Sprint {sprint} - {module}"
	tg, err := Resolve(cfg, Request{Sprint: 3, Module: "Facturacion"}, monday)
	if err != nil {
		t.Fatal(err)
	}
	if tg.SprintName != "Sprint 3 - Facturacion" {
		t.Errorf("SprintName = %q", tg.SprintName)
	}
}

func TestResolveKanbanMode(t *testing.T) {
	tg, err := Resolve(validConfig(), Request{Module: "Pagos"}, monday)
	if err != nil {
		t.Fatal(err)
	}
	if !tg.Kanban || tg.Since != "2026-09-07" || tg.Period != "07/09 al 21/09/2026" {
		t.Errorf("ventana Kanban mal resuelta: %+v", tg)
	}
	if tg.Module != "" {
		t.Errorf("en Kanban el módulo se ignora, quedó %q", tg.Module)
	}
	if len(tg.Warnings) != 1 || !strings.Contains(tg.Warnings[0], "Kanban") {
		t.Errorf("esperaba un aviso por el módulo ignorado: %v", tg.Warnings)
	}
}

func TestResolveRejectsBadRequests(t *testing.T) {
	cfg := validConfig()
	if _, err := Resolve(cfg, Request{Sprint: 12}, monday); err == nil {
		t.Error("modo Sprint sin módulo debe fallar")
	}
	if _, err := Resolve(cfg, Request{Sprint: -1, Module: "Pagos"}, monday); err == nil {
		t.Error("un sprint negativo debe fallar")
	}
	cfg.QAJira.Fields.DevStatus = ""
	if _, err := Resolve(cfg, Request{}, monday); err == nil || !strings.Contains(err.Error(), "qa_jira") {
		t.Errorf("qa_jira incompleto debe fallar antes de consultar: %v", err)
	}
}

func TestResolveSpace(t *testing.T) {
	tg, err := Resolve(validConfig(), Request{Space: "FLAG"}, monday)
	if err != nil || tg.SpaceKey != "FLAG" {
		t.Errorf("el flag debe ganar: %q %v", tg.SpaceKey, err)
	}
	cfg := validConfig()
	cfg.ConfluenceSpaceKey = ""
	if _, err := Resolve(cfg, Request{}, monday); !errors.Is(err, ErrNoSpace) {
		t.Errorf("esperaba ErrNoSpace, obtuve %v", err)
	}
}

func TestResolveWithoutReportNamesWarns(t *testing.T) {
	cfg := validConfig()
	cfg.QAReport = nil
	tg, err := Resolve(cfg, Request{}, monday)
	if err != nil {
		t.Fatal(err)
	}
	if len(tg.Warnings) != 1 || !strings.Contains(tg.Warnings[0], "qa_report") {
		t.Errorf("esperaba un aviso por qa_report: %v", tg.Warnings)
	}
}

// --- CheckAccount ---

func TestCheckAccount(t *testing.T) {
	cases := []struct {
		name    string
		qaEmail string
		authed  string
		wantErr bool
	}{
		{"coincide sin importar mayúsculas", "qa@example.test", "QA@Example.test", false},
		{"otra cuenta", "qa@example.test", "dev@example.test", true},
		{"qa_email sin configurar", "", "dev@example.test", true},
	}
	for _, c := range cases {
		tg := sprintTarget(t)
		tg.QAEmail = c.qaEmail
		s := NewSession(&fakeAPI{me: &atlassian.JiraUser{EmailAddress: c.authed}}, tg)
		err := s.CheckAccount()
		if c.wantErr != errors.Is(err, ErrNotQAAccount) {
			t.Errorf("%s: error = %v", c.name, err)
		}
		if s.Account == nil || s.Account.EmailAddress != c.authed {
			t.Errorf("%s: Account debe quedar para mostrarla", c.name)
		}
	}
}

func TestCheckAccountPropagatesAuthError(t *testing.T) {
	s := NewSession(&fakeAPI{meErr: errors.New("401")}, sprintTarget(t))
	if err := s.CheckAccount(); err == nil || errors.Is(err, ErrNotQAAccount) {
		t.Errorf("un fallo de autenticación no es ErrNotQAAccount: %v", err)
	}
}

// --- Collect ---

func TestCollectSprintLinksReviewTasks(t *testing.T) {
	api := &fakeAPI{
		review: []atlassian.QAIssue{{Key: "OPS-7"}, {Key: "OPS-8"}},
		assigned: []atlassian.QAIssue{
			{Key: "OPS-20", URL: "https://example.test/browse/OPS-20", Summary: "Revisión de Tarea - OPS-7"},
		},
	}
	s := NewSession(api, sprintTarget(t))
	if err := s.Collect(); err != nil {
		t.Fatal(err)
	}
	if api.gotSprintName != "Pagos_Sprint 12" || api.gotModule != "Pagos" || api.gotQAEmail != "qa@example.test" {
		t.Errorf("consulta mal armada: sprint=%q módulo=%q qa=%q", api.gotSprintName, api.gotModule, api.gotQAEmail)
	}
	if api.gotSchema.Project != "OPS" {
		t.Errorf("no se pasó el schema: %+v", api.gotSchema)
	}
	if s.Tasks[0].ReviewTaskKey != "OPS-20" || s.Tasks[0].ReviewTaskURL == "" {
		t.Errorf("OPS-7 debe enlazar su tarea de revisión: %+v", s.Tasks[0])
	}
	if s.Tasks[1].ReviewTaskKey != "" {
		t.Errorf("OPS-8 no tiene tarea de revisión: %+v", s.Tasks[1])
	}
}

func TestCollectKanbanUsesWindow(t *testing.T) {
	tg, _ := Resolve(validConfig(), Request{}, monday)
	api := &fakeAPI{}
	if err := NewSession(api, tg).Collect(); err != nil {
		t.Fatal(err)
	}
	if api.gotSince != "2026-09-07" {
		t.Errorf("since = %q, esperaba 2026-09-07", api.gotSince)
	}
}

func TestCollectPropagatesErrors(t *testing.T) {
	s := NewSession(&fakeAPI{reviewErr: errors.New("timeout")}, sprintTarget(t))
	if err := s.Collect(); err == nil {
		t.Error("si la búsqueda falla no hay reporte que armar")
	}
}

// --- EvaluateTask ---

func TestEvaluateTaskFailedLookupIsUnknown(t *testing.T) {
	api := &fakeAPI{docErr: map[string]error{"OPS-7": errors.New("403")}}
	s := NewSession(api, sprintTarget(t))
	s.Tasks = []atlassian.QAIssue{{Key: "OPS-7"}}

	warnings := s.EvaluateTask(0)
	if !s.Tasks[0].DeployDocUnknown || s.Tasks[0].HasDeployDoc {
		t.Errorf("una consulta fallida es desconocida, no un control fallido: %+v", s.Tasks[0])
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "OPS-7") {
		t.Errorf("esperaba un aviso con la tarea: %v", warnings)
	}

	// Re-evaluating after the error clears is not stuck on unknown.
	api.docErr = nil
	api.docLinks = map[string]bool{"OPS-7": true}
	if w := s.EvaluateTask(0); len(w) != 0 || s.Tasks[0].DeployDocUnknown || !s.Tasks[0].HasDeployDoc {
		t.Errorf("reevaluar debe limpiar el estado: %+v %v", s.Tasks[0], w)
	}
}

func TestEvaluateTaskObservations(t *testing.T) {
	api := &fakeAPI{novedad: map[string]string{"OPS-7": "falta validar el monto"}}
	s := NewSession(api, sprintTarget(t))
	s.Tasks = []atlassian.QAIssue{{Key: "OPS-7", HasCodingErrors: true}, {Key: "OPS-8"}}

	s.EvaluateTask(0)
	s.EvaluateTask(1)
	if s.Tasks[0].Observations != "falta validar el monto" {
		t.Errorf("observaciones = %q", s.Tasks[0].Observations)
	}
	if !reflect.DeepEqual(api.commentsRead, []string{"OPS-7"}) {
		t.Errorf("solo se leen comentarios de tareas con errores: %v", api.commentsRead)
	}

	api.novedadErr = errors.New("500")
	if w := s.EvaluateTask(0); len(w) != 1 || !strings.Contains(s.Tasks[0].Observations, "no se pudieron leer") {
		t.Errorf("un fallo debe decirse en la celda y en un aviso: %q %v", s.Tasks[0].Observations, w)
	}
}

// --- Draft ---

// roundTrip passes a document through JSON, as it comes back from Confluence.
func roundTrip(t *testing.T, doc map[string]any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func publishedReport(t *testing.T, kept document.QAPreserved) map[string]any {
	t.Helper()
	return roundTrip(t, document.BuildQA(document.QADoc{Sprint: 12, Module: "Pagos", Preserved: kept}))
}

func TestDraftUpdatePreservesHandWrittenParts(t *testing.T) {
	note := []any{map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": "Sin bloqueos"}}}}
	consider := []any{map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": "Reprobar en staging"}}}}
	api := &fakeAPI{current: &atlassian.PageADF{
		Page: atlassian.Page{ID: "p1", Version: 4},
		ADF:  publishedReport(t, document.QAPreserved{Observation: note, Considerations: consider}),
	}}
	s := NewSession(api, sprintTarget(t))
	s.Existing = &atlassian.Page{ID: "p1"}

	d, err := s.DraftUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if len(d.NotPreserved) != 0 {
		t.Errorf("todo se pudo leer, NotPreserved = %v", d.NotPreserved)
	}
	kept := document.ExtractQAPreserved(roundTrip(t, d.ADF))
	if len(kept.Observation) != 1 || len(kept.Considerations) != 1 {
		t.Errorf("el borrador perdió lo escrito a mano: %#v", kept)
	}
}

func TestDraftUpdateRefusesWhenPageUnreadable(t *testing.T) {
	s := NewSession(&fakeAPI{currentErr: errors.New("timeout")}, sprintTarget(t))
	s.Existing = &atlassian.Page{ID: "p1"}
	if d, err := s.DraftUpdate(); err == nil || d != nil {
		t.Error("sin leer la página actual no se arma un reemplazo")
	}
}

func TestDraftUpdateReportsLostParts(t *testing.T) {
	api := &fakeAPI{current: &atlassian.PageADF{Page: atlassian.Page{ID: "p1"}}} // not ADF
	s := NewSession(api, sprintTarget(t))
	s.Existing = &atlassian.Page{ID: "p1"}

	d, err := s.DraftUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d.NotPreserved, []string{PartObservation, PartConsiderations}) {
		t.Errorf("NotPreserved = %v", d.NotPreserved)
	}
	if len(d.Warnings) != 1 {
		t.Errorf("esperaba un aviso por el formato: %v", d.Warnings)
	}
}

// --- Locations and Publish ---

func TestLocations(t *testing.T) {
	api := &fakeAPI{}
	tg, _ := Resolve(validConfig(), Request{}, monday)
	if _, err := NewSession(api, tg).Locations(); !errors.Is(err, ErrNoLocations) {
		t.Errorf("sin consolidados previos esperaba ErrNoLocations, obtuve %v", err)
	}
	if api.gotPagesFor == nil || *api.gotPagesFor != "" {
		t.Errorf("en Kanban se buscan consolidados de cualquier módulo")
	}

	api.qaPages = []atlassian.Page{{ID: "q1"}}
	if _, err := NewSession(api, sprintTarget(t)).Locations(); err != nil || *api.gotPagesFor != "Pagos" {
		t.Errorf("en Sprint se buscan los del módulo: %q %v", *api.gotPagesFor, err)
	}
}

func TestPublishUpdateUsesCurrentVersion(t *testing.T) {
	api := &fakeAPI{}
	s := NewSession(api, sprintTarget(t))
	res, err := s.Publish(&Draft{Title: "T", Replaces: &atlassian.PageADF{Page: atlassian.Page{ID: "p1", Version: 4}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Updated || api.updated.id != "p1" || api.updated.version != 4 {
		t.Errorf("update mal hecho: %+v %+v", res, api.updated)
	}
}

func TestPublishCreateNextToSibling(t *testing.T) {
	api := &fakeAPI{pages: map[string]*atlassian.Page{"q1": {ID: "q1", SpaceID: "99", ParentID: "50"}}}
	s := NewSession(api, sprintTarget(t))
	d := s.DraftCreate()

	if _, err := s.Publish(d, nil); err == nil {
		t.Error("crear sin ubicación debe fallar")
	}
	res, err := s.Publish(d, &atlassian.Page{ID: "q1"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated || api.created.spaceID != "99" || api.created.parentID != "50" || api.created.title != s.Target.Title {
		t.Errorf("create mal hecho: %+v", api.created)
	}
}
