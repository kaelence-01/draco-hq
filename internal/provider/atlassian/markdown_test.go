package atlassian

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kaelence-01/draco-hq/internal/document"
	"github.com/kaelence-01/draco-hq/internal/golden"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

func content(t *testing.T, md string) []any {
	t.Helper()
	return MarkdownToADF(md)["content"].([]any)
}

func node(v any) map[string]any { return v.(map[string]any) }

func children(v any) []any { return node(v)["content"].([]any) }

func asJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func TestMarkdownHeadingAndMarks(t *testing.T) {
	got := content(t, "## Título\n\nHola **negrita** _cursiva_ `código` ~~tachado~~")
	if h := node(got[0]); h["type"] != "heading" || node(h["attrs"])["level"] != 2 {
		t.Errorf("título = %v", asJSON(h))
	}
	want := `[{"text":"Hola ","type":"text"},{"marks":[{"type":"strong"}],"text":"negrita","type":"text"},{"text":" ","type":"text"},{"marks":[{"type":"em"}],"text":"cursiva","type":"text"},{"text":" ","type":"text"},{"marks":[{"type":"code"}],"text":"código","type":"text"},{"text":" ","type":"text"},{"marks":[{"type":"strike"}],"text":"tachado","type":"text"}]`
	if got := asJSON(children(got[1])); got != want {
		t.Errorf("marcas:\n got  %s\n want %s", got, want)
	}
}

func TestMarkdownEscapesAndEntities(t *testing.T) {
	got := node(children(content(t, `Pagos \*urgente\* &amp; más &#241;`)[0])[0])["text"]
	if want := "Pagos *urgente* & más \u00f1"; got != want {
		t.Errorf("texto = %q, esperaba %q", got, want)
	}
}

func TestMarkdownLinksAndInlineCards(t *testing.T) {
	got := children(content(t, "[ver](https://example.test/a?x=1&y=2) <https://example.test/browse/OPS-7> [https://example.test/b](https://example.test/b)")[0])
	link := node(got[0])
	if href := node(node(link["marks"].([]any)[0])["attrs"])["href"]; link["text"] != "ver" || href != "https://example.test/a?x=1&y=2" {
		t.Errorf("enlace = %s", asJSON(link))
	}
	for _, i := range []int{2, 4} {
		if node(got[i])["type"] != "inlineCard" {
			t.Errorf("nodo %d debía ser tarjeta: %s", i, asJSON(got[i]))
		}
	}
}

func TestMarkdownEmoji(t *testing.T) {
	got := children(content(t, "ok ✔ / no ✖")[0])
	if len(got) != 4 || node(got[1])["type"] != "emoji" || node(got[3])["type"] != "emoji" {
		t.Fatalf("emoji = %s", asJSON(got))
	}
	if node(node(got[3])["attrs"])["shortName"] != ":cross_mark:" {
		t.Errorf("✖ debía ser :cross_mark:: %s", asJSON(got[3]))
	}
}

func TestMarkdownLists(t *testing.T) {
	got := content(t, "- uno\n- \n\n3. tres\n4. cuatro")
	bullet := node(got[0])
	if bullet["type"] != "bulletList" || len(children(bullet)) != 2 {
		t.Fatalf("viñetas = %s", asJSON(bullet))
	}
	if empty := children(children(bullet)[1]); asJSON(empty) != `[{"type":"paragraph"}]` {
		t.Errorf("un ítem vacío debe tener un párrafo vacío: %s", asJSON(empty))
	}
	if ol := node(got[1]); ol["type"] != "orderedList" || node(ol["attrs"])["order"] != 3 {
		t.Errorf("numerada = %s", asJSON(ol))
	}
}

func TestMarkdownTaskList(t *testing.T) {
	got := content(t, "- [ ] Respaldar base\n- [x] Desplegar **backend**\n\nTexto\n\n- [ ] Otro bloque")
	first := node(got[0])
	if first["type"] != "taskList" || node(first["attrs"])["localId"] != "tasklist-1" {
		t.Fatalf("lista de tareas = %s", asJSON(first))
	}
	items := children(first)
	if node(node(items[0])["attrs"])["state"] != "TODO" || node(node(items[1])["attrs"])["state"] != "DONE" {
		t.Errorf("estados = %s", asJSON(items))
	}
	if txt := asJSON(children(items[0])); txt != `[{"text":"Respaldar base","type":"text"}]` {
		t.Errorf("el contenido de la tarea es en línea, sin la casilla: %s", txt)
	}
	if id := node(node(children(got[2])[0])["attrs"])["localId"]; id != "task-2-1" {
		t.Errorf("los localId no deben repetirse entre listas: %v", id)
	}
}

func TestMarkdownTableConventions(t *testing.T) {
	md := "|  |  |\n|---|---|\n| **Épica** | _sin épica_ |\n| **Tarea(s)** | <https://example.test/browse/OPS-7> |\n\n" +
		"| Archivo | Enlace |\n|---|---|\n| a.php<br>b.php | [link](https://example.test/c) |\n"
	got := content(t, md)

	kv := children(got[0])
	if len(kv) != 2 {
		t.Fatalf("la fila de encabezado vacía debe omitirse: %s", asJSON(kv))
	}
	key := node(children(kv[0])[0])
	if key["type"] != "tableHeader" || asJSON(children(children(key)[0])) != `[{"text":"Épica","type":"text"}]` {
		t.Errorf("una celda en negrita es encabezado, sin la marca: %s", asJSON(key))
	}
	if node(children(kv[0])[1])["type"] != "tableCell" {
		t.Errorf("la celda del valor es normal: %s", asJSON(kv[0]))
	}

	files := children(got[1])
	if node(children(files[0])[0])["type"] != "tableHeader" {
		t.Errorf("una fila de encabezado con texto es encabezado: %s", asJSON(files[0]))
	}
	cell := asJSON(children(node(children(files[1])[0])))
	if cell != `[{"content":[{"text":"a.php","type":"text"},{"type":"hardBreak"},{"text":"b.php","type":"text"}],"type":"paragraph"}]` {
		t.Errorf("<br> en una celda es un salto de línea: %s", cell)
	}
	if layout := node(node(got[1])["attrs"])["layout"]; layout != "full-width" {
		t.Errorf("tablas a todo el ancho, obtuve %v", layout)
	}
}

func TestMarkdownRawHTMLIsLiteral(t *testing.T) {
	var texts []string
	for _, p := range content(t, "hola <script>alert(1)</script>\n\n<div>bloque</div>") {
		for _, n := range children(p) {
			if node(n)["type"] != "text" {
				t.Errorf("solo debía haber texto: %s", asJSON(n))
			}
			texts = append(texts, node(n)["text"].(string))
		}
	}
	if got := strings.Join(texts, ""); got != "hola <script>alert(1)</script><div>bloque</div>" {
		t.Errorf("el HTML debe quedar como texto literal: %q", got)
	}
}

func TestMarkdownCodeBlock(t *testing.T) {
	got := node(content(t, "```sql\nSELECT 1;\n```")[0])
	want := map[string]any{"type": "codeBlock", "attrs": map[string]any{"language": "sql"}, "content": []any{map[string]any{"type": "text", "text": "SELECT 1;\n"}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("bloque de código = %s", asJSON(got))
	}
}

func TestMarkdownEmptyDocument(t *testing.T) {
	if got := asJSON(MarkdownToADF("")); got != `{"content":[],"type":"doc","version":1}` {
		t.Errorf("documento vacío = %s", got)
	}
}

const deploySample = `|  |  |
|---|---|
| **Épica** | _Enlace a épica o función de Jira relacionada_ |
| **Tarea(s)** | <https://example.test/browse/OPS-7> |

## Arquitecturas e interfaces

### Proyectos y formularios - Backend:

| Servidor | Aplicación web | Ubicación | Nombre del archivo | Observación |
|---|---|---|---|---|
| [pagos-api](https://bitbucket.org/mi-org/pagos-api) | pagos-api | app/Http/Pagos | FiltroController.php<br>Rango.php | [link](https://bitbucket.org/mi-org/pagos-api/commits/a1b2c3d#chg-app/Http/Pagos/FiltroController.php)<br>[link](https://bitbucket.org/mi-org/pagos-api/commits/a1b2c3d#chg-app/Http/Pagos/Rango.php) |

## A considerar:

- [ ] Respaldar base de datos
- [ ] Desplegar backend
`

// TestGoldenPublishedADF fixes what Confluence receives for draco's real
// documents: the Markdown of internal/document through this converter.
func TestGoldenPublishedADF(t *testing.T) {
	deploy := document.DeployMarkdown(document.DeployDoc{
		IssueKey: "OPS-7", IssueURL: "https://example.test/browse/OPS-7",
		VCSHost: "https://bitbucket.org", VCSOrg: "mi-org",
		BackendRepo: "pagos-api", BackendCommit: "a1b2c3d",
		BackendFiles: map[string][]string{"app/Http/Pagos": {"FiltroController.php", "Rango.php"}},
		Checklist:    []string{"Respaldar base de datos", "Desplegar backend"},
	})
	golden.Assert(t, "adf-deploy", MarkdownToADF(deploy))

	qa := document.QAMarkdown(document.QADoc{
		Sprint: 12, Module: "Pagos", Date: time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC),
		Tasks: []provider.QAIssue{
			{URL: "https://example.test/browse/OPS-7", HasDeployDoc: true, PRMerged: true, ReviewTaskURL: "https://example.test/browse/OPS-20"},
			{URL: "https://example.test/browse/OPS-8", HasCodingErrors: true, DeployDocUnknown: true, Observations: "falta validar el monto"},
		},
	})
	golden.Assert(t, "adf-qa", MarkdownToADF(qa))
}

func FuzzMarkdownToADF(f *testing.F) {
	for _, seed := range []string{deploySample, "- [ ] a\n  - [x] b", "| a |\n|---|\n| **b** <br> ✔ |", "<https://x>", "> cita\n\n---\n\n```\ncódigo\n```"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, md string) {
		if _, err := json.Marshal(MarkdownToADF(md)); err != nil {
			t.Fatalf("ADF no serializable: %v", err)
		}
	})
}
