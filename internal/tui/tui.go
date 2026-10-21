// Package tui is draco's visual mode: screens over the same flows as the
// command mode, with no business logic of its own.
package tui

import (
	"context"
	"io"

	tea "charm.land/bubbletea/v2"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/profile"
)

// Deps are what the command mode passes in; its tests replace them.
type Deps struct {
	LoadConfig func() (*config.Config, error)
	Version    string
	// LoadFile and SaveConfig change the config: only the file, never the
	// environment, so a token from ATLASSIAN_TOKEN is not written to disk.
	LoadFile   func() (*config.Config, error)
	SaveConfig func(*config.Config) error
	// OpenProfile builds the adapters of a profile (profile.Open).
	OpenProfile func(*config.Config, profile.Options) (*profile.Bundle, error)
}

// Run shows the visual mode on a terminal until the user quits.
func Run(ctx context.Context, d Deps, in io.Reader, out io.Writer) error {
	_, err := tea.NewProgram(newApp(d), tea.WithContext(ctx), tea.WithInput(in), tea.WithOutput(out)).Run()
	return err
}

// state is what every screen shares: the config and the active choices.
type state struct {
	deps Deps

	cfg           *config.Config // nil when it could not be read
	profile       string
	tracker, docs string
	project       string // active project of this session, never saved
	notice        string // a problem with the config, shown on the home screen
}

func loadState(d Deps) *state {
	st := &state{deps: d}
	st.load()
	return st
}

// load reads the config again; the active project stays while it exists.
func (st *state) load() {
	active := st.project
	st.cfg, st.profile, st.tracker, st.docs, st.project, st.notice = nil, "", "", "", "", ""
	cfg, err := st.deps.LoadConfig()
	if err != nil {
		st.notice = "No se pudo leer la configuración: " + err.Error()
		return
	}
	st.cfg = cfg
	_, st.project, _ = cfg.GetProject("")
	if _, ok := cfg.Projects[active]; ok {
		st.project = active
	}
	p, name, err := cfg.Profile("")
	if err != nil {
		st.notice = err.Error()
		return
	}
	st.profile, st.tracker, st.docs = name, p.Tracker.Provider, p.Docs.Provider
	if (st.tracker == "jira" || st.docs == "confluence") && !cfg.HasAtlassianCredentials() {
		st.notice = "Falta configurar la conexión con Atlassian: corre draco init"
	}
}

// save changes the config file with change and reads the config again.
func (st *state) save(change func(*config.Config) error) error {
	cfg, err := st.deps.LoadFile()
	if err != nil {
		return err
	}
	if err := change(cfg); err != nil {
		return err
	}
	if err := st.deps.SaveConfig(cfg); err != nil {
		return err
	}
	st.load()
	return nil
}
