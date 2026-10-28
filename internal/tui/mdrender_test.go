package tui

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// drawn is the Markdown as the preview shows it, without colors and without
// the spaces that pad each line.
func drawn(md string, width int) []string {
	lines := strings.Split(ansi.Strip(renderMarkdown(md, width)), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return lines
}

func TestRenderMarkdownByConstruction(t *testing.T) {
	cases := []struct {
		name, md string
		want     []string // lines that must be there, in this order
		not      []string // text that must not be shown
	}{
		{"títulos sin su marca", "# Documento\n\n## Sección\n\n### Parte\n", []string{"Documento", "", "Sección", "", "Parte"}, []string{"#"}},
		{"un # final es del título", "## Soporte C#\n", []string{"Soporte C#"}, nil},
		{"los comentarios de draco no se muestran", "<!-- draco:tabla anchos=1,3 -->\n| A | B |\n|---|---|\n| x | y |\n", []string{"A", "B"}, []string{"draco", "<!--", "anchos"}},
		{"encabezado vacío descartado", "|  |  |\n|---|---|\n| **Épica** | _Enlace a la épica_ |\n", []string{"Épica  Enlace a la épica"}, []string{"─", "|", "**", "_"}},
		{"marcas de la tabla", "| T | E | D | C |\n|---|---|---|---|\n| a | ✔ | ✖ | ? |\n", []string{"T  E  D  C", "a  ✓  ✗  ?"}, []string{"✔", "✖"}},
		{"viñetas dentro de una celda", "| Archivos |\n|---|\n| - a.php<br>- b.php |\n", []string{"Archivos", "· a.php", "· b.php"}, []string{"<br>", "- a.php"}},
		{"casillas y viñetas", "- [ ] uno\n- [x] dos\n- tres\n-\n", []string{"[ ] uno", "[x] dos", "· tres", "·"}, []string{"- ["}},
		{"negrita, cursiva, código y enlaces", "**negrita** y _cursiva_ y `código` y [texto](https://x.test/a) y <https://y.test>", []string{"negrita y cursiva y código y texto y https://y.test"}, []string{"**", "](", "<https"}},
		{"un guion bajo dentro de una palabra no es cursiva", "snake_case_name y _cursiva_", []string{"snake_case_name y cursiva"}, nil},
		{"los escapes se quitan", `a\|b \*literal\* 1\. dos`, []string{"a|b *literal* 1. dos"}, []string{`\`}},
		{"una barra escapada no parte la celda", "| A | B |\n|---|---|\n| a\\|b | c |\n", []string{"A    B", "a|b  c"}, []string{`\`}},
		{"un párrafo une sus líneas", "uno dos\ntres\n\ncuatro\n", []string{"uno dos tres", "", "cuatro"}, nil},
	}
	for _, c := range cases {
		out := strings.Join(drawn(c.md, 60), "\n")
		at := 0
		for _, want := range c.want {
			i := strings.Index(out[at:], want)
			if i < 0 {
				t.Errorf("%s: falta %q (en este orden) en:\n%s", c.name, want, out)
				break
			}
			at += i + len(want)
		}
		for _, not := range c.not {
			if strings.Contains(out, not) {
				t.Errorf("%s: sobra %q en:\n%s", c.name, not, out)
			}
		}
	}
}

func TestRenderMarkdownShowsWhatItDoesNotKnowAsItIs(t *testing.T) {
	unknown := []string{"> una cita", "```", "~~~", "1. primero", "2) segundo", "---", "    sangrado de código", "<div>html</div>", "| una barra sin tabla |"}
	for _, line := range unknown {
		got := drawn("antes\n\n"+line+"\n\ndespués\n", 60)
		if got[0] != "antes" || got[2] != line || got[4] != "después" {
			t.Errorf("%q se muestra tal cual, entre sus vecinas: %q", line, got)
		}
	}
}

func TestRenderMarkdownTableColumns(t *testing.T) {
	md := "<!-- draco:tabla anchos=1,3 -->\n| A | B |\n|---|---|\n| uno | dos tres cuatro cinco seis siete ocho |\n"
	lines := drawn(md, 40)
	if i := strings.Index(lines[0], "B"); i != 11 {
		t.Errorf("las columnas siguen las proporciones del comentario (9 y 29, con 2 de separación): la B empieza en %d:\n%s", i, strings.Join(lines, "\n"))
	}
	if !strings.HasPrefix(lines[1], "─") || len([]rune(lines[1])) != 40 {
		t.Errorf("una raya del ancho de la tabla separa el encabezado: %q", lines[1])
	}

	natural := drawn("| A | B |\n|---|---|\n| uno | dos |\n", 60)
	if natural[0] != "A    B" || natural[2] != "uno  dos" {
		t.Errorf("sin comentario, cada columna es tan ancha como su contenido: %q", natural)
	}
}

func TestRenderMarkdownBox(t *testing.T) {
	lines := drawn("<!-- draco:recuadro -->\n- [ ] uno\n- [x] dos más largo\n\nfuera\n", 30)
	if len(lines) != 6 || !strings.HasPrefix(lines[0], "╭") || !strings.HasPrefix(lines[1], "│ [ ] uno") || !strings.HasPrefix(lines[2], "│ [x] dos más largo │") ||
		!strings.HasPrefix(lines[3], "╰") || lines[4] != "" || lines[5] != "fuera" {
		t.Errorf("el recuadro envuelve solo su bloque:\n%s", strings.Join(lines, "\n"))
	}
	if ansi.StringWidth(lines[0]) != ansi.StringWidth(lines[2]) || ansi.StringWidth(lines[0]) != len([]rune("│ [x] dos más largo │")) {
		t.Errorf("el recuadro mide lo que su contenido, no el ancho de la pantalla:\n%s", strings.Join(lines, "\n"))
	}
	for _, l := range lines {
		if ansi.StringWidth(l) > 30 {
			t.Errorf("el recuadro cabe en el ancho: %q", l)
		}
	}
}

func TestRenderMarkdownLinkCells(t *testing.T) {
	md := "| Tarea | Ok |\n|---|---|\n| <https://example.test/browse/OPS-7> | ✔ |\n"
	if got := drawn(md, 60)[2]; !strings.Contains(got, "https://example.test/browse/OPS-7") {
		t.Errorf("si cabe, la URL se ve entera: %q", got)
	}
	got := strings.Join(drawn("<!-- draco:tabla anchos=1,1 -->\n"+md, 30), "\n")
	if !strings.Contains(got, "…/browse/OPS-7") || strings.Contains(got, "example.test") {
		t.Errorf("si no cabe, se ve su cola, que lleva la clave de la tarea:\n%s", got)
	}
}

func TestShortURL(t *testing.T) {
	for _, c := range []struct {
		url   string
		width int
		want  string
	}{
		{"https://example.test/browse/OPS-7", 40, "https://example.test/browse/OPS-7"},
		{"https://example.test/browse/OPS-7", 14, "…/browse/OPS-7"},
		{"https://example.test/browse/OPS-7", 9, "…/OPS-7"},
		{"https://example.test/browse/OPS-7", 5, "…PS-7"},
		{"https://example.test", 8, "…le.test"},
	} {
		if got := shortURL(c.url, c.width); got != c.want {
			t.Errorf("shortURL(%q, %d) = %q, se esperaba %q", c.url, c.width, got, c.want)
		}
	}
}

func TestColumnWidths(t *testing.T) {
	cases := []struct {
		name                    string
		weights, natural, lower []int
		avail                   int
		want                    []int
	}{
		{"cabe tal cual", nil, []int{5, 10}, []int{3, 4}, 40, []int{5, 10}},
		{"proporciones del comentario", []int{1, 3}, []int{3, 3}, []int{3, 3}, 40, []int{10, 30}},
		{"una palabra larga no se parte si hay sitio", []int{1, 9}, []int{12, 20}, []int{12, 5}, 50, []int{12, 38}},
		{"sin sitio, cede la que tiene de sobra", nil, []int{30, 10}, []int{8, 10}, 30, []int{20, 10}},
		{"sin sitio para las palabras, bajan al mínimo", nil, []int{30, 30}, []int{20, 20}, 12, []int{6, 6}},
	}
	for _, c := range cases {
		got := columnWidths(c.weights, c.natural, c.lower, c.avail)
		if len(got) != len(c.want) || got[0] != c.want[0] || got[1] != c.want[1] {
			t.Errorf("%s: %v, se esperaba %v", c.name, got, c.want)
		}
		if sum := sumOf(got); sum > c.avail && sum > minColumn*len(got) {
			t.Errorf("%s: las columnas suman %d y hay %d", c.name, sum, c.avail)
		}
	}
}

func TestRenderMarkdownOfRealDocumentsFits(t *testing.T) {
	for _, f := range []string{"../qa/testdata/draft-create-sprint.golden.md", "../deploy/testdata/draft-create.golden.md", "../document/testdata/deploy-completo.golden.md"} {
		md, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, width := range []int{56, 80, 100, 160} {
			out := renderMarkdown(string(md), width)
			for _, l := range strings.Split(out, "\n") {
				if w := ansi.StringWidth(l); w > width {
					t.Errorf("%s a %d columnas: una línea mide %d: %q", f, width, w, ansi.Strip(l))
				}
			}
			plain := ansi.Strip(out)
			for _, hidden := range []string{"<!--", "draco:", "|---|", "**"} {
				if strings.Contains(plain, hidden) {
					t.Errorf("%s a %d columnas: se ve %q:\n%s", f, width, hidden, plain)
				}
			}
		}
	}
}
