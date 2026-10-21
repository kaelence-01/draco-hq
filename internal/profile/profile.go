// Package profile builds the adapters a profile names, so callers (the CLI and
// the TUI) never pick an adapter themselves.
package profile

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/provider"
	"github.com/kaelence-01/draco-hq/internal/provider/atlassian"
	"github.com/kaelence-01/draco-hq/internal/provider/git"
	"github.com/kaelence-01/draco-hq/internal/provider/markdown"
)

var (
	// ErrNoSpace means the profile publishes to Confluence and no space key
	// came from the request, the project or the profile.
	ErrNoSpace = errors.New("Confluence space key no configurado")

	// ErrNoQA means the profile's tracker cannot produce the QA consolidated
	// report.
	ErrNoQA = errors.New("el tracker de este perfil no soporta el consolidado QA")

	// ErrNoCredentials means a Jira or Confluence profile without
	// atlassian_email, atlassian_token or base_url.
	ErrNoCredentials = errors.New("configuración incompleta de Atlassian. Corre: draco init")
)

// Options choose the profile and refine it for one run.
type Options struct {
	Name string // empty: default_profile (see config.Profile)

	// Scope overrides the destination scope: a Confluence space key, or a
	// Markdown folder. Empty uses ProjectSpace or the profile's.
	Scope string

	// ProjectSpace is the project's Confluence space: it comes before the
	// profile's, and a Markdown destination ignores it.
	ProjectSpace string

	// AnyScope lets a Confluence destination work without a space, searching
	// every space (exporting does).
	AnyScope bool
}

// Bundle is a profile with its adapters built.
type Bundle struct {
	Name     string
	Profile  config.Profile
	Tracker  provider.Tracker
	Docs     provider.DocStore
	Changes  provider.ChangeSource
	Template string // template folder of the profile; empty: repo's or user's

	qa    provider.QATracker
	qaErr error
}

// Open builds the adapters of the profile.
func Open(cfg *config.Config, o Options) (*Bundle, error) {
	p, name, err := cfg.Profile(o.Name)
	if err != nil {
		return nil, err
	}
	b := &Bundle{Name: name, Profile: *p, Changes: git.Local{}, Template: config.ExpandHome(p.Template)}

	var client *atlassian.Client
	atlassianClient := func() (*atlassian.Client, error) {
		if client == nil {
			if !cfg.HasAtlassianCredentials() {
				return nil, ErrNoCredentials
			}
			client = atlassian.NewClient(cfg.BaseURL, cfg.AtlassianEmail, cfg.AtlassianToken)
		}
		return client, nil
	}

	switch p.Tracker.Provider {
	case "jira":
		c, err := atlassianClient()
		if err != nil {
			return nil, err
		}
		b.Tracker = atlassian.NewTracker(c)
		if err := cfg.QAJira.Validate(); err != nil {
			b.qaErr = err
		} else {
			b.qa = atlassian.NewQATracker(c, cfg.QAJira)
		}
	default:
		return nil, unknown(name, "tracker", p.Tracker.Provider, "jira")
	}

	scope := o.Scope
	switch p.Docs.Provider {
	case "confluence":
		if scope == "" {
			scope = o.ProjectSpace
		}
		if scope == "" {
			scope = p.Docs.Space
		}
		if scope == "" && !o.AnyScope {
			return nil, ErrNoSpace
		}
		c, err := atlassianClient()
		if err != nil {
			return nil, err
		}
		b.Docs = atlassian.NewDocStore(c, scope)
	case "markdown":
		if scope == "" {
			scope = config.ExpandHome(p.Docs.Path)
		}
		if scope == "" {
			return nil, fmt.Errorf("perfil '%s': docs.path es obligatorio con provider: markdown", name)
		}
		if b.Docs, err = markdown.New(scope); err != nil {
			return nil, err
		}
	default:
		return nil, unknown(name, "docs", p.Docs.Provider, "confluence", "markdown")
	}
	return b, nil
}

// QA returns the tracker as a QA tracker: ErrNoQA when the tracker has no such
// capability, or why its QA settings (qa_jira) are not usable.
func (b *Bundle) QA() (provider.QATracker, error) {
	if b.qa != nil {
		return b.qa, nil
	}
	if b.qaErr != nil {
		return nil, b.qaErr
	}
	if q, ok := b.Tracker.(provider.QATracker); ok {
		return q, nil
	}
	return nil, ErrNoQA
}

// SameAccount reports whether email is the authenticated account's. A token
// for another account works, but searches by author would come back empty; an
// account that hides its email counts as the same.
func SameAccount(email string, u *provider.User) bool {
	return u.Email == "" || strings.EqualFold(u.Email, email)
}

func unknown(profile, role, got string, valid ...string) error {
	if got == "" {
		return fmt.Errorf("perfil '%s': falta %s.provider (%s)", profile, role, strings.Join(valid, ", "))
	}
	return fmt.Errorf("perfil '%s': %s.provider %q no existe (%s)", profile, role, got, strings.Join(valid, ", "))
}
