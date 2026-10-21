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
	return huh.NewForm(pages...).WithShowHelp(false).WithTheme(huh.ThemeFunc(huh.ThemeBase16)).WithWidth(64)
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
	bundle, err := g.st.deps.OpenProfile(g.st.cfg, profile.Options{Name: g.st.profile, Scope: sides.SpaceKey})
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

func (g *generate) view(width, height int) string { return formView(g.form, g.err, width) }

// formView is a form with the error of its last submission above it.
func formView(f *huh.Form, err string, width int) string {
	body := f.View()
	if err != "" {
		body = lipgloss.JoinVertical(lipgloss.Left, lipgloss.NewStyle().Width(max(width-4, 10)).Render(errStyle.Render("✗ "+err)), "", body)
	}
	return lipgloss.NewStyle().Padding(1, 2).MaxWidth(width).Render(body)
}
