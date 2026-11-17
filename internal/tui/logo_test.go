package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestLogoFitsEightyColumns(t *testing.T) {
	if logoWidth+4 > 80 {
		t.Errorf("el logo mide %d columnas y con el margen no cabe en 80", logoWidth)
	}
}

func TestLogoBarIsAlignedAndAllCellsAreSingleWidth(t *testing.T) {
	col := -1
	for i, line := range logoLines {
		if w := len([]rune(line)); w != ansi.StringWidth(line) {
			t.Errorf("la línea %d del logo tiene símbolos que no miden una columna: %d runas, %d columnas", i+1, w, ansi.StringWidth(line))
		}
		idx := strings.IndexRune(line, '│')
		if idx < 0 {
			continue
		}
		c := ansi.StringWidth(line[:idx])
		if col >= 0 && c != col {
			t.Errorf("la barra de la línea %d está en la columna %d y la anterior en la %d", i+1, c, col)
		}
		col = c
	}
	if col < 0 {
		t.Error("el logo no tiene la barra entre el dragón y el nombre")
	}
}

func TestFrameShowsTheLogoOnEveryScreenWhenThereIsRoom(t *testing.T) {
	m := newTestApp(t, testConfig(), nil, 100, 40)
	send(m, openMsg{&fakeScreen{}})
	out := view(m)
	if strings.Count(out, "d r a c o") != 1 || !strings.Contains(out, "⣿") {
		t.Errorf("en una pantalla abierta el logo sale una vez, en la cabecera:\n%s", out)
	}
	send(m, backMsg{})
	if out := view(m); strings.Count(out, "d r a c o") != 1 {
		t.Errorf("en la principal el logo del marco no se repite:\n%s", out)
	}
}

func TestFrameKeepsTheBodyTheScreensWereDesignedFor(t *testing.T) {
	m := newTestApp(t, testConfig(), nil, 100, 29)
	if !m.hasRoomForBanner() {
		t.Fatal("con 29 líneas el logo cabe")
	}
	if lines := strings.Split(view(m), "\n"); len(lines) != 29 {
		t.Errorf("la pantalla ocupa %d líneas, se esperaban 29", len(lines))
	}
	body := 29 - len(logoLines) - 1 - footerLines
	if body < minBody {
		t.Errorf("con el logo el cuerpo queda en %d líneas y las pantallas necesitan %d", body, minBody)
	}
	for _, size := range [][2]int{{100, 28}, {80, 24}, {logoWidth + bannerInfoWidth - 1, 40}} {
		m := newTestApp(t, testConfig(), nil, size[0], size[1])
		send(m, openMsg{&fakeScreen{}})
		if lines := strings.Split(view(m), "\n"); m.hasRoomForBanner() || !strings.HasPrefix(lines[1], "──") {
			t.Errorf("%dx%d: sin espacio la cabecera del marco es una línea, sin logo:\n%s", size[0], size[1], strings.Join(lines[:3], "\n"))
		}
	}
}

func TestLogoShowsDragonAndNameOrFallsBackToName(t *testing.T) {
	wide := ansi.Strip(logo(logoWidth + 4))
	if !strings.Contains(wide, "d r a c o") || !strings.Contains(wide, "⣿") || len(strings.Split(wide, "\n")) != len(logoLines) {
		t.Errorf("con ancho suficiente el logo lleva el dragón y el nombre:\n%s", wide)
	}
	if narrow := ansi.Strip(logo(logoWidth + 3)); narrow != "draco" {
		t.Errorf("sin ancho suficiente va solo el nombre, se obtuvo %q", narrow)
	}
}
