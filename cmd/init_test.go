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

func TestInitCreatesConfigSpaceAndProject(t *testing.T) {
	path := configHome(t)
	fakeAccount(t, &meTracker{user: &provider.User{Name: "Dev", Email: "dev@example.test"}})

	input := strings.Join([]string{
		"dev@example.test", "secreto", "example.test", "https://example.test/", // the first URL is rejected
		"DEV",
		"s", "pagos", "/repos/pagos-api", "", "", "", "mi-org", "", // project
	}, "\n") + "\n"
	code, out, errOut := runWith(input, "init")
	if code != exitOK {
		t.Fatalf("código %d\n%s\n%s", code, out, errOut)
	}
	for _, want := range []string{"debe empezar con https://", "Configuración guardada", "Autenticado como Dev (dev@example.test)", "Generar un documento necesita un proyecto", "Proyecto 'pagos' guardado", "Listo."} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en:\n%s", want, out)
		}
	}
	if strings.Contains(out, "secreto") {
		t.Error("el token nunca se muestra")
	}

	cfg := readConfig(t)
	if cfg.AtlassianToken != "secreto" || cfg.BaseURL != "https://example.test" || cfg.ConfluenceSpaceKey != "DEV" {
		t.Errorf("config guardada: %+v", cfg)
	}
	if len(cfg.Profiles) != 0 || cfg.DefaultProfile != "" {
		t.Errorf("draco init no crea perfiles: la config sin perfiles es Jira → Confluence: %v %q", cfg.ProfileNames(), cfg.DefaultProfile)
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

const gttConfigYAML = `atlassian_email: dev@example.test
atlassian_token: token-de-gtt
base_url: https://example.test
confluence_space_key: DEV
default_project: pagos
projects:
  pagos:
    backend_path: /repos/pagos-api
    backend_repo: pagos-api
    vcs_org: mi-org
`

// writeGttConfig writes gtt's config.yaml next to draco's in the test home.
func writeGttConfig(t *testing.T, dracoPath string) string {
	t.Helper()
	path := filepath.Join(filepath.Dir(filepath.Dir(dracoPath)), "gtt", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(gttConfigYAML), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestInitImportsTheGttConfig(t *testing.T) {
	path := configHome(t)
	gtt := writeGttConfig(t, path)
	fakeAccount(t, &meTracker{user: &provider.User{Name: "Dev", Email: "dev@example.test"}})

	code, out, errOut := runWith("\n", "init")
	if code != exitOK {
		t.Fatalf("código %d\n%s\n%s", code, out, errOut)
	}
	for _, want := range []string{"Se encontró la configuración de gtt en " + gtt, "Configuración importada de gtt", "dev@example.test en https://example.test", "Proyectos: 1 (por defecto: pagos)", "Autenticado como Dev", "Listo."} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en:\n%s", want, out)
		}
	}
	if strings.Contains(out, "token-de-gtt") || strings.Contains(out, "API token") {
		t.Errorf("importar no muestra el token ni lo vuelve a pedir:\n%s", out)
	}
	cfg := readConfig(t)
	if cfg.AtlassianToken != "token-de-gtt" || cfg.ConfluenceSpaceKey != "DEV" || cfg.Projects["pagos"] == nil || cfg.Projects["pagos"].VCSOrg != "mi-org" {
		t.Errorf("la config de draco queda igual a la de gtt: %+v", cfg)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("la config importada guarda el token: permisos 0600: %v %v", info.Mode(), err)
	}
	if raw, _ := os.ReadFile(gtt); string(raw) != gttConfigYAML {
		t.Error("la configuración de gtt no se modifica")
	}
}

func TestInitWithoutImportingGttAsksAsUsual(t *testing.T) {
	path := configHome(t)
	writeGttConfig(t, path)
	fakeAccount(t, &meTracker{user: &provider.User{Name: "Dev", Email: "dev@example.test"}})
	// No to gtt, the account's email and a project.
	code, out, errOut := runWith("n\notra@example.test\nsecreto\nhttps://example.test\n\nn\nn\n", "init")
	if code != exitOK || !strings.Contains(out, "API token") || strings.Contains(out, "importada") {
		t.Fatalf("respondiendo n, init pregunta todo: %d\n%s\n%s", code, out, errOut)
	}
	if cfg := readConfig(t); cfg.AtlassianEmail != "otra@example.test" || len(cfg.Projects) != 0 {
		t.Errorf("no se importó nada de gtt: %+v", cfg)
	}

	if _, out, _ := runWith("n\n", "init"); strings.Contains(out, "gtt") {
		t.Errorf("con una config de draco no se ofrece importar:\n%s", out)
	}
}

func TestInitReportsBadCredentialsAndEndOfInput(t *testing.T) {
	configHome(t)
	fakeAccount(t, &meTracker{err: errors.New("no autorizado (401)")})
	code, _, errOut := runWith("dev@example.test\nmalo\nhttps://example.test\nDEV\n", "init")
	if code != exitError || !strings.Contains(errOut, "401") || !strings.Contains(errOut, "vuelve a correr draco init") {
		t.Errorf("credenciales inválidas: %d %s", code, errOut)
	}

	configHome(t)
	if code, _, errOut := runWith("dev@example.test\n", "init"); code != exitError || !strings.Contains(errOut, "la entrada terminó") {
		t.Errorf("sin más respuestas no se queda esperando: %d %s", code, errOut)
	}
}
