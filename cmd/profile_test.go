package cmd

import (
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/config"
)

func TestProfileFirstAddKeepsTheImplicitProfile(t *testing.T) {
	configHome(t)
	writeConfig(t, &config.Config{ConfluenceSpaceKey: "DEV"})

	if _, out, _ := run("profile", "list"); !strings.Contains(out, "perfil implícito 'atlassian' (jira → confluence (space DEV))") {
		t.Errorf("list sin perfiles:\n%s", out)
	}

	// Markdown, a folder, no template, not the default.
	code, out, errOut := runWith("2\n./docs/despliegues\n\nn\n", "profile", "add", "externo")
	if code != exitOK || !strings.Contains(out, "También se guarda el perfil 'atlassian'") {
		t.Fatalf("%d\n%s\n%s", code, out, errOut)
	}
	cfg := readConfig(t)
	if cfg.DefaultProfile != "atlassian" || cfg.Profiles["atlassian"].Docs.Space != "DEV" || cfg.Profiles["externo"].Docs.Path != "./docs/despliegues" {
		t.Errorf("los comandos de siempre deben seguir usando Confluence: %+v", cfg.Profiles)
	}

	_, out, _ = run("profile", "list")
	if !strings.Contains(out, "* atlassian") || !strings.Contains(out, "jira → markdown (./docs/despliegues)") {
		t.Errorf("list:\n%s", out)
	}

	if code, _, _ := run("profile", "default", "externo"); code != exitOK || readConfig(t).DefaultProfile != "externo" {
		t.Error("default")
	}
	if code, out, _ := runWith("s\n", "profile", "remove", "externo"); code != exitOK || !strings.Contains(out, "Perfil 'externo' eliminado") {
		t.Errorf("remove: %d %s", code, out)
	}
	if cfg := readConfig(t); cfg.Profiles["externo"] != nil || cfg.DefaultProfile != "" {
		t.Errorf("remove no guardó: %+v", cfg)
	}
}

func TestProfileAddConfluenceAndErrors(t *testing.T) {
	configHome(t)
	if code, out, errOut := runWith("1\nDEV\n/plantillas/equipo\n", "profile", "add", "atlassian"); code != exitOK {
		t.Fatalf("%d\n%s\n%s", code, out, errOut)
	}
	cfg := readConfig(t)
	if p := cfg.Profiles["atlassian"]; cfg.DefaultProfile != "atlassian" || len(cfg.Profiles) != 1 || p.Docs.Space != "DEV" || p.Template != "/plantillas/equipo" {
		t.Errorf("el primer perfil es el de por defecto: %+v", cfg)
	}
	if code, _, errOut := runWith("3\n", "profile", "add", "otro"); code != exitError || !strings.Contains(errOut, "opción inválida") {
		t.Errorf("opción fuera de rango: %d %s", code, errOut)
	}
	if code, _, errOut := run("profile", "rm", "nadie"); code != exitError || !strings.Contains(errOut, "perfil 'nadie' no encontrado") {
		t.Errorf("remove inexistente: %d %s", code, errOut)
	}
}
