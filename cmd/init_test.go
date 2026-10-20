package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/profile"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

// configHome points the config at a temporary home and returns its path.
func configHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return filepath.Join(home, ".config", "draco", "config.yaml")
}

func writeConfig(t *testing.T, cfg *config.Config) {
	t.Helper()
	if err := config.Save(cfg); err != nil {
		t.Fatal(err)
	}
}

func readConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.LoadFile()
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

type meTracker struct {
	fakeTracker
	user *provider.User
	err  error
}

func (m *meTracker) Me(context.Context) (*provider.User, error) { return m.user, m.err }

// fakeAccount answers the credential check of draco init with trk.
func fakeAccount(t *testing.T, trk provider.Tracker) {
	prev := openProfile
	t.Cleanup(func() { openProfile = prev })
	openProfile = func(cfg *config.Config, o profile.Options) (*profile.Bundle, error) {
		if !cfg.HasAtlassianCredentials() {
			return nil, profile.ErrNoCredentials
		}
		return &profile.Bundle{Tracker: trk}, nil
	}
}

func TestInitCreatesConfigProfileAndProject(t *testing.T) {
	path := configHome(t)
	fakeAccount(t, &meTracker{user: &provider.User{Name: "Dev", Email: "dev@example.test"}})

	input := strings.Join([]string{
		"dev@example.test", "secreto", "example.test", "https://example.test/", // the first URL is rejected
		"", "2", "./docs/despliegues", // profile: name by default, Markdown
		"s", "pagos", "/repos/pagos-api", "", "", "", "mi-org", "", // project
	}, "\n") + "\n"
	code, out, errOut := runWith(input, "init")
	if code != exitOK {
		t.Fatalf("código %d\n%s\n%s", code, out, errOut)
	}
	for _, want := range []string{"debe empezar con https://", "Configuración guardada", "Autenticado como Dev (dev@example.test)", "Proyecto 'pagos' guardado", "Listo."} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en:\n%s", want, out)
		}
	}
	if strings.Contains(out, "secreto") {
		t.Error("el token nunca se muestra")
	}

	cfg := readConfig(t)
	p := cfg.Profiles["equipo"]
	if cfg.AtlassianToken != "secreto" || cfg.BaseURL != "https://example.test" || cfg.DefaultProfile != "equipo" ||
		p == nil || p.Docs.Provider != "markdown" || p.Docs.Path != "./docs/despliegues" || p.Tracker.Provider != "jira" {
		t.Errorf("config guardada: %+v %+v", cfg, p)
	}
	proj := cfg.Projects["pagos"]
	if cfg.DefaultProject != "pagos" || proj == nil || proj.BackendRepo != "pagos-api" || proj.VCSOrg != "mi-org" || proj.FrontendPath != "" {
		t.Errorf("proyecto: %q %+v", cfg.DefaultProject, proj)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("la config guarda el token: permisos 0600: %v %v", info.Mode(), err)
	}
}

func TestInitOnExistingConfigKeepsTheRest(t *testing.T) {
	configHome(t)
	writeConfig(t, &config.Config{
		AtlassianEmail: "dev@example.test", AtlassianToken: "anterior", BaseURL: "https://example.test",
		Projects: map[string]*config.ProjectConfig{"pagos": {BackendPath: "/repos/pagos-api"}}, DefaultProject: "pagos",
		Profiles: map[string]*config.Profile{"equipo": {Tracker: config.ProviderConfig{Provider: "jira"}, Docs: config.ProviderConfig{Provider: "confluence", Space: "DEV"}}},
	})
	fakeAccount(t, &meTracker{user: &provider.User{Email: "otra@example.test"}})

	if code, out, _ := runWith("n\n", "init"); code != exitOK || !strings.Contains(out, "Cancelado") {
		t.Fatalf("responder n no cambia nada: %d\n%s", code, out)
	}

	// Continue, Enter keeps email, token and URL; accept the account's email; no project.
	code, out, errOut := runWith("s\n\n\n\n\nn\n", "init")
	if code != exitOK || !strings.Contains(out, "Perfiles existentes: equipo") || !strings.Contains(out, "Email corregido a otra@example.test") {
		t.Fatalf("código %d\n%s\n%s", code, out, errOut)
	}
	cfg := readConfig(t)
	if cfg.AtlassianToken != "anterior" || cfg.AtlassianEmail != "otra@example.test" || cfg.Projects["pagos"] == nil || cfg.Profiles["equipo"].Docs.Space != "DEV" {
		t.Errorf("se pierde algo de la config anterior: %+v", cfg)
	}
}

func TestInitReportsBadCredentialsAndEndOfInput(t *testing.T) {
	configHome(t)
	fakeAccount(t, &meTracker{err: errors.New("no autorizado (401)")})
	code, _, errOut := runWith("dev@example.test\nmalo\nhttps://example.test\n\n1\nDEV\n", "init")
	if code != exitError || !strings.Contains(errOut, "401") || !strings.Contains(errOut, "vuelve a correr draco init") {
		t.Errorf("credenciales inválidas: %d %s", code, errOut)
	}

	configHome(t)
	if code, _, errOut := runWith("dev@example.test\n", "init"); code != exitError || !strings.Contains(errOut, "la entrada terminó") {
		t.Errorf("sin más respuestas no se queda esperando: %d %s", code, errOut)
	}
}
