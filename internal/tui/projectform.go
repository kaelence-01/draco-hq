package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/kaelence-01/draco-hq/internal/backlog"
	"github.com/kaelence-01/draco-hq/internal/config"
)

var (
	nextFieldKey = key.NewBinding(key.WithKeys("enter", "tab"), key.WithHelp("enter/tab", "siguiente"))
	prevFieldKey = key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "anterior"))
)

// projectForm creates a project or edits one. Editing keeps what the form does
// not show (checklist, the rest of the backlog), as draco project add does. A
// new project left unfinished (esc, ctrl+c, or saved without its paths) is
// kept as a draft, which the form can resume.
type projectForm struct {
	parent  *projects
	name    string
	edit    bool
	draft   string // the draft being resumed
	saved   bool
	v       config.ProjectConfig // only the fields of the form
	backlog [2]backlogPatterns   // backend, frontend
	form    *huh.Form
	pages   pager
}

// backlogPatterns are a repo's class globs and modules, as comma-separated text.
type backlogPatterns struct{ classes, modules string }

func newProjectForm(parent *projects, name string) *projectForm {
	f := &projectForm{parent: parent, name: name, edit: name != ""}
	if f.edit {
		f.load(parent.st.cfg.Projects[name])
	}
	f.build()
	return f
}

// newDraftForm resumes a draft with what it had.
func newDraftForm(parent *projects, draft string) *projectForm {
	f := &projectForm{parent: parent, name: draft, draft: draft}
	f.load(parent.st.cfg.ProjectDrafts[draft])
	f.build()
	return f
}

// load fills the form with the fields it shows of old.
func (f *projectForm) load(old *config.ProjectConfig) {
	f.v = config.ProjectConfig{
		BackendPath: old.BackendPath, BackendRepo: old.BackendRepo, FrontendPath: old.FrontendPath, FrontendRepo: old.FrontendRepo,
		VCSHost: old.VCSHost, VCSOrg: old.VCSOrg, ConfluenceSpaceKey: old.ConfluenceSpaceKey, Profile: old.Profile,
	}
	for i, repo := range []string{"backend", "frontend"} {
		if b := old.BacklogFor(repo); b != nil {
			f.backlog[i] = backlogPatterns{classes: strings.Join(b.ClassGlobs, ", "), modules: strings.Join(b.Modules, ", ")}
		}
	}
}

func (f *projectForm) build() {
	parent := f.parent
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
	pages := []page{newPage("Repositorios y VCS", fields...)}
	if names := parent.st.cfg.ProfileNames(); len(names) > 1 {
		options := []huh.Option[string]{huh.NewOption("El perfil por defecto", "")}
		for _, n := range names {
			options = append(options, huh.NewOption(n, n))
		}
		pages = append(pages, newPage("Perfil", huh.NewSelect[string]().Title("Perfil del proyecto").
			Description("Destino y plantilla al generar sus documentos").Options(options...).Value(&f.v.Profile)))
	}
	pages = append(pages,
		f.backlogPage("backend", &f.backlog[0], "app/**/Services/**/*.php", "app/Http/{modulo}/**", &f.v.BackendPath),
		f.backlogPage("frontend", &f.backlog[1], "src/app/**/*.service.ts", "src/app/{modulo}/**", &f.v.FrontendPath))
	f.form, f.pages = newPagedForm(pages...)
}

// backlogPage is the optional page of one repo's backlog patterns, hidden
// while the repo has no path.
func (f *projectForm) backlogPage(repo string, p *backlogPatterns, classes, modules string, repoPath *string) page {
	return newPage("Backlog del "+repo,
		huh.NewInput().Title("Backlog técnico del "+repo+": clases que deben tener tests").
			Description("Opcional, separadas por comas. Vacío: sin ese detector").Placeholder(classes).Value(&p.classes).
			Validate(func(s string) error { return backlog.CheckPatterns(config.SplitList(s), nil) }),
		huh.NewInput().Title("Backlog técnico del "+repo+": módulos").
			Description("Opcional, separados por comas; {modulo} es la carpeta que nombra el módulo").Placeholder(modules).Value(&p.modules).
			Validate(func(s string) error { return backlog.CheckPatterns(nil, config.SplitList(s)) }),
	).hiddenWhen(func() bool { return strings.TrimSpace(*repoPath) == "" })
}

func (f *projectForm) title() string {
	switch {
	case f.edit:
		return "Editar " + f.name
	case f.draft != "":
		return "Borrador " + f.draft
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
	if err := f.v.Complete(); err != nil {
		p.note, p.failed = "El proyecto no se guardó: "+err.Error(), true
		return
	}
	err := p.st.save(func(c *config.Config) error {
		pc := &config.ProjectConfig{}
		if old, ok := c.Projects[f.name]; ok {
			*pc = *old
		}
		f.apply(pc)
		if err := c.SetProject(f.name, pc); err != nil {
			return err
		}
		if _, ok := c.ProjectDrafts[f.draft]; ok {
			_ = c.RemoveProjectDraft(f.draft)
		}
		if c.DefaultProject == "" {
			c.DefaultProject = f.name
		}
		return nil
	})
	f.saved = err == nil
	p.done("Proyecto guardado: "+f.name, err)
}

// apply writes the form's fields over pc.
func (f *projectForm) apply(pc *config.ProjectConfig) {
	pc.BackendPath, pc.BackendRepo, pc.FrontendPath, pc.FrontendRepo = f.v.BackendPath, f.v.BackendRepo, f.v.FrontendPath, f.v.FrontendRepo
	pc.VCSHost, pc.VCSOrg, pc.ConfluenceSpaceKey, pc.Profile = f.v.VCSHost, f.v.VCSOrg, f.v.ConfluenceSpaceKey, f.v.Profile
	for i, side := range []struct{ repo, path string }{{"backend", pc.BackendPath}, {"frontend", pc.FrontendPath}} {
		if side.path != "" {
			pc.SetBacklogPatterns(side.repo, config.SplitList(f.backlog[i].classes), config.SplitList(f.backlog[i].modules))
		}
	}
}

// close keeps a new project left unfinished as a draft, so leaving by
// mistake loses nothing. The app closes every screen on ctrl+c too.
func (f *projectForm) close() {
	if f.edit || f.saved || !f.typed() {
		return
	}
	p := f.parent
	var name string
	err := p.st.save(func(c *config.Config) error {
		name = c.DraftName(f.name, f.draft)
		if f.draft != "" && f.draft != name {
			_ = c.RemoveProjectDraft(f.draft)
		}
		pc := &config.ProjectConfig{}
		f.apply(pc)
		return c.SetProjectDraft(name, pc)
	})
	note := "Quedó como borrador «" + name + "»: enter lo retoma; no se puede usar hasta completarlo"
	if p.failed && p.note != "" {
		note = p.note + ". " + note
	}
	p.done(note, err)
}

// typed reports whether the form has anything written.
func (f *projectForm) typed() bool {
	v := f.v
	for _, s := range []string{f.name, v.BackendPath, v.BackendRepo, v.FrontendPath, v.FrontendRepo, v.VCSHost, v.VCSOrg, v.ConfluenceSpaceKey,
		f.backlog[0].classes, f.backlog[0].modules, f.backlog[1].classes, f.backlog[1].modules} {
		if strings.TrimSpace(s) != "" {
			return true
		}
	}
	return false
}

func (f *projectForm) view(width, height int) string {
	return f.pages.formView(f.form, "", width, height)
}
