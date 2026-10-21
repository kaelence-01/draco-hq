package profile

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/provider"
	"github.com/kaelence-01/draco-hq/internal/provider/atlassian"
	"github.com/kaelence-01/draco-hq/internal/provider/markdown"
)

func todaysConfig() *config.Config {
	return &config.Config{
		AtlassianEmail: "qa@example.test", AtlassianToken: "token", BaseURL: "https://empresa.atlassian.net",
		ConfluenceSpaceKey: "DEV",
		QAJira: &config.QAJiraConfig{
			Project: "OPS", Statuses: []string{"10001"},
			Fields: config.QAJiraFields{CodingErrors: "customfield_1", DevReturns: "customfield_2", DevStatus: "customfield_3"},
		},
	}
}

func TestTodaysConfigOpensJiraAndConfluence(t *testing.T) {
	b, err := Open(todaysConfig(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if b.Name != config.LegacyProfile {
		t.Errorf("perfil = %q", b.Name)
	}
	if _, ok := b.Tracker.(*atlassian.Tracker); !ok {
		t.Errorf("tracker = %T", b.Tracker)
	}
	if _, ok := b.Docs.(*atlassian.DocStore); !ok {
		t.Errorf("docs = %T", b.Docs)
	}
	if qa, err := b.QA(); err != nil || qa == nil {
		t.Errorf("con qa_jira válido el consolidado está disponible: %v", err)
	}
}

func TestQAExplainsWhatIsMissing(t *testing.T) {
	cfg := todaysConfig()
	cfg.QAJira = nil
	b, err := Open(cfg, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.QA(); err == nil || !strings.Contains(err.Error(), "qa_jira") {
		t.Errorf("sin qa_jira el error debe pedirlo: %v", err)
	}
	if _, err := (&Bundle{Tracker: plainTracker{}}).QA(); !errors.Is(err, ErrNoQA) {
		t.Errorf("un tracker sin la capacidad es ErrNoQA: %v", err)
	}
}

func TestConfluenceNeedsASpace(t *testing.T) {
	cfg := todaysConfig()
	cfg.ConfluenceSpaceKey = ""
	if _, err := Open(cfg, Options{}); !errors.Is(err, ErrNoSpace) {
		t.Errorf("esperaba ErrNoSpace, obtuve %v", err)
	}
	if _, err := Open(cfg, Options{AnyScope: true}); err != nil {
		t.Errorf("exportar busca en todos los spaces: %v", err)
	}
	if _, err := Open(cfg, Options{Scope: "FLAG"}); err != nil {
		t.Errorf("un space explícito alcanza: %v", err)
	}
}

func TestMarkdownProfile(t *testing.T) {
	cfg := todaysConfig()
	cfg.Profiles = map[string]*config.Profile{"externo": {
		Tracker:  config.ProviderConfig{Provider: "jira"},
		Docs:     config.ProviderConfig{Provider: "markdown", Path: t.TempDir()},
		Template: "./plantillas",
	}}
	b, err := Open(cfg, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := b.Docs.(*markdown.Store); !ok || b.Template != "./plantillas" || b.Name != "externo" {
		t.Errorf("bundle = %+v (docs %T)", b, b.Docs)
	}
	cfg.Profiles["externo"].Docs.Path = ""
	if _, err := Open(cfg, Options{}); err == nil || !strings.Contains(err.Error(), "docs.path") {
		t.Errorf("markdown sin carpeta: %v", err)
	}
}

func TestConfigErrors(t *testing.T) {
	cfg := todaysConfig()
	cfg.AtlassianToken = ""
	if _, err := Open(cfg, Options{}); !errors.Is(err, ErrNoCredentials) {
		t.Errorf("sin credenciales: %v", err)
	}
	cfg = todaysConfig()
	cfg.Profiles = map[string]*config.Profile{"raro": {Tracker: config.ProviderConfig{Provider: "jira"}, Docs: config.ProviderConfig{Provider: "notion"}}}
	if _, err := Open(cfg, Options{}); err == nil || !strings.Contains(err.Error(), `"notion" no existe (confluence, markdown)`) {
		t.Errorf("destino desconocido: %v", err)
	}
	cfg.Profiles["raro"] = &config.Profile{Docs: config.ProviderConfig{Provider: "markdown", Path: "x"}}
	if _, err := Open(cfg, Options{}); err == nil || !strings.Contains(err.Error(), "falta tracker.provider") {
		t.Errorf("tracker sin provider: %v", err)
	}
}

func TestSameAccount(t *testing.T) {
	cases := []struct {
		email, account string
		want           bool
	}{
		{"tu@example.test", "Tu@Example.test", true},
		{"tu@example.test", "", true},
		{"tu@example.test", "otra@example.test", false},
	}
	for _, c := range cases {
		if got := SameAccount(c.email, &provider.User{Email: c.account}); got != c.want {
			t.Errorf("SameAccount(%q, %q) = %v", c.email, c.account, got)
		}
	}
}

type plainTracker struct{}

func (plainTracker) Issue(context.Context, string) (*provider.Issue, error) { return nil, nil }
func (plainTracker) Me(context.Context) (*provider.User, error)             { return nil, nil }
func (plainTracker) LinkDocument(context.Context, string, provider.DocRef) error {
	return nil
}
