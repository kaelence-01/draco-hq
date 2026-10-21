package tui

import (
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/cli"
	"github.com/kaelence-01/draco-hq/internal/config"
)

// routeSpecs mirror the commands of the command mode that the bar opens.
var routeSpecs = []cli.Command{
	{Names: []string{"backlog"}, Summary: "Deuda técnica", Subcommands: []cli.Command{{Names: []string{"scan"}, Summary: "Analiza", Flags: []cli.Flag{
		{Long: "--project", Short: "-p", Value: true}, {Long: "--repo", Short: "-r", Value: true}, {Long: "--path", Value: true},
		{Long: "--since", Value: true}, {Long: "--modulo", Short: "-m", Value: true}, {Long: "--limit", Short: "-l", Value: true},
		{Long: "--output", Short: "-o", Value: true, Optional: true}, {Long: "--json"}, {Long: "--deps"},
	}}}},
	{Names: []string{"fetch", "f"}, Summary: "Exportar", Flags: []cli.Flag{
		{Long: "--issue", Short: "-i", Value: true}, {Long: "--output", Short: "-o", Value: true}, {Long: "--space", Short: "-s", Value: true}, {Long: "--profile", Value: true},
	}},
	{Names: []string{"generate", "gen", "g"}, Summary: "Documento de despliegue", Flags: []cli.Flag{
		{Long: "--issue", Short: "-i", Value: true}, {Long: "--commit-backend", Short: "-b", Value: true}, {Long: "--commit-frontend", Short: "-f", Value: true},
		{Long: "--project", Short: "-p", Value: true}, {Long: "--profile", Value: true}, {Long: "--space", Short: "-s", Value: true},
		{Long: "--template", Value: true}, {Long: "--dry-run"},
	}},
	{Names: []string{"init"}, Summary: "Configuración inicial"},
	{Names: []string{"profile"}, Summary: "Perfiles", Subcommands: []cli.Command{{Names: []string{"list", "ls"}}, {Names: []string{"add"}}}},
	{Names: []string{"project"}, Summary: "Proyectos", Subcommands: []cli.Command{{Names: []string{"list", "ls"}}, {Names: []string{"add"}}}},
	{Names: []string{"qa"}, Summary: "Consolidado QA", Flags: []cli.Flag{
		{Long: "--sprint", Short: "-s", Value: true}, {Long: "--module", Short: "-m", Value: true}, {Long: "--dry-run"},
	}},
	{Names: []string{"template"}, Summary: "Plantillas"},
	{Names: []string{"update"}, Summary: "Actualizar"},
}

// runBar goes back to the home screen and runs line in a new command bar.
func runBar(m *app, line string) {
	send(m, homeMsg{})
	m.stack[0].(*home).bar = nil
	m.st.deps.Commands = routeSpecs
	typeText(m, "/"+line)
	send(m, keyEnter)
}

func TestBarOpensGenerateAtTheCommits(t *testing.T) {
	e := newGenerateApp(t, nil)
	runBar(e.m, "g -i OPS-7 -b a1b2c3d,ffff999 -f e4f5")
	out := view(e.m)
	for _, want := range []string{"Inicio › Generar documento › Commits de OPS-7", "Backend (2)", "Frontend (1)"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q: -b marca a1b2c3d y agrega ffff999 a mano; -f marca e4f5a6b por prefijo:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "ffff999") || len(e.opened) != 1 || e.opened[0].ProjectSpace != "PAG" {
		t.Errorf("el perfil se abre como desde el formulario: %+v\n%s", e.opened, out)
	}
	send(e.m, keyEsc)
	if out := view(e.m); !strings.Contains(out, "Inicio › Generar documento") || !strings.Contains(out, "OPS-7") {
		t.Errorf("esc vuelve al formulario con la tarea:\n%s", out)
	}
}

func TestBarGenerateWithoutTheTaskOpensTheForm(t *testing.T) {
	e := newGenerateApp(t, nil)
	runBar(e.m, "g -p facturacion")
	if out := view(e.m); len(e.m.stack) != 2 || !strings.Contains(out, "Inicio › Generar documento") || !strings.Contains(out, "facturacion") {
		t.Errorf("sin -i se abre el formulario con lo que se dio:\n%s", out)
	}
	runBar(e.m, "g -i OPS-7 -p facturacion -f e4f5")
	if out := view(e.m); !strings.Contains(out, "Commits de OPS-7") || !strings.Contains(out, "El proyecto no tiene frontend: se ignoró -f") {
		t.Errorf("un lado que el proyecto no tiene se explica:\n%s", out)
	}
	runBar(e.m, "g -i OPS-7 -p nada")
	if out := view(e.m); len(e.m.stack) != 1 || !strings.Contains(out, "✗ proyecto 'nada' no encontrado") {
		t.Errorf("un proyecto que no existe es un error en la barra, como en el modo comando:\n%s", out)
	}
}

func TestBarOpensQA(t *testing.T) {
	e := newQAApp(t, qaTracker("qa@example.test"))
	runBar(e.m, "qa -s 12 -m Pagos")
	if out := view(e.m); !strings.Contains(out, "Inicio › Consolidado QA › Evaluar") {
		t.Errorf("qa con sprint y módulo llega a la evaluación:\n%s", out)
	}
	runBar(e.m, "qa -s doce")
	if out := view(e.m); len(e.m.stack) != 2 || !strings.Contains(out, "el sprint debe ser un número positivo") {
		t.Errorf("un sprint inválido queda en el formulario:\n%s", out)
	}
}

func TestBarOpensExport(t *testing.T) {
	e := newExportApp(t, deployDoc, "Consolidado de Pruebas QA - Pagos - Sprint 12")
	runBar(e.m, "f -i ops-7")
	if out := view(e.m); !strings.Contains(out, "Inicio › Exportar página › Documentos de OPS-7") {
		t.Errorf("f -i llega a los documentos:\n%s", out)
	}
}

func TestBarOpensBacklog(t *testing.T) {
	e := newBacklogApp(t)
	runBar(e.m, "backlog scan -r backend --since 12w")
	if out := view(e.m); !strings.Contains(out, "› Analizar") || !strings.Contains(out, "backend · /repos/api") || strings.Contains(out, "/repos/web") {
		t.Errorf("backlog scan analiza con las opciones de la línea:\n%s", out)
	}
	if so := e.scanned["/repos/api"]; so.Since != "12 weeks ago" {
		t.Errorf("--since llega al análisis: %+v", so)
	}
	runBar(e.m, "backlog scan -r web")
	if out := view(e.m); len(e.m.stack) != 2 || !strings.Contains(out, `el repo debe ser backend o frontend, no "web"`) {
		t.Errorf("un repo inválido queda en el formulario, sin analizar otra cosa:\n%s", out)
	}
	runBar(e.m, "backlog scan -p nada")
	if out := view(e.m); len(e.m.stack) != 1 || !strings.Contains(out, "proyecto 'nada' no encontrado") {
		t.Errorf("un proyecto que no existe no analiza la carpeta actual:\n%s", out)
	}
	runBar(e.m, "backlog")
	if out := view(e.m); len(e.m.stack) != 2 || !strings.Contains(out, "Inicio › Backlog técnico") {
		t.Errorf("backlog sin scan abre el formulario, sin analizar:\n%s", out)
	}
}

func TestBarOpensProjectsAndSettings(t *testing.T) {
	e := newSettingsApp(t, savedConfig())
	cases := map[string]string{
		"project":     "Inicio › Proyectos",
		"project add": "Inicio › Proyectos › Nuevo proyecto",
		"init":        "Inicio › Configuración › Conexión",
		"profile ls":  "Inicio › Configuración › Perfiles",
		"profile add": "Inicio › Configuración › Perfiles › Nuevo perfil",
	}
	for line, want := range cases {
		runBar(e.m, line)
		if out := view(e.m); !strings.Contains(out, want) {
			t.Errorf("/%s abre %q:\n%s", line, want, out)
		}
	}
	if e.file.saves != 0 {
		t.Errorf("abrir pantallas no guarda nada: %d escrituras", e.file.saves)
	}
}

func TestBarExplainsWhatTheVisualModeDoesNotHave(t *testing.T) {
	m := newTestApp(t, testConfig(), nil, 100, 30)
	cases := map[string]string{
		"g -i OPS-7 --dry-run":           "--dry-run sobra en el modo visual",
		"g -i OPS-7 --template x":        "--template no está en el modo visual: usa draco generate en la terminal",
		"g -i OPS-7 --template x -s DEV": "--space no está en el modo visual",
		"backlog scan --json":            "--json no está en el modo visual",
		"update":                         "«update» se corre fuera del modo visual",
		"template":                       "«template» se corre fuera del modo visual",
	}
	for line, want := range cases {
		runBar(m, line)
		if out := view(m); len(m.stack) != 1 || !strings.Contains(out, want) {
			t.Errorf("/%s: esperaba %q sin abrir nada:\n%s", line, want, out)
		}
	}
}

func TestPanelExamplesWorkInTheBar(t *testing.T) {
	h := newHome(&state{cfg: &config.Config{}})
	for _, it := range h.items {
		l, err := readLine(routeSpecs, strings.TrimPrefix(it.example, "draco "))
		if err == nil {
			err = barUnsupported(l)
		}
		if err != nil {
			t.Errorf("el ejemplo de %s (%s) no funciona en la barra: %v", it.label, it.example, err)
		}
	}
}
