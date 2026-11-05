package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestPagerShowsTheFocusedPage(t *testing.T) {
	var a, b, c string
	hidden := true
	form, pages := newPagedForm(
		newPage("Uno", huh.NewInput().Value(&a)),
		newPage("Oculta", huh.NewInput().Value(&b)).hiddenWhen(func() bool { return hidden }),
		newPage("Tres", huh.NewInput().Value(&c)),
	)
	form.Init()
	if got := ansi.Strip(pages.line(form)); got != "●○  Página 1 de 2 · Uno" {
		t.Errorf("una página oculta no cuenta: %q", got)
	}
	hidden = false
	if got := ansi.Strip(pages.line(form)); got != "●○○  Página 1 de 3 · Uno" {
		t.Errorf("visible, sí cuenta: %q", got)
	}
	single, one := newPagedForm(newPage("Única", huh.NewInput().Value(&a)))
	single.Init()
	if one.line(single) != "" {
		t.Error("con una sola página no hay paginador")
	}
}

func TestQAFormShowsPagerAndFieldRules(t *testing.T) {
	e := newSettingsApp(t, savedConfig())
	e.toQA(t)
	out := view(e.m)
	if !strings.Contains(out, "●○○○○  Página 1 de 5 · Cuenta y encabezado") || !strings.Contains(out, "┄┄┄┄┄┄┄┄┄┄") {
		t.Fatalf("el formulario muestra la página y la línea bajo cada campo:\n%s", out)
	}
	e.fill("qa@example.test", "", "", "")
	if out := view(e.m); !strings.Contains(out, "○●○○○  Página 2 de 5 · Jira") {
		t.Errorf("al pasar de página el paginador avanza:\n%s", out)
	}
}

func TestTallFormScrollsToTheFocusedField(t *testing.T) {
	m, _ := newProjectsApp(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	typeText(m, "nreportes")
	for range 7 {
		send(m, keyEnter)
	}
	out := view(m)
	if !strings.Contains(out, "┃ Space de Confluence del proyecto") {
		t.Errorf("a 80x24 el último campo, con el foco, queda a la vista:\n%s", out)
	}
	for _, line := range strings.Split(out, "\n") {
		if ansi.StringWidth(line) > 80 {
			t.Errorf("ninguna línea pasa de 80 columnas: %q", line)
		}
	}
}
