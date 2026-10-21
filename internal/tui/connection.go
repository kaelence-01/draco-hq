package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/profile"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

// connection is the Atlassian account and, without profiles, the default
// space, as draco init asks them. The form starts from the file, never from
// the environment, so a token from ATLASSIAN_TOKEN is not saved.
type connection struct {
	parent                   *settings
	email, token, url, space string
	hasToken                 bool
	withSpace                bool     // with profiles, each one has its space
	env                      []string // variables that win over the file
	form                     *huh.Form
	err                      string
}

func newConnection(parent *settings) *connection {
	c := &connection{parent: parent}
	file, err := parent.st.deps.LoadFile()
	if err != nil {
		c.err, file = err.Error(), &config.Config{}
	}
	c.email, c.url, c.hasToken = file.AtlassianEmail, file.BaseURL, file.AtlassianToken != ""
	c.space, c.withSpace = file.ConfluenceSpaceKey, len(file.Profiles) == 0
	eff := parent.st.cfg
	for _, v := range []struct{ name, eff, file string }{
		{"ATLASSIAN_EMAIL", eff.AtlassianEmail, file.AtlassianEmail},
		{"ATLASSIAN_TOKEN", eff.AtlassianToken, file.AtlassianToken},
		{"ATLASSIAN_BASE_URL", eff.BaseURL, file.BaseURL},
		{"CONFLUENCE_SPACE_KEY", eff.ConfluenceSpaceKey, file.ConfluenceSpaceKey},
	} {
		if v.eff != "" && v.eff != v.file {
			c.env = append(c.env, v.name)
		}
	}
	c.form = c.build()
	return c
}

func (c *connection) build() *huh.Form {
	tokenHelp := "Se crea en tu cuenta de Atlassian: Seguridad › Tokens de API. No se muestra al escribirlo."
	if c.hasToken {
		tokenHelp = "Vacío conserva el actual. No se muestra al escribirlo."
	}
	fields := []huh.Field{
		huh.NewInput().Title("Email").Placeholder("tu@example.test").Value(&c.email).Validate(func(s string) error {
			if strings.TrimSpace(s) == "" {
				return errors.New("falta el email")
			}
			return nil
		}),
		huh.NewInput().Title("API token").Description(tokenHelp).EchoMode(huh.EchoModeNone).Value(&c.token).Validate(func(s string) error {
			if strings.TrimSpace(s) == "" && !c.hasToken {
				return errors.New("falta el token")
			}
			return nil
		}),
		huh.NewInput().Title("URL del sitio").Placeholder("https://empresa.atlassian.net").Value(&c.url).Validate(func(s string) error {
			_, err := config.NormalizeBaseURL(s)
			return err
		}),
	}
	if c.withSpace {
		fields = append(fields, huh.NewInput().Title("Space de Confluence").Placeholder("DEV").
			Description("Donde se publican los documentos. Un proyecto puede usar otro").Value(&c.space))
	}
	return newForm(fields...)
}

func (c *connection) title() string       { return "Conexión" }
func (c *connection) keys() []key.Binding { return []key.Binding{nextFieldKey, prevFieldKey} }

func (c *connection) captures(k tea.KeyPressMsg) bool { return !key.Matches(k, backKey) }

func (c *connection) init() tea.Cmd {
	if c.form.State != huh.StateNormal {
		c.form = c.build()
	}
	return c.form.Init()
}

func (c *connection) update(msg tea.Msg) (screen, tea.Cmd) {
	m, cmd := c.form.Update(msg)
	c.form = m.(*huh.Form)
	if c.form.State != huh.StateCompleted {
		return c, cmd
	}
	email, token, space := strings.TrimSpace(c.email), strings.TrimSpace(c.token), strings.TrimSpace(c.space)
	url, _ := config.NormalizeBaseURL(c.url)
	c.token = ""
	err := c.parent.st.save(func(cfg *config.Config) error {
		cfg.AtlassianEmail, cfg.BaseURL = email, url
		if token != "" {
			cfg.AtlassianToken = token
		}
		if c.withSpace {
			cfg.ConfluenceSpaceKey = space
		}
		return nil
	})
	if err != nil {
		c.err = "No se pudo guardar: " + err.Error()
		c.form = c.build()
		return c, c.form.Init()
	}
	return c, tea.Sequence(back, open(newAccountCheck(c.parent)))
}

func (c *connection) view(width, height int) string {
	var notes []string
	for _, v := range c.env {
		notes = append(notes, warnStyle.Render("⚠ "+v+" está definida en el entorno: gana sobre lo que guardes aquí al usar draco."))
	}
	body := formView(c.form, c.err, width)
	if len(notes) == 0 {
		return body
	}
	top := lipgloss.NewStyle().Padding(1, 2, 0).Width(width).Render(lipgloss.JoinVertical(lipgloss.Left, notes...))
	return lipgloss.JoinVertical(lipgloss.Left, top, body)
}

type checkedMsg struct {
	owner        *accountCheck
	user         *provider.User
	email        string // the one saved
	firstProject bool   // the file has no projects yet
	err          error
}

var (
	continueKey = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "continuar"))
	fixEmailKey = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "usar el email de la cuenta"))
)

// accountCheck verifies the saved credentials against the tracker, as draco
// init does, and offers to fix an email that is not the account's.
type accountCheck struct {
	parent  *settings
	started bool
	result  *checkedMsg
	fixed   string
	saveErr error
	spinner spinner.Model
	ctx     context.Context
	cancel  context.CancelFunc
}

func newAccountCheck(parent *settings) *accountCheck {
	a := &accountCheck{parent: parent, spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot))}
	a.ctx, a.cancel = context.WithCancel(context.Background())
	return a
}

func (a *accountCheck) title() string { return "Verificar la cuenta" }
func (a *accountCheck) close()        { a.cancel() }

func (a *accountCheck) keys() []key.Binding {
	switch {
	case a.result == nil:
		return nil
	case a.mismatch():
		return []key.Binding{continueKey, fixEmailKey}
	}
	return []key.Binding{continueKey}
}

func (a *accountCheck) init() tea.Cmd {
	if a.started {
		return nil
	}
	a.started = true
	d, ctx := a.parent.st.deps, a.ctx
	return tea.Batch(a.spinner.Tick, func() tea.Msg {
		msg := checkedMsg{owner: a}
		file, err := d.LoadFile()
		if err != nil {
			msg.err = err
			return msg
		}
		msg.email, msg.firstProject = file.AtlassianEmail, len(file.Projects) == 0
		bundle, err := d.OpenProfile(file, profile.Options{AnyScope: true})
		if err != nil {
			msg.err = explainProfile(err)
			return msg
		}
		msg.user, msg.err = bundle.Tracker.Me(ctx)
		return msg
	})
}

func (a *accountCheck) mismatch() bool {
	r := a.result
	return r != nil && r.err == nil && a.fixed == "" && !profile.SameAccount(r.email, r.user)
}

func (a *accountCheck) update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case checkedMsg:
		if msg.owner == a && a.ctx.Err() == nil {
			a.result = &msg
		}
	case spinner.TickMsg:
		if a.result != nil {
			return a, nil
		}
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		return a, cmd
	case tea.KeyPressMsg:
		switch {
		case a.result == nil:
		case key.Matches(msg, fixEmailKey) && a.mismatch():
			email := a.result.user.Email
			if a.saveErr = a.parent.st.save(func(c *config.Config) error { c.AtlassianEmail = email; return nil }); a.saveErr == nil {
				a.fixed = email
			}
		case key.Matches(msg, continueKey) && a.result.err == nil && a.result.firstProject:
			ps := newProjects(a.parent.st)
			return a, tea.Sequence(back, open(ps), open(newProjectForm(ps, "")))
		case key.Matches(msg, continueKey):
			return a, back
		}
	}
	return a, nil
}

func (a *accountCheck) view(width, height int) string {
	text := lipgloss.NewStyle().Width(width - 2)
	r := a.result
	if r == nil {
		return pad(a.spinner.View() + " Verificando la cuenta…")
	}
	if r.err != nil {
		return pad(text.Render(lipgloss.JoinVertical(lipgloss.Left,
			errStyle.Render(fmt.Sprintf("✗ No se pudo validar las credenciales: %v", r.err)), "",
			"Lo guardado se conserva. Revisa el email, el token y la URL, y vuelve a abrir Conexión.", "",
			faintStyle.Render("enter vuelve a Configuración"))))
	}
	lines := []string{okStyle.Render("✓ ") + "Autenticado como " + strings.TrimSpace(r.user.Name+" "+parens(r.user.Email))}
	switch {
	case a.fixed != "":
		lines = append(lines, okStyle.Render("✓ ")+"Email corregido a "+a.fixed)
	case a.saveErr != nil:
		lines = append(lines, errStyle.Render("✗ No se pudo guardar: "+a.saveErr.Error()))
	case a.mismatch():
		lines = append(lines, "", warnStyle.Render(fmt.Sprintf("⚠ El email guardado (%s) no es el de la cuenta autenticada (%s): las búsquedas por autor volverían vacías. s usa el de la cuenta.", r.email, r.user.Email)))
	}
	next := "enter vuelve a Configuración"
	if r.firstProject {
		next = "enter sigue con el primer proyecto: dónde están tus repos"
	}
	lines = append(lines, "", faintStyle.Render(next))
	return pad(text.Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))
}

func parens(s string) string {
	if s == "" {
		return ""
	}
	return "(" + s + ")"
}
