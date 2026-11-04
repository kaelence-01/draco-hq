package config

import (
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
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

func multiProjectQAJira() *QAJiraConfig {
	return &QAJiraConfig{
		Project:       "OPS",
		Statuses:      []string{"Testing", "Aprobado"},
		TestingStatus: "Testing",
		Projects: map[string]QAJiraProject{
			"PAG": {},
			"FAC": {Statuses: []string{"En pruebas", "Listo"}, TestingStatus: "En pruebas", SprintName: "FAC Sprint {sprint}"},
		},
		Fields: QAJiraFields{CodingErrors: "customfield_1", DevReturns: "customfield_2", DevStatus: "customfield_3"},
	}
}

func TestQAJiraProjectKeysDefaultFirst(t *testing.T) {
	if got := multiProjectQAJira().ProjectKeys(); !reflect.DeepEqual(got, []string{"OPS", "FAC", "PAG"}) {
		t.Errorf("ProjectKeys = %q, esperaba el por defecto primero y los demás ordenados", got)
	}
	q := multiProjectQAJira()
	q.Projects["ops"] = QAJiraProject{SprintName: "{sprint}"}
	if got := q.ProjectKeys(); !reflect.DeepEqual(got, []string{"OPS", "FAC", "PAG"}) {
		t.Errorf("el por defecto no se repite aunque cambie la caja: %q", got)
	}
	var none *QAJiraConfig
	if none.ProjectKeys() != nil {
		t.Error("sin qa_jira no hay proyectos")
	}
}

func TestQAJiraForProject(t *testing.T) {
	q := multiProjectQAJira()
	def, err := q.ForProject("")
	if err != nil || def.Project != "OPS" || !reflect.DeepEqual(def.Statuses, []string{"Testing", "Aprobado"}) || def.Projects != nil {
		t.Fatalf("sin proyecto se usa el por defecto: %+v %v", def, err)
	}
	pag, err := q.ForProject("pag")
	if err != nil || pag.Project != "PAG" || pag.TestingStatus != "Testing" || pag.SprintNameFor("Pagos", 3) != "Pagos_Sprint 3" {
		t.Errorf("un proyecto sin valores propios toma los del por defecto: %+v %v", pag, err)
	}
	fac, err := q.ForProject("FAC")
	if err != nil || !reflect.DeepEqual(fac.Statuses, []string{"En pruebas", "Listo"}) || fac.TestingStatus != "En pruebas" || fac.SprintNameFor("x", 3) != "FAC Sprint 3" {
		t.Errorf("un proyecto con valores propios los usa: %+v %v", fac, err)
	}
	if fac.Fields != q.Fields {
		t.Errorf("los campos son de la instancia, iguales para todos: %+v", fac.Fields)
	}
	if !reflect.DeepEqual(q.Statuses, []string{"Testing", "Aprobado"}) || len(q.Projects) != 2 {
		t.Errorf("ForProject no modifica la config: %+v", q)
	}
	if _, err := q.ForProject("XYZ"); err == nil || !strings.Contains(err.Error(), "OPS, FAC, PAG") {
		t.Errorf("un proyecto desconocido lista los configurados: %v", err)
	}
	q.Fields.DevStatus = ""
	if _, err := q.ForProject("FAC"); err == nil || !strings.Contains(err.Error(), "fields.dev_status") {
		t.Errorf("ForProject valida antes: %v", err)
	}
}

func TestQAJiraLookupDoesNotValidate(t *testing.T) {
	q := &QAJiraConfig{Project: "OPS", Projects: map[string]QAJiraProject{"FAC": {Statuses: []string{"7"}}}}
	if r, ok := q.Lookup("fac"); !ok || r.Project != "FAC" || !reflect.DeepEqual(r.Statuses, []string{"7"}) {
		t.Errorf("Lookup lee un qa_jira incompleto: %+v %v", r, ok)
	}
	if _, ok := q.Lookup("XYZ"); ok {
		t.Error("un proyecto que no está no se encuentra")
	}
}

func TestQAJiraSetProject(t *testing.T) {
	q := &QAJiraConfig{}
	q.SetProject("OPS", []string{"2", "3"}, "Testing")
	if q.Project != "OPS" || !reflect.DeepEqual(q.Statuses, []string{"2", "3"}) || q.TestingStatus != "" {
		t.Errorf("sin proyecto, el primero es el por defecto y Testing queda implícito: %+v", q)
	}

	q.Projects = map[string]QAJiraProject{"PAG": {SprintName: "PAG {sprint}"}, "ops": {Statuses: []string{"9"}}}
	q.SetProject("ops", []string{"4"}, "En QA")
	if q.Project != "OPS" || !reflect.DeepEqual(q.Statuses, []string{"4"}) || q.TestingStatus != "En QA" {
		t.Errorf("el por defecto se escribe arriba: %+v", q)
	}
	if _, ok := q.Projects["ops"]; ok {
		t.Errorf("lo que pisaba al por defecto se quita: %+v", q.Projects)
	}

	q.SetProject("PAG", []string{"4"}, "En QA")
	if p := q.Projects["PAG"]; p.Statuses != nil || p.TestingStatus != "" || p.SprintName != "PAG {sprint}" {
		t.Errorf("otro proyecto igual al por defecto queda sin valores propios y conserva su sprint: %+v", p)
	}
	q.SetProject("FAC", []string{"5"}, "Testing")
	if p := q.Projects["FAC"]; !reflect.DeepEqual(p.Statuses, []string{"5"}) || p.TestingStatus != "Testing" {
		t.Errorf("otro proyecto distinto guarda lo suyo: %+v", p)
	}
	if r, _ := q.Lookup("FAC"); r.Testing() != "Testing" || !reflect.DeepEqual(r.Statuses, []string{"5"}) {
		t.Errorf("y se lee como se eligió: %+v", r)
	}
}

func TestQAJiraProjectsFromYAML(t *testing.T) {
	raw := `
qa_jira:
  project: OPS
  statuses: [Testing]
  projects:
    PAG:
    FAC: {statuses: [En pruebas]}
`
	var cfg Config
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}
	if got := cfg.QAJira.ProjectKeys(); !reflect.DeepEqual(got, []string{"OPS", "FAC", "PAG"}) {
		t.Errorf("un proyecto sin valores (PAG:) también cuenta: %q", got)
	}
	out, err := yaml.Marshal(&QAJiraConfig{Project: "OPS", Projects: map[string]QAJiraProject{"PAG": {}}})
	if err != nil || !strings.Contains(string(out), "PAG: {}") {
		t.Errorf("un proyecto sin valores se guarda como {}:\n%s", out)
	}
}

func TestQAJiraValidateProjects(t *testing.T) {
	q := multiProjectQAJira()
	q.Projects["MAL CLAVE"] = QAJiraProject{}
	q.Projects["FAC"] = QAJiraProject{SprintName: "FAC Sprint"}
	err := q.Validate()
	for _, want := range []string{`clave de proyecto inválida "MAL CLAVE"`, "qa_jira.projects.FAC.sprint_name"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("el error no menciona %s:\n%v", want, err)
		}
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
		"otro":     {Projects: map[string]QAJiraProject{"PAG": {}}},
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
