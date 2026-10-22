package qa

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

type fakeAPI struct {
	provider.QATracker
	provider.DocStore

	me          *provider.User
	meErr       error
	review      []provider.QAIssue
	reviewErr   error
	assigned    []provider.QAIssue
	docLinks    map[string]bool
	docErr      map[string]error
	novedad     map[string]string
	novedadErr  error
	byTitle     *provider.DocRef
	sections    []string
	sectionsErr error
	qaPages     []provider.DocRef

	gotScope          provider.QAScope
	gotQAEmail        string
	gotLinkPrefixes   []string
	gotTitleSearch    *string
	locations         []provider.DocRef
	gotLocationPrefix string
	commentsRead      []string
	created           struct{ near, title string }
	updated           struct {
		id  string
		doc provider.Document
	}
}

func (f *fakeAPI) Me(context.Context) (*provider.User, error) { return f.me, f.meErr }

func (f *fakeAPI) ReviewTasks(_ context.Context, sc provider.QAScope) ([]provider.QAIssue, error) {
	f.gotScope = sc
	return f.review, f.reviewErr
}

func (f *fakeAPI) AssignedTasks(_ context.Context, _ provider.QAScope, email string) ([]provider.QAIssue, error) {
	f.gotQAEmail = email
	return f.assigned, nil
}

func (f *fakeAPI) HasDocumentLink(_ context.Context, key string, prefixes []string) (bool, error) {
	f.gotLinkPrefixes = prefixes
	return f.docLinks[key], f.docErr[key]
}

func (f *fakeAPI) Observations(_ context.Context, key string) (string, error) {
	f.commentsRead = append(f.commentsRead, key)
	return f.novedad[key], f.novedadErr
}

func (f *fakeAPI) FindByTitle(context.Context, string) (*provider.DocRef, error) {
	return f.byTitle, nil
}

func (f *fakeAPI) Sections(context.Context, provider.DocRef, []string) ([]string, error) {
	return f.sections, f.sectionsErr
}

func (f *fakeAPI) SearchTitle(_ context.Context, text string) ([]provider.DocRef, error) {
	f.gotTitleSearch = &text
	return f.qaPages, nil
}

func (f *fakeAPI) Locations(_ context.Context, q provider.LocationQuery) (*provider.Locations, error) {
	f.gotLocationPrefix = q.TitlePrefix
	return &provider.Locations{Docs: f.locations}, nil
}

func (f *fakeAPI) Create(_ context.Context, near provider.DocRef, doc provider.Document) (*provider.DocRef, error) {
	f.created.near, f.created.title = near.ID, doc.Title
	return &provider.DocRef{ID: "new", URL: "https://example.test/new"}, nil
}

func (f *fakeAPI) Update(_ context.Context, r provider.DocRef, doc provider.Document) (*provider.DocRef, error) {
	f.updated.id, f.updated.doc = r.ID, doc
	return &provider.DocRef{ID: r.ID, URL: "https://example.test/" + r.ID}, nil
}

func newSession(api *fakeAPI, t *Target) *Session { return NewSession(api, api, t) }

var ctx = context.Background()

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

func TestResolveSprintMode(t *testing.T) {
	tg := sprintTarget(t)
	if tg.Kanban {
		t.Fatal("con sprint no debe ser Kanban")
	}
	if tg.SprintName != "Pagos_Sprint 12" {
		t.Errorf("SprintName = %q", tg.SprintName)
	}
	if tg.Title != "Consolidado de Pruebas QA - Pagos - Sprint 12" {
		t.Errorf("Title = %q", tg.Title)
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
	if tg, _ := Resolve(validConfig(), Request{}, monday); tg.SpaceKey != "" {
		t.Errorf("sin flag queda vacío: lo completa el perfil, obtuve %q", tg.SpaceKey)
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
		s := newSession(&fakeAPI{me: &provider.User{Email: c.authed}}, tg)
		err := s.CheckAccount(ctx)
		if c.wantErr != errors.Is(err, ErrNotQAAccount) {
			t.Errorf("%s: error = %v", c.name, err)
		}
		if s.Account == nil || s.Account.Email != c.authed {
			t.Errorf("%s: Account debe quedar para mostrarla", c.name)
		}
	}
}

func TestCheckAccountPropagatesAuthError(t *testing.T) {
	s := newSession(&fakeAPI{meErr: errors.New("401")}, sprintTarget(t))
	if err := s.CheckAccount(ctx); err == nil || errors.Is(err, ErrNotQAAccount) {
		t.Errorf("un fallo de autenticación no es ErrNotQAAccount: %v", err)
	}
}

func TestCollectSprintLinksReviewTasks(t *testing.T) {
	api := &fakeAPI{
		review: []provider.QAIssue{{Key: "OPS-7"}, {Key: "OPS-8"}, {Key: "OPS-21", Summary: "Revisión de Tarea - OPS-8"}},
		assigned: []provider.QAIssue{
			{Key: "OPS-20", URL: "https://example.test/browse/OPS-20", Summary: "Revisión de Tarea - OPS-7"},
		},
	}
	s := newSession(api, sprintTarget(t))
	if err := s.Collect(ctx); err != nil {
		t.Fatal(err)
	}
	if api.gotScope != (provider.QAScope{SprintName: "Pagos_Sprint 12", Module: "Pagos"}) || api.gotQAEmail != "qa@example.test" {
		t.Errorf("consulta mal armada: %+v qa=%q", api.gotScope, api.gotQAEmail)
	}
	if len(s.Tasks) != 2 {
		t.Errorf("las tareas de revisión no son filas del reporte: %+v", s.Tasks)
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
	if err := newSession(api, tg).Collect(ctx); err != nil {
		t.Fatal(err)
	}
	if api.gotScope != (provider.QAScope{Since: "2026-09-07"}) {
		t.Errorf("alcance = %+v, esperaba desde 2026-09-07", api.gotScope)
	}
}

func TestCollectPropagatesErrors(t *testing.T) {
	s := newSession(&fakeAPI{reviewErr: errors.New("timeout")}, sprintTarget(t))
	if err := s.Collect(ctx); err == nil {
		t.Error("si la búsqueda falla no hay reporte que armar")
	}
}

// deployTemplate writes a deploy template that only changes the prefix.
func deployTemplate(t *testing.T, dir, prefix string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "template.yaml"), []byte("prefix: \""+prefix+"\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDeployPrefixesOfEveryTemplate(t *testing.T) {
	t.Chdir(t.TempDir())
	repo, profileTemplates := t.TempDir(), t.TempDir()
	deployTemplate(t, filepath.Join(repo, ".draco", "templates", "deploy"), "Despliegue Pagos")
	deployTemplate(t, filepath.Join(profileTemplates, "deploy"), "Despliegue Servicios")
	cfg := &config.Config{
		DefaultProfile: "equipo",
		Profiles: map[string]*config.Profile{
			"equipo":    {Tracker: config.ProviderConfig{Provider: "jira"}, Docs: config.ProviderConfig{Provider: "confluence", Space: "DEV"}},
			"servicios": {Tracker: config.ProviderConfig{Provider: "jira"}, Docs: config.ProviderConfig{Provider: "confluence", Space: "SRV"}, Template: profileTemplates},
		},
		Projects: map[string]*config.ProjectConfig{
			"pagos":     {BackendPath: repo},
			"api-pagos": {BackendPath: t.TempDir(), Profile: "servicios"},
			"roto":      {BackendPath: t.TempDir(), Profile: "no-existe"},
		},
	}
	var warnings []string
	got := deployPrefixes(cfg, filepath.Join(t.TempDir(), "sin-plantillas"), func(w string) { warnings = append(warnings, w) })
	want := []string{"Documento de Despliegue", "Despliegue Servicios", "Despliegue Pagos"}
	if !slices.Equal(got, want) {
		t.Errorf("el de siempre, el de la plantilla del perfil y el del repo, sin repetir: %q", got)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "proyecto roto") {
		t.Errorf("un perfil que no existe se avisa y no detiene el consolidado: %q", warnings)
	}
}

func TestEvaluateTaskFailedLookupIsUnknown(t *testing.T) {
	api := &fakeAPI{docErr: map[string]error{"OPS-7": errors.New("403")}}
	s := newSession(api, sprintTarget(t))
	s.Tasks = []provider.QAIssue{{Key: "OPS-7"}}

	warnings := s.EvaluateTask(ctx, 0)
	if !slices.Equal(api.gotLinkPrefixes, []string{"Documento de Despliegue"}) {
		t.Errorf("el enlace se busca con el prefijo de siempre: %q", api.gotLinkPrefixes)
	}
	if !s.Tasks[0].DeployDocUnknown || s.Tasks[0].HasDeployDoc {
		t.Errorf("una consulta fallida es desconocida, no un control fallido: %+v", s.Tasks[0])
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "OPS-7") {
		t.Errorf("esperaba un aviso con la tarea: %v", warnings)
	}

	api.docErr = nil
	api.docLinks = map[string]bool{"OPS-7": true}
	if w := s.EvaluateTask(ctx, 0); len(w) != 0 || s.Tasks[0].DeployDocUnknown || !s.Tasks[0].HasDeployDoc {
		t.Errorf("reevaluar debe limpiar el estado: %+v %v", s.Tasks[0], w)
	}
}

func TestEvaluateTaskObservations(t *testing.T) {
	api := &fakeAPI{novedad: map[string]string{"OPS-7": "falta validar el monto"}}
	s := newSession(api, sprintTarget(t))
	s.Tasks = []provider.QAIssue{{Key: "OPS-7", HasCodingErrors: true}, {Key: "OPS-8"}}

	s.EvaluateTask(ctx, 0)
	s.EvaluateTask(ctx, 1)
	if s.Tasks[0].Observations != "falta validar el monto" {
		t.Errorf("observaciones = %q", s.Tasks[0].Observations)
	}
	if !reflect.DeepEqual(api.commentsRead, []string{"OPS-7"}) {
		t.Errorf("solo se leen comentarios de tareas con errores: %v", api.commentsRead)
	}

	api.novedadErr = errors.New("500")
	if w := s.EvaluateTask(ctx, 0); len(w) != 1 || !strings.Contains(s.Tasks[0].Observations, "no se pudieron leer") {
		t.Errorf("un fallo debe decirse en la celda y en un aviso: %q %v", s.Tasks[0].Observations, w)
	}
}

func TestDraftUpdateKeepsWhatTheReportHas(t *testing.T) {
	api := &fakeAPI{sections: []string{"Observación", "Consideraciones"}}
	s := newSession(api, sprintTarget(t))
	s.Existing = &provider.DocRef{ID: "p1"}

	d, err := s.DraftUpdate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.NotPreserved) != 0 || d.Replaces.ID != "p1" {
		t.Errorf("borrador = %+v", d)
	}
}

func TestDraftUpdateRefusesWhenPageUnreadable(t *testing.T) {
	s := newSession(&fakeAPI{sectionsErr: errors.New("timeout")}, sprintTarget(t))
	s.Existing = &provider.DocRef{ID: "p1"}
	if d, err := s.DraftUpdate(ctx); err == nil || d != nil {
		t.Error("sin leer la página actual no se arma un reemplazo")
	}
}

func TestDraftUpdateReportsMissingSections(t *testing.T) {
	api := &fakeAPI{sections: []string{"Consideraciones"}}
	s := newSession(api, sprintTarget(t))
	s.Existing = &provider.DocRef{ID: "p1"}

	d, err := s.DraftUpdate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d.NotPreserved, []string{"Observación"}) {
		t.Errorf("NotPreserved = %v", d.NotPreserved)
	}
}

func TestLocations(t *testing.T) {
	api := &fakeAPI{}
	tg, _ := Resolve(validConfig(), Request{}, monday)
	if _, err := newSession(api, tg).Locations(ctx); !errors.Is(err, ErrNoLocations) {
		t.Errorf("sin consolidados previos esperaba ErrNoLocations, obtuve %v", err)
	}
	if api.gotTitleSearch == nil || *api.gotTitleSearch != "Consolidado de Pruebas QA" {
		t.Errorf("en Kanban se buscan consolidados de cualquier módulo: %v", api.gotTitleSearch)
	}

	api.locations = []provider.DocRef{{ID: "./"}}
	got, err := newSession(api, tg).Locations(ctx)
	if err != nil || len(got) != 1 || got[0].ID != "./" || api.gotLocationPrefix != "Consolidado de Pruebas QA" {
		t.Errorf("sin consolidados se usan las ubicaciones del destino: %v %v (prefijo %q)", got, err, api.gotLocationPrefix)
	}

	api.qaPages = []provider.DocRef{{ID: "q1"}}
	api.gotLocationPrefix = ""
	if got, err := newSession(api, sprintTarget(t)).Locations(ctx); err != nil || got[0].ID != "q1" || *api.gotTitleSearch != "Consolidado de Pruebas QA - Pagos" || api.gotLocationPrefix != "" {
		t.Errorf("en Sprint se buscan los del módulo, sin respaldo si los hay: %v %q %v", got, *api.gotTitleSearch, err)
	}
}

func TestPublishUpdateKeepsHandWritten(t *testing.T) {
	api := &fakeAPI{}
	s := newSession(api, sprintTarget(t))
	res, err := s.Publish(ctx, &Draft{Title: "T", Markdown: "md", Replaces: &provider.DocRef{ID: "p1"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Updated || api.updated.id != "p1" || api.updated.doc.Markdown != "md" {
		t.Errorf("update mal hecho: %+v %+v", res, api.updated)
	}
	if !reflect.DeepEqual(api.updated.doc.Preserve, []string{"Observación", "Consideraciones"}) {
		t.Errorf("el destino debe conservar lo escrito a mano: %v", api.updated.doc.Preserve)
	}
}

func TestPublishCreateNextToSibling(t *testing.T) {
	api := &fakeAPI{}
	s := newSession(api, sprintTarget(t))
	d := draft(t, s)

	if _, err := s.Publish(ctx, d, nil); err == nil {
		t.Error("crear sin ubicación debe fallar")
	}
	res, err := s.Publish(ctx, d, &provider.DocRef{ID: "q1"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated || api.created.near != "q1" || api.created.title != s.Target.Title {
		t.Errorf("create mal hecho: %+v", api.created)
	}
}

func draft(t *testing.T, s *Session) *Draft {
	t.Helper()
	d, err := s.DraftCreate()
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestCustomReviewPrefixesComeFromConfig(t *testing.T) {
	cfg := validConfig()
	cfg.QAJira.ReviewPrefixes = []string{"Validación"}
	tg, err := Resolve(cfg, Request{Sprint: 12, Module: "Pagos"}, monday)
	if err != nil {
		t.Fatal(err)
	}
	api := &fakeAPI{review: []provider.QAIssue{{Key: "OPS-7"}, {Key: "OPS-9", Summary: "Validación de OPS-7"}, {Key: "OPS-10", Summary: "QA de OPS-7"}}}
	s := newSession(api, tg)
	if err := s.Collect(ctx); err != nil {
		t.Fatal(err)
	}
	if len(s.Tasks) != 2 || s.Tasks[1].Key != "OPS-10" {
		t.Errorf("con prefijos propios solo «Validación» es tarea de revisión: %+v", s.Tasks)
	}
}
