package tui

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kaelence-01/draco-hq/internal/cli"
	"github.com/kaelence-01/draco-hq/internal/config"
)

var barSpecs = []cli.Command{
	{Names: []string{"backlog"}, Summary: "Deuda técnica", Subcommands: []cli.Command{{Names: []string{"scan"}, Summary: "Analiza", Flags: []cli.Flag{
		{Long: "--repo", Short: "-r", Value: true, Desc: "Solo ese repo"},
		{Long: "--output", Short: "-o", Value: true, Optional: true, Desc: "Guarda el reporte"},
		{Long: "--deps", Desc: "Paquetes vulnerables"},
	}}}},
	{Names: []string{"generate", "gen", "g"}, Summary: "Documento de despliegue", Flags: []cli.Flag{
		{Long: "--issue", Short: "-i", Value: true, Desc: "Clave de la tarea"},
		{Long: "--commit-backend", Short: "-b", Value: true, Desc: "Commits del backend"},
		{Long: "--project", Short: "-p", Value: true, Desc: "Proyecto"},
		{Long: "--profile", Value: true, Desc: "Perfil"},
		{Long: "--dry-run", Desc: "Sin publicar"},
	}},
	{Names: []string{"qa"}, Summary: "Consolidado QA", Flags: []cli.Flag{{Long: "--sprint", Short: "-s", Value: true, Desc: "Sprint"}}},
	{Names: []string{"version"}, Summary: "Versión"},
}

func barConfig() *config.Config {
	return &config.Config{
		Projects: map[string]*config.ProjectConfig{"pagos": {}, "facturacion": {}},
		Profiles: map[string]*config.Profile{"equipo": {}, "externo": {}},
	}
}

func texts(s []suggestion) []string {
	var out []string
	for _, x := range s {
		out = append(out, x.text)
	}
	return out
}

func TestBarWords(t *testing.T) {
	cases := []struct {
		line   string
		done   []string
		typing string
	}{
		{"", nil, ""},
		{"g", nil, "g"},
		{"/g -i ", []string{"g", "-i"}, ""},
		{"draco g -i OPS", []string{"g", "-i"}, "OPS"},
		{"draco", nil, "draco"},
	}
	for _, c := range cases {
		done, typing := barWords(c.line)
		if len(done) == 0 {
			done = nil
		}
		if !reflect.DeepEqual(done, c.done) || typing != c.typing {
			t.Errorf("barWords(%q) = %q %q", c.line, done, typing)
		}
	}
}

func TestSuggest(t *testing.T) {
	cmds := append([]cli.Command{helpCommand}, barSpecs...)
	cases := []struct {
		line string
		want []string
	}{
		{"", []string{"help", "backlog", "g", "qa", "version"}},
		{"ge", []string{"gen"}},
		{"g ", []string{"-i", "-b", "-p", "--profile", "--dry-run"}},
		{"g --pro", []string{"--project", "--profile"}},
		{"g -i OPS-7 ", []string{"-b", "-p", "--profile", "--dry-run"}},
		{"g -i=OPS-7 -", []string{"-b", "-p", "--profile", "--dry-run"}},
		{"g -p ", []string{"facturacion", "pagos"}},
		{"draco g -p pa", []string{"pagos"}},
		{"g --profile ", []string{"equipo", "externo"}},
		{"g -i ", nil},
		{"g -i -", []string{"-b", "-p", "--profile", "--dry-run"}},
		{"g OPS", nil},
		{"backlog ", []string{"scan"}},
		{"backlog scan -r ", []string{"backend", "frontend"}},
		{"backlog scan -o ", []string{"-r", "--deps"}},
		{"help ", []string{"backlog", "g", "qa", "version"}},
		{"nada ", nil},
	}
	for _, c := range cases {
		if got := texts(suggest(cmds, barConfig(), c.line)); !reflect.DeepEqual(got, c.want) {
			t.Errorf("suggest(%q) = %q, esperaba %q", c.line, got, c.want)
		}
	}
	if s := suggest(cmds, barConfig(), "g"); len(s) != 1 || s[0].label != "generate, gen, g" || s[0].desc != "Documento de despliegue" {
		t.Errorf("la sugerencia de un comando muestra sus nombres y su resumen: %+v", s)
	}
	if s := suggest(cmds, nil, "g -p "); s != nil {
		t.Errorf("sin config no hay valores que sugerir: %+v", s)
	}
}

func TestComplete(t *testing.T) {
	cases := []struct{ line, text, want string }{
		{"", "g", "g "},
		{"g --pro", "--project", "g --project "},
		{"g -p ", "pagos", "g -p pagos "},
	}
	for _, c := range cases {
		if got := complete(c.line, suggestion{text: c.text}); got != c.want {
			t.Errorf("complete(%q, %q) = %q", c.line, c.text, got)
		}
	}
}

func TestReadLine(t *testing.T) {
	l, err := readLine(barSpecs, "/draco qa -s 12")
	if err != nil || l.cmd.Names[0] != "qa" || l.flags.Get("--sprint") != "12" {
		t.Errorf("se lee como en el modo comando: %+v %v", l, err)
	}
	if l, err := readLine(barSpecs, "backlog scan -r backend"); err != nil || l.sub.Names[0] != "scan" || l.flags.Get("--repo") != "backend" {
		t.Errorf("con subcomando: %+v %v", l, err)
	}
	if l, err := readLine(barSpecs, "  "); err != nil || l.cmd != nil {
		t.Errorf("una línea vacía no es un comando: %+v %v", l, err)
	}
	for line, want := range map[string]string{
		"g --x":        "flag desconocido: --x",
		"g -i":         "falta el valor de -i",
		"nada":         "comando desconocido: nada",
		"backlog nada": "subcomando desconocido: backlog nada",
	} {
		if _, err := readLine(barSpecs, line); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("readLine(%q): esperaba %q, obtuve %v", line, want, err)
		}
	}
}

func newBarApp(t *testing.T) *app {
	t.Helper()
	m := newApp(Deps{
		LoadConfig: func() (*config.Config, error) { return testConfig(), nil },
		Version:    "v9.9.9",
		Commands:   barSpecs,
		Help: func(name string) (string, error) {
			if name == "generate" {
				return "draco generate — Documento de despliegue\n\nUso:\n  draco g -i <tarea> -b <commit>", nil
			}
			return "", errors.New("sin ayuda")
		},
	})
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return m
}

func TestBarSuggestsAndCompletes(t *testing.T) {
	m := newBarApp(t)
	if out := view(m); !strings.Contains(out, "/ comando (/help: ayuda)") {
		t.Fatalf("el pie ofrece la barra:\n%s", out)
	}
	typeText(m, "/")
	out := view(m)
	for _, want := range []string{"/█", "▸ help", "generate, gen, g", "Documento de despliegue", "tab completar", "esc cerrar"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q con la barra abierta:\n%s", want, out)
		}
	}
	typeText(m, "g --pro")
	send(m, keyDown)
	send(m, tea.KeyPressMsg{Code: tea.KeyTab})
	if out := view(m); !strings.Contains(out, "/g --profile █") || !strings.Contains(out, "equipo") {
		t.Errorf("tab completa la sugerencia elegida y siguen los valores:\n%s", out)
	}
	typeText(m, "q?")
	if m.help.ShowAll || len(m.stack) != 1 {
		t.Error("con la barra abierta, q y ? se escriben")
	}
	send(m, keyEsc)
	if out := view(m); strings.Contains(out, "/g --profile") || len(m.stack) != 1 {
		t.Errorf("esc cierra la barra sin salir de la principal:\n%s", out)
	}
}

func TestBarRunsHelpAndExplainsTheRest(t *testing.T) {
	m := newBarApp(t)
	typeText(m, "/g --x")
	send(m, keyEnter)
	if out := view(m); !strings.Contains(out, "✗ flag desconocido: --x") {
		t.Errorf("un flag desconocido es un error, como en el modo comando:\n%s", out)
	}
	for range len("g --x") {
		send(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	typeText(m, "version")
	send(m, keyEnter)
	if out := view(m); !strings.Contains(out, "draco v9.9.9") {
		t.Errorf("version responde en la barra:\n%s", out)
	}
	for range len("version") {
		send(m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	typeText(m, "help g")
	send(m, keyEnter)
	out := view(m)
	if !strings.Contains(out, "Inicio › Ayuda de generate") || !strings.Contains(out, "draco g -i <tarea> -b <commit>") {
		t.Fatalf("/help g muestra la ayuda del comando:\n%s", out)
	}
	send(m, keyEsc)
	if out := view(m); strings.Contains(out, "/help g") || strings.Contains(out, "/█") || len(m.stack) != 1 {
		t.Errorf("al volver, la barra está cerrada:\n%s", out)
	}
	typeText(m, "/help")
	send(m, keyEnter)
	out = view(m)
	for _, want := range []string{"Inicio › Ayuda", "Barra de comandos", "Comandos", "generate, gen, g", "help"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en la ayuda general:\n%s", want, out)
		}
	}
	send(m, keyEsc)
	typeText(m, "/g -i OPS-7")
	send(m, keyEnter)
	if out := view(m); !strings.Contains(out, "todavía no abre «generate»") {
		t.Errorf("un comando válido sin pantalla desde la barra lo explica:\n%s", out)
	}
}

func TestBarFitsIn80By24(t *testing.T) {
	m := newBarApp(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	typeText(m, "/g ")
	typeText(m, "--x")
	send(m, keyEnter)
	out := view(m)
	lines := strings.Split(out, "\n")
	if len(lines) != 24 || !strings.Contains(out, "/g --x█") || !strings.Contains(out, "✗ flag desconocido: --x") || !strings.Contains(out, "tab completar") {
		t.Errorf("en 80×24, la línea, el error y el pie caben (%d líneas):\n%s", len(lines), out)
	}
	for _, l := range lines {
		if w := len([]rune(l)); w > 80 {
			t.Errorf("una línea mide %d columnas: %q", w, l)
		}
	}
}

func TestWithoutCommandsThereIsNoBar(t *testing.T) {
	m := newTestApp(t, testConfig(), nil, 100, 30)
	typeText(m, "/")
	if out := view(m); strings.Contains(out, "/█") || strings.Contains(out, "/ comando") {
		t.Errorf("sin descripción de comandos no hay barra:\n%s", out)
	}
}
