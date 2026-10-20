package deploy

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/document"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

type fakeAPI struct {
	provider.DocStore

	issue        *provider.Issue
	byTitle      *provider.DocRef
	byIssue      *provider.DocRef
	byIssueErr   error
	current      *provider.Doc
	currentErr   error
	me           *provider.User
	meErr        error
	lastDocs     []provider.DocRef
	lastFellBack bool
	linkErr      error

	gotLocations provider.LocationQuery
	created      struct{ near, title string }
	updated      struct {
		id      string
		version int
	}
	linked provider.DocRef
}

func (f *fakeAPI) Issue(context.Context, string) (*provider.Issue, error) { return f.issue, nil }
func (f *fakeAPI) Me(context.Context) (*provider.User, error)             { return f.me, f.meErr }

func (f *fakeAPI) LinkDocument(_ context.Context, _ string, doc provider.DocRef) error {
	f.linked = doc
	return f.linkErr
}

func (f *fakeAPI) FindByTitle(context.Context, string) (*provider.DocRef, error) {
	return f.byTitle, nil
}

func (f *fakeAPI) FindByTitleKey(context.Context, string, string) (*provider.DocRef, error) {
	return f.byIssue, f.byIssueErr
}

func (f *fakeAPI) Read(context.Context, provider.DocRef) (*provider.Doc, error) {
	return f.current, f.currentErr
}

func (f *fakeAPI) Locations(_ context.Context, q provider.LocationQuery) (*provider.Locations, error) {
	f.gotLocations = q
	return &provider.Locations{Docs: f.lastDocs, TeamWide: f.lastFellBack}, nil
}

func (f *fakeAPI) Create(_ context.Context, near provider.DocRef, title string, _ map[string]any) (*provider.DocRef, error) {
	f.created.near, f.created.title = near.ID, title
	return &provider.DocRef{ID: "new", URL: "https://example.test/new"}, nil
}

func (f *fakeAPI) Update(_ context.Context, r provider.DocRef, _ string, _ map[string]any) (*provider.DocRef, error) {
	f.updated.id, f.updated.version = r.ID, r.Version
	return &provider.DocRef{ID: r.ID, URL: "https://example.test/" + r.ID}, nil
}

var ctx = context.Background()

func newSession(api *fakeAPI, files ChangedFilesFunc, t *Target) *Session {
	return NewSession(api, api, files, t)
}

func filesOK(hashes []string, _ string) ([]string, error) {
	return []string{"app/Pago.php", "app/Cobro.php", "routes/api.php"}, nil
}

func target() *Target {
	return &Target{
		Issue:    "OPS-7",
		SpaceKey: "DEV",
		Backend:  Repo{Name: "api", Hashes: []string{"a1b2c3d"}},
	}
}

func readySession(api *fakeAPI) *Session {
	s := newSession(api, filesOK, target())
	if err := s.FetchIssue(ctx); err != nil {
		panic(err)
	}
	if err := s.ReadCommits(); err != nil {
		panic(err)
	}
	return s
}

func newAPI() *fakeAPI {
	return &fakeAPI{issue: &provider.Issue{Key: "OPS-7", Summary: "Filtro de pagos", URL: "https://example.test/browse/OPS-7"}}
}

func TestResolveSpacePrecedence(t *testing.T) {
	cfg := &config.Config{
		ConfluenceSpaceKey: "GLOBAL",
		Projects:           map[string]*config.ProjectConfig{"app": {ConfluenceSpaceKey: "PROJ", BackendRepo: "api"}},
	}
	req := Request{Issue: "OPS-1", BackendHashes: []string{"a"}, Project: "app"}

	cases := []struct {
		name  string
		space string
		cfg   *config.Config
		want  string
	}{
		{"flag gana", "FLAG", cfg, "FLAG"},
		{"luego el proyecto", "", cfg, "PROJ"},
		{"luego el global", "", &config.Config{ConfluenceSpaceKey: "GLOBAL", Projects: map[string]*config.ProjectConfig{"app": {BackendRepo: "api"}}}, "GLOBAL"},
	}
	for _, c := range cases {
		r := req
		r.Space = c.space
		got, err := Resolve(c.cfg, r)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got.SpaceKey != c.want {
			t.Errorf("%s: space = %q, esperaba %q", c.name, got.SpaceKey, c.want)
		}
	}
}

func TestResolveWithoutSpaceIsErrNoSpace(t *testing.T) {
	_, err := Resolve(&config.Config{}, Request{Issue: "OPS-1", BackendHashes: []string{"a"}})
	if !errors.Is(err, ErrNoSpace) {
		t.Errorf("esperaba ErrNoSpace, obtuve %v", err)
	}
}

func TestResolveRequiresIssueAndCommits(t *testing.T) {
	cfg := &config.Config{ConfluenceSpaceKey: "DEV"}
	if _, err := Resolve(cfg, Request{BackendHashes: []string{"a"}}); err == nil {
		t.Error("sin issue debe fallar")
	}
	if _, err := Resolve(cfg, Request{Issue: "OPS-1"}); err == nil {
		t.Error("sin commits debe fallar")
	}
}

func TestResolveGuessesRepoNameFromDirAndWarns(t *testing.T) {
	cfg := &config.Config{
		ConfluenceSpaceKey: "DEV",
		DefaultProject:     "app",
		Projects: map[string]*config.ProjectConfig{"app": {
			BackendPath: "/home/dev/repos/pagos-api",
			VCSHost:     "https://bitbucket.org",
			VCSOrg:      "mi-org",
		}},
	}
	got, err := Resolve(cfg, Request{Issue: "OPS-1", BackendHashes: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Backend.Name != "pagos-api" {
		t.Errorf("nombre del repo = %q, esperaba pagos-api", got.Backend.Name)
	}
	if len(got.Warnings) != 1 || !strings.Contains(got.Warnings[0], "pagos-api") {
		t.Errorf("esperaba un aviso por el nombre adivinado, obtuve %v", got.Warnings)
	}
	if got.ProjectName != "app" {
		t.Errorf("proyecto = %q, esperaba el default app", got.ProjectName)
	}
}

func TestResolveRequestOverridesVCS(t *testing.T) {
	cfg := &config.Config{
		ConfluenceSpaceKey: "DEV",
		Projects:           map[string]*config.ProjectConfig{"app": {BackendRepo: "api", VCSHost: "https://bitbucket.org", VCSOrg: "org"}},
	}
	got, err := Resolve(cfg, Request{Issue: "OPS-1", BackendHashes: []string{"a"}, Project: "app", VCSOrg: "fork"})
	if err != nil {
		t.Fatal(err)
	}
	if got.VCSHost != "https://bitbucket.org" || got.VCSOrg != "fork" {
		t.Errorf("vcs = %s / %s, esperaba bitbucket / fork", got.VCSHost, got.VCSOrg)
	}
}

func TestFindExistingFallsBackToIssueSearch(t *testing.T) {
	api := newAPI()
	api.byIssue = &provider.DocRef{ID: "42"}
	s := readySession(api)
	if err := s.FindExisting(ctx); err != nil {
		t.Fatal(err)
	}
	if s.Existing == nil || s.Existing.ID != "42" {
		t.Errorf("Existing = %+v, esperaba la página 42 del fallback", s.Existing)
	}
}

func TestFindExistingIgnoresFallbackError(t *testing.T) {
	api := newAPI()
	api.byIssueErr = errors.New("índice no disponible")
	s := readySession(api)
	if err := s.FindExisting(ctx); err != nil {
		t.Errorf("el fallback es best-effort y no debe fallar: %v", err)
	}
	if s.Existing != nil {
		t.Errorf("Existing = %+v, esperaba nil", s.Existing)
	}
}

func TestReadCommitsKeepsGoingWhenOneSideFails(t *testing.T) {
	tg := target()
	tg.Frontend = Repo{Name: "web", Hashes: []string{"bad"}}
	files := func(h []string, dir string) ([]string, error) {
		if h[0] == "bad" {
			return nil, errors.New("commit no encontrado")
		}
		return filesOK(h, dir)
	}
	s := newSession(newAPI(), files, tg)
	if err := s.ReadCommits(); err != nil {
		t.Fatalf("con un lado bueno no debe fallar: %v", err)
	}
	if !s.PartialFailure() || s.Frontend.Err == nil {
		t.Error("esperaba falla parcial en frontend")
	}
	if s.Backend.Count != 3 || len(s.Backend.Files["app"]) != 2 {
		t.Errorf("backend = %+v, esperaba 3 archivos con 2 en app", s.Backend)
	}
}

func TestReadCommitsFailsWhenEverySideFails(t *testing.T) {
	files := func([]string, string) ([]string, error) { return nil, errors.New("x") }
	if err := newSession(newAPI(), files, target()).ReadCommits(); err == nil {
		t.Error("sin ningún commit legible debe fallar")
	}
}

func considerDoc(text string) map[string]any {
	return map[string]any{"type": "doc", "content": []any{
		map[string]any{"type": "heading", "attrs": map[string]any{"level": 2}, "content": []any{
			map[string]any{"type": "text", "text": document.ConsiderHeading},
		}},
		map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": text}}},
	}}
}

func TestDraftUpdatePreservesConsiderSection(t *testing.T) {
	api := newAPI()
	api.current = &provider.Doc{DocRef: provider.DocRef{ID: "42", Version: 5}, Body: considerDoc("Reiniciar colas a mano")}
	s := readySession(api)
	s.Existing = &provider.DocRef{ID: "42"}

	d, err := s.DraftUpdate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Preserved {
		t.Error("esperaba Preserved = true")
	}
	got := document.ExtractSection(d.ADF, document.ConsiderHeading)
	want := document.ExtractSection(api.current.Body, document.ConsiderHeading)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("la sección no se preservó:\n obtuve %v\n esperaba %v", got, want)
	}
}

func TestDraftUpdateRefusesWhenCurrentPageUnreadable(t *testing.T) {
	api := newAPI()
	api.currentErr = errors.New("timeout")
	s := readySession(api)
	s.Existing = &provider.DocRef{ID: "42"}
	if d, err := s.DraftUpdate(ctx); err == nil || d != nil {
		t.Error("sin poder leer la página actual no debe haber borrador: se sobrescribiría a ciegas")
	}
}

func TestDraftUpdateWarnsOnNonADFPage(t *testing.T) {
	api := newAPI()
	api.current = &provider.Doc{DocRef: provider.DocRef{ID: "42"}}
	s := readySession(api)
	s.Existing = &provider.DocRef{ID: "42"}
	d, err := s.DraftUpdate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if d.Preserved || len(d.Warnings) != 1 {
		t.Errorf("esperaba Preserved=false y un aviso, obtuve %v / %v", d.Preserved, d.Warnings)
	}
}

func TestLocationsFallsBackToTeamWhenUserUnknown(t *testing.T) {
	api := newAPI()
	api.meErr = errors.New("403")
	api.lastDocs = []provider.DocRef{{ID: "1"}}
	loc, err := readySession(api).Locations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !loc.OwnerUnknown || !loc.TeamWide || api.gotLocations.AuthorID != "" {
		t.Errorf("esperaba lista de todo el equipo sin autor, obtuve %+v (autor %q)", loc, api.gotLocations.AuthorID)
	}
}

func TestLocationsWithoutPreviousDocs(t *testing.T) {
	api := newAPI()
	api.me = &provider.User{ID: "me"}
	if _, err := readySession(api).Locations(ctx); !errors.Is(err, ErrNoLocations) {
		t.Errorf("esperaba ErrNoLocations, obtuve %v", err)
	}
	if api.gotLocations.AuthorID != "me" || api.gotLocations.TitlePrefix != document.DeployTitlePrefix {
		t.Errorf("consulta de ubicaciones = %+v", api.gotLocations)
	}
}

func TestPublishCreatesNextToSiblingAndReportsLinkError(t *testing.T) {
	api := newAPI()
	api.linkErr = errors.New("sin permisos")
	s := readySession(api)

	res, err := s.Publish(ctx, s.DraftCreate(), &provider.DocRef{ID: "9"})
	if err != nil {
		t.Fatalf("un fallo del enlace no debe hacer fallar la publicación: %v", err)
	}
	if api.created.near != "9" || api.created.title != s.Title {
		t.Errorf("creada en %+v, esperaba junto a 9 con el título del issue", api.created)
	}
	if res.LinkErr == nil || api.linked.ID != "new" || api.linked.Title != s.Title {
		t.Error("el error del enlace a Jira debe llegar al llamador en LinkErr")
	}
}

func TestPublishUpdateUsesCurrentVersion(t *testing.T) {
	api := newAPI()
	api.current = &provider.Doc{DocRef: provider.DocRef{ID: "42", Version: 5}, Body: considerDoc("x")}
	s := readySession(api)
	s.Existing = &provider.DocRef{ID: "42"}
	d, err := s.DraftUpdate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.Publish(ctx, d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Updated || api.updated.id != "42" || api.updated.version != 5 {
		t.Errorf("actualización = %+v, esperaba página 42 sobre la versión 5", api.updated)
	}
}

func TestPublishNewWithoutSiblingFails(t *testing.T) {
	s := readySession(newAPI())
	if _, err := s.Publish(ctx, s.DraftCreate(), nil); err == nil {
		t.Error("un documento nuevo necesita ubicación")
	}
}

func TestSplitHashes(t *testing.T) {
	got := SplitHashes(" a1, ,b2,c3 ")
	if !reflect.DeepEqual(got, []string{"a1", "b2", "c3"}) {
		t.Errorf("SplitHashes = %v", got)
	}
	if SplitHashes("") != nil {
		t.Error("cadena vacía debe dar nil")
	}
}
