package cmd

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func run(args ...string) (code int, out, errOut string) {
	return runWith("", args...)
}

// runWith runs a command answering its prompts with input, one answer per line.
func runWith(input string, args ...string) (code int, out, errOut string) {
	var o, e bytes.Buffer
	code = New(strings.NewReader(input), &o, &e).Run(args)
	return code, o.String(), e.String()
}

func TestEveryCommandHasHelpAndSummary(t *testing.T) {
	for name, c := range commands {
		if c.help == nil || c.run == nil || c.summary == "" {
			t.Errorf("el comando %q necesita run, help y summary", name)
		}
	}
}

func TestGeneralHelp(t *testing.T) {
	for _, args := range [][]string{nil, {"help"}, {"--help"}, {"-h"}} {
		code, out, _ := run(args...)
		if code != exitOK || !strings.Contains(out, "Uso:") || !strings.Contains(out, "version") || !strings.Contains(out, "https://github.com/kaelence-01/draco-hq") || strings.Contains(out, "docs/roadmap.md") {
			t.Errorf("%v: código %d\n%s", args, code, out)
		}
	}
}

func TestCommandHelpNeverRuns(t *testing.T) {
	ran := false
	register(&command{run: func(*App, []string) error { ran = true; return nil }, help: func(a *App) { a.printf("ayuda de prueba\n") }, summary: "prueba"}, "prueba")
	defer delete(commands, "prueba")

	for _, args := range [][]string{{"help", "prueba"}, {"prueba", "-h"}, {"prueba", "--issue", "x", "--help"}} {
		code, out, _ := run(args...)
		if code != exitOK || out != "ayuda de prueba\n" {
			t.Errorf("%v: código %d, salida %q", args, code, out)
		}
	}
	if ran {
		t.Error("pedir ayuda nunca ejecuta el comando")
	}
	if code, _, _ := run("prueba", "--", "-h"); code != exitOK || !ran {
		t.Error("después de -- un -h es un argumento, no un pedido de ayuda")
	}
}

func TestVersion(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"--version"}, {"-v"}} {
		if code, out, _ := run(args...); code != exitOK || out != "draco dev\n" {
			t.Errorf("%v: %d %q", args, code, out)
		}
	}
}

func TestUsageErrorsExitWithTwo(t *testing.T) {
	cases := map[string][]string{
		"comando desconocido: nada":                      {"nada"},
		`no existe el comando "nada"`:                    {"help", "nada"},
		"flag desconocido: --x (ver draco help version)": {"version", "--x"},
	}
	for want, args := range cases {
		code, out, errOut := run(args...)
		if code != exitUsage || !strings.Contains(errOut, want) || out != "" {
			t.Errorf("%v: código %d, stderr %q, stdout %q", args, code, errOut, out)
		}
	}
}

func TestNoColorOutsideATerminal(t *testing.T) {
	_, out, errOut := run("nada")
	if strings.Contains(out+errOut, "\033[") {
		t.Errorf("sin terminal no debe haber códigos de color: %q", out+errOut)
	}
}

func TestNoArgumentsOpensTheVisualModeOnlyOnATerminal(t *testing.T) {
	prev := runTUI
	t.Cleanup(func() { runTUI = prev })
	opened := 0
	runTUI = func(*App) error { opened++; return nil }

	if code, out, _ := run(); code != exitOK || !strings.Contains(out, "Uso:") || opened != 0 {
		t.Errorf("sin terminal: código %d, TUI abierta %d veces\n%s", code, opened, out)
	}

	var o, e bytes.Buffer
	a := New(strings.NewReader(""), &o, &e)
	a.tty = true
	if code := a.Run(nil); code != exitOK || opened != 1 || o.Len() != 0 {
		t.Errorf("en una terminal: código %d, TUI abierta %d veces, salida %q", code, opened, o.String())
	}

	runTUI = func(*App) error { return errors.New("la terminal no responde") }
	e.Reset()
	if code := a.Run(nil); code != exitError || !strings.Contains(e.String(), "la terminal no responde") {
		t.Errorf("un error de la TUI sale con 1: código %d, %q", code, e.String())
	}
}
