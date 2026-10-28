package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestColumns(t *testing.T) {
	cases := []struct {
		width, gap, a, b int
		left, right      int
	}{
		{100, 2, 6, 4, 58, 40},
		{160, 2, 5, 4, 87, 71},
		{96, 0, 4, 6, 38, 58},
		{3, 5, 6, 4, 0, 0},
	}
	for _, c := range cases {
		left, right := columns(c.width, c.gap, c.a, c.b)
		if left != c.left || right != c.right {
			t.Errorf("columns(%d, %d, %d, %d) = %d, %d; se esperaba %d, %d", c.width, c.gap, c.a, c.b, left, right, c.left, c.right)
		}
		if left+right+c.gap > max(c.width, c.gap) {
			t.Errorf("las columnas y el espacio entre ellas pasan de %d", c.width)
		}
	}
}

func TestStepper(t *testing.T) {
	steps := []string{"Tarea", "Commits", "Vista previa"}
	for current, want := range map[int]string{
		0: "Tarea  ›  Commits  ›  Vista previa",
		1: "✓ Tarea  ›  Commits  ›  Vista previa",
		2: "✓ Tarea  ›  ✓ Commits  ›  Vista previa",
	} {
		if got := ansi.Strip(stepper(steps, current)); got != want {
			t.Errorf("paso %d: %q, se esperaba %q", current, got, want)
		}
	}
}

func TestTitleLine(t *testing.T) {
	steps := []string{"Tarea", "Commits"}
	wide := ansi.Strip(titleLine("OPS-7", steps, 1, 100))
	if !strings.HasPrefix(wide, "OPS-7 ") || !strings.HasSuffix(wide, "✓ Tarea  ›  Commits") || ansi.StringWidth(wide) != 100 {
		t.Errorf("desde 96 columnas, el título a la izquierda y los pasos al final de la línea: %q", wide)
	}
	if got := ansi.Strip(titleLine("OPS-7", steps, 1, 95)); got != "OPS-7" {
		t.Errorf("debajo de 96 columnas solo va el título: %q", got)
	}
	if got := ansi.Strip(titleLine(strings.Repeat("x", 90), steps, 1, 100)); strings.Contains(got, "Tarea") {
		t.Errorf("si no caben juntos, los pasos ceden: %q", got)
	}
}

func TestStretchFillsTheHeight(t *testing.T) {
	out := ansi.Strip(stretch("uno\ndos", 20, 8))
	lines := strings.Split(out, "\n")
	if len(lines) != 8 || ansi.StringWidth(lines[0]) != 20 || !strings.HasPrefix(lines[len(lines)-1], "╰") {
		t.Errorf("la caja mide 20x8 y su borde llega abajo:\n%s", out)
	}
}

func TestHangWrapsUnderTheText(t *testing.T) {
	lines := strings.Split(ansi.Strip(hang("Ruta   ", "uno dos tres cuatro", 16)), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	if got, want := strings.Join(lines, "|"), "Ruta   uno dos|       tres|       cuatro"; got != want {
		t.Errorf("el texto sigue bajo sí mismo, no bajo la etiqueta:\n%q\nse esperaba\n%q", got, want)
	}
}
