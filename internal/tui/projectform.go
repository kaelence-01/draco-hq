package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/backlog"
	"github.com/kaelence-01/draco-hq/internal/config"
)

var (
	nextFieldKey = key.NewBinding(key.WithKeys("enter", "tab"), key.WithHelp("enter/tab", "siguiente"))
	prevFieldKey = key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "anterior"))
)

// projectForm creates a project or edits one. Editing keeps what the form does
// not show (checklist, the rest of the backlog), as draco project add does.
type projectForm struct {
	parent  *projects
	name    string
	edit    bool
	v       config.ProjectConfig // only the fields of the form
	backlog [2]backlogPatterns   // backend, frontend
	form    *huh.Form
}

// backlogPatterns are a repo's class globs and modules, as comma-separated text.
type backlogPatterns struct{ classes, modules string }

func newProjectForm(parent *projects, name string) *projectForm {
	f := &projectForm{parent: parent, name: name, edit: name != ""}
	var fields []huh.Field
	if !f.edit {
		fields = append(fields, huh.NewInput().Title("Nombre").Placeholder("pagos").Value(&f.name).Validate(func(s string) error {
			if err := config.ValidName("proyecto", s); err != nil {
				return err
			}
			if _, ok := parent.st.cfg.Projects[s]; ok {
				return fmt.Errorf("el proyecto '%s' ya existe: edítalo con e", s)
			}
			return nil
		}))
	} else {
		old := parent.st.cfg.Projects[name]
		f.v = config.ProjectConfig{
			BackendPath: old.BackendPath, BackendRepo: old.BackendRepo, FrontendPath: old.FrontendPath, FrontendRepo: old.FrontendRepo,
			VCSHost: old.VCSHost, VCSOrg: old.VCSOrg, ConfluenceSpaceKey: old.ConfluenceSpaceKey,
		}
		for i, repo := range []string{"backend", "frontend"} {
			if b := old.BacklogFor(repo); b != nil {
				f.backlog[i] = backlogPatterns{classes: strings.Join(b.ClassGlobs, ", "), modules: strings.Join(b.Modules, ", ")}
			}
		}
	}
	fields = append(fields,
		huh.NewInput().Title("Ruta local del repositorio backend").Value(&f.v.BackendPath),
		huh.NewInput().Title("Nombre del repositorio backend en el VCS").Description("Vacío: el nombre de la carpeta").Value(&f.v.BackendRepo),
		huh.NewInput().Title("Ruta local del repositorio frontend").Value(&f.v.FrontendPath),
		huh.NewInput().Title("Nombre del repositorio frontend en el VCS").Description("Vacío: el nombre de la carpeta").Value(&f.v.FrontendRepo),
		huh.NewInput().Title("Host del VCS").Placeholder("https://github.com").Value(&f.v.VCSHost),
		huh.NewInput().Title("Organización o workspace del VCS").Placeholder("mi-org").Value(&f.v.VCSOrg),
		huh.NewInput().Title("Space de Confluence del proyecto").Description("Vacío: el de Conexión o el del perfil").Value(&f.v.ConfluenceSpaceKey),
	)
	f.form = newPagedForm(huh.NewGroup(fields...),
		f.backlogPage("backend", &f.backlog[0], "app/**/Services/**/*.php", "app/Http/{modulo}/**", &f.v.BackendPath),
		f.backlogPage("frontend", &f.backlog[1], "src/app/**/*.service.ts", "src/app/{modulo}/**", &f.v.FrontendPath))
	return f
}

// backlogPage is the optional page of one repo's backlog patterns, hidden
// while the repo has no path.
func (f *projectForm) backlogPage(repo string, p *backlogPatterns, classes, modules string, repoPath *string) *huh.Group {
	return huh.NewGroup(
		huh.NewInput().Title("Backlog técnico del "+repo+": clases que deben tener tests").
			Description("Opcional, separadas por comas. Vacío: sin ese detector").Placeholder(classes).Value(&p.classes).
			Validate(func(s string) error { return backlog.CheckPatterns(config.SplitList(s), nil) }),
		huh.NewInput().Title("Backlog técnico del "+repo+": módulos").
			Description("Opcional, separados por comas; {modulo} es la carpeta que nombra el módulo").Placeholder(modules).Value(&p.modules).
			Validate(func(s string) error { return backlog.CheckPatterns(nil, config.SplitList(s)) }),
	).WithHideFunc(func() bool { return strings.TrimSpace(*repoPath) == "" })
}

func (f *projectForm) title() string {
	if f.edit {
		return "Editar " + f.name
	}
	return "Nuevo proyecto"
}

func (f *projectForm) keys() []key.Binding { return []key.Binding{nextFieldKey, prevFieldKey} }

// captures leaves only esc to the app: it closes the form without saving.
func (f *projectForm) captures(k tea.KeyPressMsg) bool { return !key.Matches(k, backKey) }

func (f *projectForm) init() tea.Cmd { return f.form.Init() }

func (f *projectForm) update(msg tea.Msg) (screen, tea.Cmd) {
	m, cmd := f.form.Update(msg)
	f.form = m.(*huh.Form)
	if f.form.State != huh.StateCompleted {
		return f, cmd
	}
	f.save()
	return f, back
}

func (f *projectForm) save() {
	p := f.parent
	for _, side := range []struct{ path, repo *string }{{&f.v.BackendPath, &f.v.BackendRepo}, {&f.v.FrontendPath, &f.v.FrontendRepo}} {
		switch {
		case *side.path == "":
			*side.repo = ""
		case *side.repo == "":
			*side.repo = filepath.Base(filepath.Clean(*side.path))
		}
	}
	if f.v.BackendPath == "" && f.v.FrontendPath == "" {
		p.note, p.failed = "Sin rutas de backend ni de frontend: el proyecto no se guardó", true
		return
	}
	err := p.st.save(func(c *config.Config) error {
		pc := &config.ProjectConfig{}
		if old, ok := c.Projects[f.name]; ok {
			*pc = *old
		}
		pc.BackendPath, pc.BackendRepo, pc.FrontendPath, pc.FrontendRepo = f.v.BackendPath, f.v.BackendRepo, f.v.FrontendPath, f.v.FrontendRepo
		pc.VCSHost, pc.VCSOrg, pc.ConfluenceSpaceKey = f.v.VCSHost, f.v.VCSOrg, f.v.ConfluenceSpaceKey
		for i, side := range []struct{ repo, path string }{{"backend", pc.BackendPath}, {"frontend", pc.FrontendPath}} {
			if side.path != "" {
				pc.SetBacklogPatterns(side.repo, config.SplitList(f.backlog[i].classes), config.SplitList(f.backlog[i].modules))
			}
		}
		if err := c.SetProject(f.name, pc); err != nil {
			return err
		}
		if c.DefaultProject == "" {
			c.DefaultProject = f.name
		}
		return nil
	})
	p.done("Proyecto guardado: "+f.name, err)
}

func (f *projectForm) view(width, height int) string {
	return lipgloss.NewStyle().Padding(1, 2).MaxWidth(width).Render(f.form.View())
}
