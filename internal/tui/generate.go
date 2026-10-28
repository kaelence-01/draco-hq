package tui

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/deploy"
	"github.com/kaelence-01/draco-hq/internal/profile"
)

// generate is the first step of the deploy document: the task and the project.
// The profile is opened here, so a missing credential or space shows up before
// the commits are chosen.
type generate struct {
	st      *state
	issue   string
	project string
	form    *huh.Form
	err     string
}

func newGenerate(st *state) *generate {
	g := &generate{st: st, project: st.project}
	g.form = g.build()
	return g
}

func (g *generate) build() *huh.Form {
	fields := []huh.Field{taskInput(&g.issue)}
	if g.st.cfg != nil && len(g.st.cfg.Projects) > 0 {
		fields = append(fields, huh.NewSelect[string]().Title("Proyecto").Options(huh.NewOptions(g.st.cfg.ProjectNames()...)...).Value(&g.project))
	}
	return newForm(fields...)
}

// taskInput is the required task key of a form.
func taskInput(v *string) *huh.Input {
	return huh.NewInput().Title("Tarea").Placeholder("OPS-7").Value(v).Validate(func(s string) error {
		if strings.TrimSpace(s) == "" {
			return errors.New("falta la clave de la tarea")
		}
		return nil
	})
}

func newForm(fields ...huh.Field) *huh.Form { return newPagedForm(huh.NewGroup(fields...)) }

// newPagedForm is a form of several pages, for what does not fit in 24 lines.
func newPagedForm(pages ...*huh.Group) *huh.Form {
	return huh.NewForm(pages...).WithShowHelp(false).WithTheme(formTheme).WithWidth(64)
}

func (g *generate) title() string       { return "Generar documento" }
func (g *generate) keys() []key.Binding { return []key.Binding{nextFieldKey, prevFieldKey} }

func (g *generate) captures(k tea.KeyPressMsg) bool { return !key.Matches(k, backKey) }

// init also runs when the user comes back from the commits: the form starts
// again with the same values.
func (g *generate) init() tea.Cmd {
	if g.form.State != huh.StateNormal {
		g.form = g.build()
	}
	return g.form.Init()
}

func (g *generate) update(msg tea.Msg) (screen, tea.Cmd) {
	m, cmd := g.form.Update(msg)
	g.form = m.(*huh.Form)
	if g.form.State != huh.StateCompleted {
		return g, cmd
	}
	g.issue = strings.TrimSpace(g.issue)
	next, err := g.next()
	if err != nil {
		g.err = err.Error()
		g.form = g.build()
		return g, g.form.Init()
	}
	g.err = ""
	return g, open(next)
}

func (g *generate) next() (screen, error) {
	if g.st.cfg == nil {
		return nil, errors.New(g.st.notice)
	}
	sides, err := deploy.ProjectSides(g.st.cfg, g.project)
	if err != nil {
		return nil, err
	}
	bundle, err := g.st.deps.OpenProfile(g.st.cfg, profile.Options{Name: g.st.cfg.ProfileFor("", sides.Project), ProjectSpace: sides.SpaceKey})
	if err != nil {
		return nil, explainProfile(err)
	}
	return newCommits(g.st, bundle, sides, g.issue), nil
}

// explainProfile adds what to do to the errors of opening a profile.
func explainProfile(err error) error {
	switch {
	case errors.Is(err, profile.ErrNoSpace):
		return fmt.Errorf("%w: ponlo en el proyecto (Proyectos, e) o en docs.space del perfil", err)
	case errors.Is(err, profile.ErrNoCredentials):
		return fmt.Errorf("%w (faltan atlassian_email, atlassian_token o base_url)", err)
	}
	return err
}

func (g *generate) view(width, height int) string {
	if width < panelMinWidth {
		return formView(g.form, g.err, width)
	}
	formWidth, nextWidth := columns(width-2, 2, 5, 4)
	g.form.WithWidth(min(64, formWidth-4))
	form := lipgloss.JoinVertical(lipgloss.Left, faintStyle.Render("TAREA Y PROYECTO"), "", formBody(g.form, g.err, formWidth-4))
	body := lipgloss.JoinHorizontal(lipgloss.Top, boxStyle.Width(formWidth).Render(form), "  ", boxStyle.Width(nextWidth).Render(g.ahead(nextWidth-4)))
	return lipgloss.NewStyle().Padding(1, 1, 0).Render(body)
}

// ahead is what follows the form: the steps, the repos of the project and
// where the document goes.
func (g *generate) ahead(width int) string {
	lines := []string{faintStyle.Render("LO QUE SIGUE"), stepper(deploySteps, 0), ""}
	cfg := g.st.cfg
	if cfg == nil {
		return strings.Join(lines, "\n")
	}
	sides, err := deploy.ProjectSides(cfg, g.project)
	if err != nil {
		return strings.Join(append(lines, faintStyle.Render(err.Error())), "\n")
	}
	back, front := sides.Deployable()
	lines = append(lines, repoLine("Backend", sides.Backend, back, width), repoLine("Frontend", sides.Frontend, front, width))
	tracker := "el tracker"
	if p, _, err := cfg.Profile(cfg.ProfileFor("", sides.Project)); err == nil {
		tracker = p.Tracker.Provider
		dest := p.Docs.Provider
		if scope := profile.Scope(p, profile.Options{ProjectSpace: sides.SpaceKey}); scope != "" {
			dest += " · " + scope
		}
		lines = append(lines, kv("Destino", dest, width))
	}
	lines = append(lines, "", lipgloss.NewStyle().Width(width).Render(faintStyle.Render("La tarea se lee de "+tracker+" y los commits, del clon local de cada repo del proyecto.")))
	return strings.Join(lines, "\n")
}

// repoLine is a repo of the project with its clone, or a dash when the project
// has no such side.
func repoLine(label string, r deploy.Repo, deployable bool, width int) string {
	if !deployable {
		return kv(label, "", width)
	}
	where := r.WorkDir
	if where == "" {
		where = "directorio actual"
	}
	name := r.Name
	if name == "" {
		name = "sin nombre"
	}
	return kv(label, name+faintStyle.Render("  "+where), width)
}

// formView is a form with the error of its last submission above it.
func formView(f *huh.Form, err string, width int) string {
	return lipgloss.NewStyle().Padding(1, 2).MaxWidth(width).Render(formBody(f, err, width))
}

func formBody(f *huh.Form, err string, width int) string {
	body := f.View()
	if err != "" {
		body = lipgloss.JoinVertical(lipgloss.Left, lipgloss.NewStyle().Width(max(width-4, 10)).Render(errStyle.Render("✗ "+err)), "", body)
	}
	return body
}
