package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/config"
)

func TestProjectAddListDefaultRemove(t *testing.T) {
	configHome(t)
	code, out, errOut := runWith("/repos/pagos-api\n\n/repos/pagos-web\nweb\nhttps://github.com\nmi-org\nDEV\n", "project", "add", "pagos")
	if code != exitOK || !strings.Contains(out, "Proyecto 'pagos' guardado") {
		t.Fatalf("add: %d\n%s\n%s", code, out, errOut)
	}
	if code, _, _ := runWith("/repos/fac-api\n\n\n\n\n\nn\n", "project", "add", "facturacion"); code != exitOK {
		t.Fatal("segundo proyecto")
	}
	cfg := readConfig(t)
	p := cfg.Projects["pagos"]
	if cfg.DefaultProject != "pagos" || p.BackendRepo != "pagos-api" || p.FrontendRepo != "web" || p.VCSHost != "https://github.com" || p.ConfluenceSpaceKey != "DEV" {
		t.Errorf("proyecto guardado: %q %+v", cfg.DefaultProject, p)
	}

	_, out, _ = run("project", "list")
	if !strings.Contains(out, "* pagos") || !strings.Contains(out, "  facturacion") || !strings.Contains(out, "/repos/pagos-web") {
		t.Errorf("list:\n%s", out)
	}

	if code, out, _ := run("project", "default", "facturacion"); code != exitOK || readConfig(t).DefaultProject != "facturacion" {
		t.Errorf("default: %d %s", code, out)
	}
	if code, out, _ := runWith("s\n", "project", "rm", "facturacion"); code != exitOK || !strings.Contains(out, "era el proyecto por defecto") {
		t.Errorf("remove: %d %s", code, out)
	}
	if cfg := readConfig(t); cfg.Projects["facturacion"] != nil || cfg.DefaultProject != "" {
		t.Errorf("remove no guardó: %+v", cfg)
	}
}

func TestProjectAddAsksItsProfile(t *testing.T) {
	configHome(t)
	writeConfig(t, &config.Config{DefaultProfile: "equipo", Profiles: map[string]*config.Profile{"equipo": {}, "servicios": {}}})
	code, out, errOut := runWith("/repos/api-pagos\n\n\n\n\n\nnada\nservicios\n", "project", "add", "api-pagos")
	if code != exitOK || !strings.Contains(out, "equipo, servicios") || !strings.Contains(out, "el perfil 'nada' no existe") {
		t.Fatalf("pregunta el perfil entre los que hay: %d\n%s\n%s", code, out, errOut)
	}
	if p := readConfig(t).Projects["api-pagos"]; p == nil || p.Profile != "servicios" {
		t.Errorf("se guarda el perfil del proyecto: %+v", p)
	}
	if _, out, _ := run("project", "list"); !strings.Contains(out, "PERFIL") || !strings.Contains(out, "servicios") {
		t.Errorf("list muestra el perfil:\n%s", out)
	}
	if code, _, errOut := runWith("s\n", "profile", "rm", "servicios"); code == exitOK || !strings.Contains(errOut, "lo usan los proyectos api-pagos") {
		t.Errorf("un perfil en uso no se elimina: %d %s", code, errOut)
	}
}

func TestProjectAddReplaceKeepsWhatItDoesNotAsk(t *testing.T) {
	configHome(t)
	writeConfig(t, &config.Config{DefaultProject: "pagos", Projects: map[string]*config.ProjectConfig{"pagos": {
		BackendPath: "/repos/pagos-api", BackendRepo: "pagos-api", DeployChecklist: []string{"Migrar"},
		Backlog: &config.BacklogConfig{Backend: &config.BacklogRepoConfig{Since: "30d"}},
	}}})
	if code, out, _ := runWith("n\n", "project", "add", "pagos"); code != exitOK || !strings.Contains(out, "Cancelado") {
		t.Fatalf("no reemplazar: %d\n%s", code, out)
	}
	// Replace, Enter keeps every value shown.
	if code, out, errOut := runWith("s\n\n\n\n\n\n\n", "project", "add", "pagos"); code != exitOK {
		t.Fatalf("%d\n%s\n%s", code, out, errOut)
	}
	p := readConfig(t).Projects["pagos"]
	if p.BackendPath != "/repos/pagos-api" || len(p.DeployChecklist) != 1 || p.Backlog == nil {
		t.Errorf("se perdió lo que el asistente no pregunta: %+v", p)
	}
}

func TestProjectAddNeverSavesTheEnvironmentToken(t *testing.T) {
	path := configHome(t)
	t.Setenv("ATLASSIAN_TOKEN", "token-del-entorno")
	if code, _, _ := runWith("/repos/pagos-api\n\n\n\n\n\n", "project", "add", "pagos"); code != exitOK {
		t.Fatal("add")
	}
	b, _ := os.ReadFile(path)
	if strings.Contains(string(b), "token-del-entorno") {
		t.Errorf("un token de las variables de entorno nunca se escribe en el archivo:\n%s", b)
	}
}

func TestProjectErrors(t *testing.T) {
	configHome(t)
	cases := []struct {
		args []string
		code int
		want string
	}{
		{[]string{"project", "renombrar"}, exitUsage, "subcomando desconocido: project renombrar"},
		{[]string{"project", "default"}, exitUsage, "uso: draco project default <nombre>"},
		{[]string{"project", "default", "pagos"}, exitError, "proyecto 'pagos' no encontrado"},
		{[]string{"project", "add", "mi proyecto"}, exitError, "sin espacios"},
	}
	for _, c := range cases {
		if code, _, errOut := run(c.args...); code != c.code || !strings.Contains(errOut, c.want) {
			t.Errorf("%v: %d %q, esperaba %d %q", c.args, code, errOut, c.code, c.want)
		}
	}
	if code, out, _ := runWith("\n\n", "project", "add", "pagos"); code != exitOK || !strings.Contains(out, "no se guardó") {
		t.Errorf("sin rutas no se guarda: %d\n%s", code, out)
	}
	if _, out, _ := run("project", "list"); !strings.Contains(out, "No hay proyectos") {
		t.Errorf("list vacío:\n%s", out)
	}
}
