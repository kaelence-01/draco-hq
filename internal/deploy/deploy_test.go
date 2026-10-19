package deploy

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/atlassian"
	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/document"
)

// fakeAPI records what the workflow asks Atlassian and answers from its fields.
type fakeAPI struct {
	issue        *atlassian.Issue
	byTitle      *atlassian.Page
	byIssue      *atlassian.Page
	byIssueErr   error
	current      *atlassian.PageADF
	currentErr   error
	pages        map[string]*atlassian.Page
	me           *atlassian.JiraUser
	meErr        error
	lastDocs     []atlassian.Page
	lastFellBack bool
	linkErr      error

	gotAccountID string
	created      struct{ spaceID, parentID, title string }
	updated      struct {
		id      string
		version int
	}
	linked bool
}

func (f *fakeAPI) GetIssue(string) (*atlassian.Issue, error)               { return f.issue, nil }
func (f *fakeAPI) FindPageByTitle(string, string) (*atlassian.Page, error) { return f.byTitle, nil }
func (f *fakeAPI) GetPageADF(string) (*atlassian.PageADF, error)           { return f.current, f.currentErr }
func (f *fakeAPI) GetPage(id string) (*atlassian.Page, error)              { return f.pages[id], nil }
func (f *fakeAPI) Whoami() (*atlassian.JiraUser, error)                    { return f.me, f.meErr }
func (f *fakeAPI) FindDeployDocByIssue(string, string) (*atlassian.Page, error) {
	return f.byIssue, f.byIssueErr
}

func (f *fakeAPI) FindLastDeployDoc(_ string, accountID string) ([]atlassian.Page, bool, error) {
	f.gotAccountID = accountID
	return f.lastDocs, f.lastFellBack, nil
}

func (f *fakeAPI) CreatePage(spaceID, parentID, title string, _ map[string]any) (*atlassian.Page, error) {
	f.created.spaceID, f.created.parentID, f.created.title = spaceID, parentID, title
	return &atlassian.Page{ID: "new", WebURL: "https://example.test/new"}, nil
}

func (f *fakeAPI) UpdatePage(id, _ string, version int, _ map[string]any) (*atlassian.Page, error) {
	f.updated.id, f.updated.version = id, version
	return &atlassian.Page{ID: id, WebURL: "https://example.test/" + id}, nil
}

func (f *fakeAPI) CreateJiraRemoteLink(string, string, string, string) error {
	f.linked = true
	return f.linkErr
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
	s := NewSession(api, filesOK, target())
	if err := s.FetchIssue(); err != nil {
		panic(err)
	}
	if err := s.ReadCommits(); err != nil {
		panic(err)
	}
	return s
}

func newAPI() *fakeAPI {
	return &fakeAPI{issue: &atlassian.Issue{Key: "OPS-7", Summary: "Filtro de pagos", URL: "https://example.test/browse/OPS-7"}}
}

// --- Resolve ---

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

// --- Steps ---

func TestFindExistingFallsBackToIssueSearch(t *testing.T) {
	api := newAPI()
	api.byIssue = &atlassian.Page{ID: "42"}
	s := readySession(api)
	if err := s.FindExisting(); err != nil {
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
	if err := s.FindExisting(); err != nil {
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
	s := NewSession(newAPI(), files, tg)
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
	if err := NewSession(newAPI(), files, target()).ReadCommits(); err == nil {
		t.Error("sin ningún commit legible debe fallar")
	}
}

// --- Drafts ---

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
	api.current = &atlassian.PageADF{Page: atlassian.Page{ID: "42", Version: 5}, ADF: considerDoc("Reiniciar colas a mano")}
	s := readySession(api)
	s.Existing = &atlassian.Page{ID: "42"}

	d, err := s.DraftUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if !d.Preserved {
		t.Error("esperaba Preserved = true")
	}
	got := document.ExtractSection(d.ADF, document.ConsiderHeading)
	want := document.ExtractSection(api.current.ADF, document.ConsiderHeading)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("la sección no se preservó:\n obtuve %v\n esperaba %v", got, want)
	}
}

func TestDraftUpdateRefusesWhenCurrentPageUnreadable(t *testing.T) {
	api := newAPI()
	api.currentErr = errors.New("timeout")
	s := readySession(api)
	s.Existing = &atlassian.Page{ID: "42"}
	if d, err := s.DraftUpdate(); err == nil || d != nil {
		t.Error("sin poder leer la página actual no debe haber borrador: se sobrescribiría a ciegas")
	}
}

func TestDraftUpdateWarnsOnNonADFPage(t *testing.T) {
	api := newAPI()
	api.current = &atlassian.PageADF{Page: atlassian.Page{ID: "42"}}
	s := readySession(api)
	s.Existing = &atlassian.Page{ID: "42"}
	d, err := s.DraftUpdate()
	if err != nil {
		t.Fatal(err)
	}
	if d.Preserved || len(d.Warnings) != 1 {
		t.Errorf("esperaba Preserved=false y un aviso, obtuve %v / %v", d.Preserved, d.Warnings)
	}
}

// --- Locations and publish ---

func TestLocationsFallsBackToTeamWhenUserUnknown(t *testing.T) {
	api := newAPI()
	api.meErr = errors.New("403")
	api.lastDocs = []atlassian.Page{{ID: "1"}}
	loc, err := readySession(api).Locations()
	if err != nil {
		t.Fatal(err)
	}
	if !loc.OwnerUnknown || !loc.TeamWide || api.gotAccountID != "" {
		t.Errorf("esperaba lista de todo el equipo sin accountID, obtuve %+v (account %q)", loc, api.gotAccountID)
	}
}

func TestLocationsWithoutPreviousDocs(t *testing.T) {
	api := newAPI()
	api.me = &atlassian.JiraUser{AccountID: "me"}
	if _, err := readySession(api).Locations(); !errors.Is(err, ErrNoLocations) {
		t.Errorf("esperaba ErrNoLocations, obtuve %v", err)
	}
}

func TestPublishCreatesNextToSiblingAndReportsLinkError(t *testing.T) {
	api := newAPI()
	api.pages = map[string]*atlassian.Page{"9": {ID: "9", SpaceID: "S1", ParentID: "P1"}}
	api.linkErr = errors.New("sin permisos")
	s := readySession(api)

	res, err := s.Publish(s.DraftCreate(), &atlassian.Page{ID: "9"})
	if err != nil {
		t.Fatalf("un fallo del enlace no debe hacer fallar la publicación: %v", err)
	}
	if api.created.spaceID != "S1" || api.created.parentID != "P1" || api.created.title != s.Title {
		t.Errorf("creada en %+v, esperaba S1/P1 con el título del issue", api.created)
	}
	if res.LinkErr == nil || !api.linked {
		t.Error("el error del enlace a Jira debe llegar al llamador en LinkErr")
	}
}

func TestPublishUpdateUsesCurrentVersion(t *testing.T) {
	api := newAPI()
	api.current = &atlassian.PageADF{Page: atlassian.Page{ID: "42", Version: 5}, ADF: considerDoc("x")}
	s := readySession(api)
	s.Existing = &atlassian.Page{ID: "42"}
	d, err := s.DraftUpdate()
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.Publish(d, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Updated || api.updated.id != "42" || api.updated.version != 5 {
		t.Errorf("actualización = %+v, esperaba página 42 sobre la versión 5", api.updated)
	}
}

func TestPublishNewWithoutSiblingFails(t *testing.T) {
	s := readySession(newAPI())
	if _, err := s.Publish(s.DraftCreate(), nil); err == nil {
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
