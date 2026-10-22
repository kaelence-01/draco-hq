package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func gttHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func writeGtt(t *testing.T, home, dir, content string) string {
	t.Helper()
	path := filepath.Join(home, ".config", dir, "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

const gttYAML = `atlassian_email: tu@example.test
atlassian_token: token-de-gtt
base_url: https://empresa.example.test/
confluence_space_key: DEV
qa_email: qa@example.test
default_project: pagos
projects:
  pagos:
    backend_path: /repos/pagos-api
    backend_repo: pagos-api
    vcs_host: https://bitbucket.org
    vcs_org: mi-org
    deploy_checklist: [Pasar backend al servidor]
    backlog:
      backend:
        class_globs: ["app/**/Services/**/*.php"]
`

func TestGttConfigReadsAsADracoConfig(t *testing.T) {
	home := gttHome(t)
	if cfg, path, err := GttConfig(); cfg != nil || path != "" || err != nil {
		t.Fatalf("sin gtt no hay nada que importar: %v %q %v", cfg, path, err)
	}
	want := writeGtt(t, home, "gtt", gttYAML)
	cfg, path, err := GttConfig()
	if err != nil || path != want {
		t.Fatalf("lee ~/.config/gtt/config.yaml: %q %v", path, err)
	}
	p := cfg.Projects["pagos"]
	if cfg.AtlassianToken != "token-de-gtt" || cfg.BaseURL != "https://empresa.example.test" || cfg.ConfluenceSpaceKey != "DEV" ||
		p == nil || p.VCSOrg != "mi-org" || len(p.DeployChecklist) != 1 || len(p.BacklogFor("backend").ClassGlobs) != 1 {
		t.Errorf("todos los campos de gtt pasan tal cual, con la URL sin / final: %+v %+v", cfg, p)
	}
}

func TestGttConfigFallsBackToTheOldFolderAndReportsErrors(t *testing.T) {
	home := gttHome(t)
	old := writeGtt(t, home, "deploy-doc", gttYAML)
	if _, path, err := GttConfig(); path != old || err != nil {
		t.Errorf("las versiones viejas de gtt guardaban en ~/.config/deploy-doc: %q %v", path, err)
	}
	writeGtt(t, home, "gtt", "projects: [roto")
	if cfg, _, err := GttConfig(); cfg != nil || err == nil || !strings.Contains(err.Error(), "formato inválido") {
		t.Errorf("un config de gtt dañado se reporta, no se importa a medias: %v %v", cfg, err)
	}
}

func TestImportNotesNeverShowTheToken(t *testing.T) {
	cfg := &Config{AtlassianEmail: "tu@example.test", AtlassianToken: "token-de-gtt", BaseURL: "https://empresa.example.test",
		QAEmail: "qa@example.test", DefaultProject: "pagos", Projects: map[string]*ProjectConfig{"pagos": {}}}
	notes := strings.Join(ImportNotes(cfg), "\n")
	for _, want := range []string{"tu@example.test en https://empresa.example.test", "Proyectos: 1 (por defecto: pagos)", "qa_jira"} {
		if !strings.Contains(notes, want) {
			t.Errorf("falta %q:\n%s", want, notes)
		}
	}
	if strings.Contains(notes, "token-de-gtt") {
		t.Error("el resumen nunca muestra el token")
	}
	if !(&Config{}).IsEmpty() || cfg.IsEmpty() {
		t.Error("IsEmpty solo es cierto para una config sin nada")
	}
}
