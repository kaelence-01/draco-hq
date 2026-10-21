package tui

import (
	"errors"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/kaelence-01/draco-hq/internal/config"
)

// profileMode follows the command each form repeats.
type profileMode int

const (
	profileNew  profileMode = iota // draco profile add: the implicit profile is kept
	profileEdit                    // draco profile add over an existing one
)

// profileForm creates or edits a profile. The tracker is Jira, the only one.
type profileForm struct {
	st         *state
	mode       profileMode
	name       string
	docs       string // confluence | markdown
	space      string
	path       string
	template   string
	setDefault bool
	report     func(note string, err error)
	finish     tea.Cmd
	form       *huh.Form
}

func newProfileForm(st *state, name string, mode profileMode, report func(string, error), finish tea.Cmd) *profileForm {
	f := &profileForm{st: st, mode: mode, name: name, docs: "confluence", space: st.cfg.ConfluenceSpaceKey, report: report, finish: finish}
	if old, ok := st.cfg.Profiles[name]; ok && mode == profileEdit {
		f.docs, f.space, f.path, f.template = old.Docs.Provider, old.Docs.Space, old.Docs.Path, old.Template
	}
	f.form = f.build()
	return f
}

func (f *profileForm) build() *huh.Form {
	var first []huh.Field
	if f.mode != profileEdit {
		first = append(first, huh.NewInput().Title("Nombre del perfil").Placeholder("equipo").Value(&f.name).Validate(func(s string) error {
			if err := config.ValidName("perfil", s); err != nil {
				return err
			}
			if _, ok := f.st.cfg.Profiles[s]; ok {
				return fmt.Errorf("el perfil '%s' ya existe: edítalo con e", s)
			}
			return nil
		}))
	}
	first = append(first, huh.NewSelect[string]().Title("Destino de los documentos").Description("Tracker: Jira, el único disponible").
		Options(huh.NewOption("Confluence", "confluence"), huh.NewOption("Carpeta Markdown local", "markdown")).Value(&f.docs))
	pages := []*huh.Group{
		huh.NewGroup(first...),
		huh.NewGroup(huh.NewInput().Title("Space de Confluence").Placeholder("DEV").Description("Vacío: el de cada proyecto").Value(&f.space)).
			WithHideFunc(func() bool { return f.docs != "confluence" }),
		huh.NewGroup(huh.NewInput().Title("Carpeta de los documentos").Placeholder("./docs/despliegues").Value(&f.path).Validate(func(s string) error {
			if strings.TrimSpace(s) == "" {
				return errors.New("falta la carpeta")
			}
			return nil
		})).WithHideFunc(func() bool { return f.docs != "markdown" }),
	}
	last := []huh.Field{huh.NewInput().Title("Carpeta de plantillas del perfil").Description("Opcional, con deploy/ y qa/ dentro. Vacío: la plantilla incluida").Value(&f.template)}
	if f.st.cfg.DefaultProfile != f.name || f.mode == profileNew {
		last = append(last, huh.NewConfirm().Title("¿Usarlo como perfil por defecto?").Affirmative("Sí").Negative("No").Value(&f.setDefault))
	}
	return newPagedForm(append(pages, huh.NewGroup(last...))...)
}

func (f *profileForm) title() string {
	if f.mode == profileEdit {
		return "Editar " + f.name
	}
	return "Nuevo perfil"
}

func (f *profileForm) keys() []key.Binding { return []key.Binding{nextFieldKey, prevFieldKey} }

func (f *profileForm) captures(k tea.KeyPressMsg) bool { return !key.Matches(k, backKey) }

func (f *profileForm) init() tea.Cmd { return f.form.Init() }

func (f *profileForm) update(msg tea.Msg) (screen, tea.Cmd) {
	m, cmd := f.form.Update(msg)
	f.form = m.(*huh.Form)
	if f.form.State != huh.StateCompleted {
		return f, cmd
	}
	f.report(f.save())
	return f, f.finish
}

func (f *profileForm) save() (string, error) {
	name := strings.TrimSpace(f.name)
	docs := config.ProviderConfig{Provider: f.docs}
	if f.docs == "markdown" {
		docs.Path = strings.TrimSpace(f.path)
	} else {
		docs.Space = strings.TrimSpace(f.space)
	}
	var kept *config.Profile
	err := f.st.save(func(c *config.Config) error {
		if f.mode == profileNew {
			var err error
			if kept, err = c.KeepImplicitProfile(name); err != nil {
				return err
			}
		}
		p := &config.Profile{}
		if old, ok := c.Profiles[name]; ok {
			*p = *old
		}
		p.Tracker, p.Docs, p.Template = config.ProviderConfig{Provider: "jira"}, docs, strings.TrimSpace(f.template)
		if err := c.SetProfile(name, p); err != nil {
			return err
		}
		if c.DefaultProfile == "" || f.setDefault {
			c.DefaultProfile = name
		}
		return nil
	})
	note := "Perfil guardado: " + name + " (jira → " + f.docs + ")"
	if kept != nil {
		note += ". También se guardó '" + config.LegacyProfile + "', el que se usaba hasta ahora"
	}
	return note, err
}

func (f *profileForm) view(width, height int) string { return formView(f.form, "", width) }
