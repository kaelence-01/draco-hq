package config

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestProjectComplete(t *testing.T) {
	cases := []struct {
		name     string
		in, want ProjectConfig
		err      error
	}{
		{"sin rutas", ProjectConfig{BackendRepo: "pagos-api"}, ProjectConfig{BackendRepo: "pagos-api"}, ErrNoProjectPaths},
		{"repo desde la carpeta", ProjectConfig{BackendPath: "/src/pagos-api/"}, ProjectConfig{BackendPath: "/src/pagos-api/", BackendRepo: "pagos-api"}, nil},
		{"repo escrito se respeta", ProjectConfig{FrontendPath: "/src/web", FrontendRepo: "pagos-web"}, ProjectConfig{FrontendPath: "/src/web", FrontendRepo: "pagos-web"}, nil},
		{"repo sin ruta se borra", ProjectConfig{BackendPath: "/src/api", FrontendRepo: "pagos-web"}, ProjectConfig{BackendPath: "/src/api", BackendRepo: "api"}, nil},
	}
	for _, c := range cases {
		got := c.in
		if err := got.Complete(); !errors.Is(err, c.err) {
			t.Errorf("%s: error %v, se esperaba %v", c.name, err, c.err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %+v, se esperaba %+v", c.name, got, c.want)
		}
	}
}

func TestExpandHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for in, want := range map[string]string{
		"~":                 home,
		"~/code/pagos-api":  filepath.Join(home, "code", "pagos-api"),
		"/repos/pagos-api":  "/repos/pagos-api",
		"./docs":            "./docs",
		"~otro/pagos-api":   "~otro/pagos-api",
		"carpeta/~/interna": "carpeta/~/interna",
		"":                  "",
	} {
		if got := ExpandHome(in); got != want {
			t.Errorf("ExpandHome(%q) = %q, esperaba %q", in, got, want)
		}
	}
}

func TestSetBacklogPatternsKeepsTheRestAndDropsWhatIsEmpty(t *testing.T) {
	p := &ProjectConfig{Backlog: &BacklogConfig{Backend: &BacklogRepoConfig{Since: "6 months ago", ClassGlobs: []string{"viejo/**"}}}}
	p.SetBacklogPatterns("backend", []string{"app/**/Services/**/*.php"}, []string{"app/Http/{modulo}/**"})
	p.SetBacklogPatterns("frontend", []string{"src/app/**/*.service.ts"}, nil)
	b, f := p.BacklogFor("backend"), p.BacklogFor("frontend")
	if b.Since != "6 months ago" || !reflect.DeepEqual(b.ClassGlobs, []string{"app/**/Services/**/*.php"}) || len(b.Modules) != 1 || f == nil || len(f.ClassGlobs) != 1 {
		t.Fatalf("reemplaza los patrones y conserva el resto: %+v %+v", b, f)
	}

	p.SetBacklogPatterns("frontend", nil, nil)
	p.SetBacklogPatterns("backend", nil, nil)
	if p.Backlog == nil || p.Backlog.Frontend != nil || p.BacklogFor("backend").Since != "6 months ago" {
		t.Fatalf("un repo sin nada se quita; el que conserva since sigue: %+v", p.Backlog)
	}
	p.Backlog.Backend.Since = ""
	p.SetBacklogPatterns("backend", nil, nil)
	if p.Backlog != nil {
		t.Errorf("sin nada en ningún repo no queda backlog: %+v", p.Backlog)
	}
}

func TestResolveDeployChecklistPrefersProject(t *testing.T) {
	cfg := &Config{DeployChecklist: []string{"global-1", "global-2"}}
	proj := &ProjectConfig{DeployChecklist: []string{"proj-1"}}

	if got := cfg.ResolveDeployChecklist(proj); !reflect.DeepEqual(got, []string{"proj-1"}) {
		t.Errorf("la lista del proyecto debe ganar: %#v", got)
	}
}

func TestResolveDeployChecklistFallsBackToGlobal(t *testing.T) {
	cfg := &Config{DeployChecklist: []string{"global-1"}}

	for name, proj := range map[string]*ProjectConfig{
		"proyecto sin lista": {},
		"sin proyecto":       nil,
	} {
		got := cfg.ResolveDeployChecklist(proj)
		if !reflect.DeepEqual(got, []string{"global-1"}) {
			t.Errorf("%s: esperaba la global, obtuve %#v", name, got)
		}
	}
}

func TestResolveDeployChecklistEmptyWhenNothingConfigured(t *testing.T) {
	cfg := &Config{}
	if got := cfg.ResolveDeployChecklist(nil); len(got) != 0 {
		t.Errorf("esperaba vacío, obtuve %#v", got)
	}
}

func TestGetProjectResolution(t *testing.T) {
	echo := &ProjectConfig{BackendRepo: "echo-api"}
	otro := &ProjectConfig{BackendRepo: "otro-api"}
	cfg := &Config{
		DefaultProject: "echo",
		Projects:       map[string]*ProjectConfig{"echo": echo, "otro": otro},
	}

	proj, name, err := cfg.GetProject("otro")
	if err != nil || proj != otro || name != "otro" {
		t.Errorf("nombre explícito: %v %q %v", proj, name, err)
	}

	proj, name, err = cfg.GetProject("")
	if err != nil || proj != echo || name != "echo" {
		t.Errorf("default: %v %q %v", proj, name, err)
	}

	if _, _, err = cfg.GetProject("inexistente"); err == nil {
		t.Error("un proyecto inexistente debe dar error")
	}
}

func TestGetProjectWithoutAnyConfigured(t *testing.T) {
	cfg := &Config{}
	proj, name, err := cfg.GetProject("")
	if err != nil || proj != nil || name != "" {
		t.Errorf("sin proyectos esperaba (nil, \"\", nil): %v %q %v", proj, name, err)
	}
}

func TestQAJiraValidateListsEveryProblem(t *testing.T) {
	err := (&QAJiraConfig{Fields: QAJiraFields{DevStatus: "customfield_10000"}}).Validate()
	if err == nil {
		t.Fatal("esperaba error con project, statuses y dos campos vacíos")
	}
	for _, want := range []string{"qa_jira.project", "qa_jira.statuses", "fields.coding_errors", "fields.dev_returns"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("el error no menciona %s:\n%v", want, err)
		}
	}
	if strings.Contains(err.Error(), "fields.dev_status") {
		t.Errorf("dev_status es válido y no debería reportarse:\n%v", err)
	}
}

func TestQAJiraValidateAcceptsCompleteConfig(t *testing.T) {
	q := &QAJiraConfig{
		Project:  "OPS",
		Statuses: []string{"10001"},
		Fields: QAJiraFields{
			CodingErrors: "customfield_1",
			DevReturns:   "customfield_2",
			DevStatus:    "customfield_3",
		},
	}
	if err := q.Validate(); err != nil {
		t.Errorf("config completa rechazada: %v", err)
	}
}

func TestSprintNameFor(t *testing.T) {
	cases := []struct {
		tmpl, want string
	}{
		{"", "Pagos_Sprint 12"},
		{"Sprint {sprint} - {module}", "Sprint 12 - Pagos"},
	}
	for _, c := range cases {
		q := &QAJiraConfig{SprintName: c.tmpl}
		if got := q.SprintNameFor("Pagos", 12); got != c.want {
			t.Errorf("plantilla %q → %q, esperaba %q", c.tmpl, got, c.want)
		}
	}
}

func TestQAJiraValidateSprintNameNeedsSprint(t *testing.T) {
	q := &QAJiraConfig{
		Project:    "OPS",
		Statuses:   []string{"10001"},
		SprintName: "{module}_Sprint",
		Fields:     QAJiraFields{CodingErrors: "customfield_1", DevReturns: "customfield_2", DevStatus: "customfield_3"},
	}
	if err := q.Validate(); err == nil || !strings.Contains(err.Error(), "qa_jira.sprint_name") {
		t.Errorf("una plantilla sin {sprint} debe rechazarse, obtuve: %v", err)
	}
}

func TestQAJiraValidateNil(t *testing.T) {
	var q *QAJiraConfig
	if err := q.Validate(); err == nil || !strings.Contains(err.Error(), "qa_jira") {
		t.Errorf("sin sección qa_jira debe pedirla, obtuve: %v", err)
	}
}

func TestQAJiraEmpty(t *testing.T) {
	if !(&QAJiraConfig{}).Empty() {
		t.Error("sin valores, qa_jira está vacío")
	}
	for name, q := range map[string]QAJiraConfig{
		"proyecto": {Project: "OPS"},
		"estados":  {Statuses: []string{"10001"}},
		"campo":    {Fields: QAJiraFields{DevStatus: "customfield_10003"}},
		"marcador": {ObservationMarker: "Obs:"},
	} {
		if q.Empty() {
			t.Errorf("con %s, qa_jira no está vacío", name)
		}
	}
}

func TestSplitList(t *testing.T) {
	if got := SplitList(" 10001, En QA ,,Testing "); !reflect.DeepEqual(got, []string{"10001", "En QA", "Testing"}) {
		t.Errorf("SplitList = %q", got)
	}
	if got := SplitList(" , "); got != nil {
		t.Errorf("sin elementos no hay lista: %q", got)
	}
}

func TestQAJiraConventionDefaults(t *testing.T) {
	q := &QAJiraConfig{}
	if q.Marker() != "Novedad::" || len(q.Reviews()) != 3 {
		t.Errorf("defaults = %q, %v", q.Marker(), q.Reviews())
	}
	q = &QAJiraConfig{ObservationMarker: "Obs:", ReviewPrefixes: []string{"Validación"}}
	if q.Marker() != "Obs:" || q.Reviews()[0] != "Validación" {
		t.Errorf("valores propios = %q, %v", q.Marker(), q.Reviews())
	}
}
