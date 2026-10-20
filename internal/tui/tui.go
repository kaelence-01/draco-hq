// Package tui is draco's visual mode: screens over the same flows as the
// command mode, with no business logic of its own.
package tui

import (
	"context"
	"io"

	tea "charm.land/bubbletea/v2"

	"github.com/kaelence-01/draco-hq/internal/config"
)

// Deps are what the command mode passes in; its tests replace them.
type Deps struct {
	LoadConfig func() (*config.Config, error)
	Version    string
}

// Run shows the visual mode on a terminal until the user quits.
func Run(ctx context.Context, d Deps, in io.Reader, out io.Writer) error {
	_, err := tea.NewProgram(newApp(d), tea.WithContext(ctx), tea.WithInput(in), tea.WithOutput(out)).Run()
	return err
}

// state is what every screen shares: the config and the active choices.
type state struct {
	deps Deps

	profile       string
	tracker, docs string
	project       string // active project of this session, never saved
	notice        string // a problem with the config, shown on the home screen
}

func loadState(d Deps) *state {
	st := &state{deps: d}
	cfg, err := d.LoadConfig()
	if err != nil {
		st.notice = "No se pudo leer la configuración: " + err.Error()
		return st
	}
	_, st.project, _ = cfg.GetProject("")
	p, name, err := cfg.Profile("")
	if err != nil {
		st.notice = err.Error()
		return st
	}
	st.profile, st.tracker, st.docs = name, p.Tracker.Provider, p.Docs.Provider
	if (st.tracker == "jira" || st.docs == "confluence") && !cfg.HasAtlassianCredentials() {
		st.notice = "Falta configurar la conexión con Atlassian: corre draco init"
	}
	return st
}
