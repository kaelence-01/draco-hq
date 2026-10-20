package config

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestTodaysConfigReadsAsTheAtlassianProfile(t *testing.T) {
	var cfg Config
	raw := "atlassian_email: qa@example.test\nbase_url: https://empresa.atlassian.net\nconfluence_space_key: DEV\n"
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}
	p, name, err := cfg.Profile("")
	if err != nil {
		t.Fatal(err)
	}
	want := &Profile{Tracker: ProviderConfig{Provider: "jira"}, Docs: ProviderConfig{Provider: "confluence", Space: "DEV"}}
	if name != LegacyProfile || !reflect.DeepEqual(p, want) {
		t.Errorf("perfil = %q %+v", name, p)
	}
	if cfg.Profiles != nil {
		t.Error("la migración es en memoria: no agrega perfiles a la config")
	}
	if _, _, err := cfg.Profile("otro"); err == nil {
		t.Error("sin perfiles, pedir uno que no es el migrado es un error")
	}
}

func TestProfileSelection(t *testing.T) {
	var cfg Config
	raw := `
default_profile: equipo
profiles:
  equipo:
    tracker: { provider: jira }
    docs: { provider: confluence, space: DEV }
  externo:
    tracker: { provider: jira }
    docs: { provider: markdown, path: ./docs/despliegues }
    template: ./plantillas/deploy
`
	if err := yaml.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}
	if _, name, _ := cfg.Profile(""); name != "equipo" {
		t.Errorf("sin nombre se usa default_profile, obtuve %q", name)
	}
	p, name, err := cfg.Profile("externo")
	if err != nil || name != "externo" || p.Docs.Path != "./docs/despliegues" || p.Template != "./plantillas/deploy" {
		t.Errorf("externo = %q %+v %v", name, p, err)
	}
	if _, _, err := cfg.Profile("nada"); err == nil || !strings.Contains(err.Error(), "equipo, externo") {
		t.Errorf("un perfil inexistente debe listar los que hay: %v", err)
	}
	cfg.DefaultProfile = ""
	if _, _, err := cfg.Profile(""); err == nil {
		t.Error("varios perfiles sin default es ambiguo")
	}
	delete(cfg.Profiles, "equipo")
	if _, name, _ := cfg.Profile(""); name != "externo" {
		t.Errorf("con un solo perfil se usa ese, obtuve %q", name)
	}
}

func TestProjectAndProfileEditing(t *testing.T) {
	cfg := &Config{}
	if err := cfg.SetProject("pagos", &ProjectConfig{BackendRepo: "pagos-api"}); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetProject("mal nombre", &ProjectConfig{}); err == nil {
		t.Error("un nombre con espacios es inválido")
	}
	if err := cfg.SetDefaultProject("pagos"); err != nil || cfg.DefaultProject != "pagos" {
		t.Errorf("default = %q %v", cfg.DefaultProject, err)
	}
	if err := cfg.SetDefaultProject("nada"); err == nil {
		t.Error("no se puede elegir por defecto un proyecto inexistente")
	}
	if err := cfg.RemoveProject("pagos"); err != nil || cfg.DefaultProject != "" || len(cfg.ProjectNames()) != 0 {
		t.Errorf("quitar el proyecto por defecto deja sin default: %q %v", cfg.DefaultProject, err)
	}

	if err := cfg.SetProfile("externo", &Profile{Docs: ProviderConfig{Provider: "markdown", Path: "./docs"}}); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetDefaultProfile("externo"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.RemoveProfile("externo"); err != nil || cfg.DefaultProfile != "" {
		t.Errorf("quitar el perfil por defecto deja sin default: %q %v", cfg.DefaultProfile, err)
	}
	if err := cfg.RemoveProfile("externo"); err == nil {
		t.Error("quitar un perfil inexistente es un error")
	}
}

func TestProfileTemplateDir(t *testing.T) {
	p := &Profile{Template: "/plantillas/equipo"}
	if got := p.TemplateDir("qa"); got != filepath.Join("/plantillas/equipo", "qa") {
		t.Errorf("TemplateDir = %q", got)
	}
	if got := (&Profile{}).TemplateDir("deploy"); got != "" {
		t.Errorf("sin carpeta no hay plantilla del perfil: %q", got)
	}
}
