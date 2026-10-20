package atlassian

import (
	"encoding/json"
	"strings"
	"testing"
)

// roundTrip passes a document through JSON, as it comes back from Confluence:
// numbers become float64.
func roundTrip(t *testing.T, doc map[string]any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

const sample = "## Arquitecturas\n\ntabla\n\n## A Considerar\n\n- [x] Reiniciar colas\n\n### Nota interna\n\nsigue en la sección\n\n## Otra\n\nfuera\n"

func TestExtractSectionStopsAtSameRankAndKeepsDeeperHeadings(t *testing.T) {
	nodes, ok := extractSection(roundTrip(t, MarkdownToADF(sample)), "a considerar:")
	got := asJSON(nodes)
	if !ok || !strings.Contains(got, `"state":"DONE"`) || !strings.Contains(got, "sigue en la sección") || strings.Contains(got, "fuera") {
		t.Errorf("sección = %v %s", ok, got)
	}
}

func TestExtractSectionFoundButEmpty(t *testing.T) {
	nodes, ok := extractSection(MarkdownToADF("### Observación\n\n### Consideraciones\n\n- \n"), "Observación")
	if !ok || nodes != nil {
		t.Errorf("un título sin contenido existe y está vacío: %v %v", ok, nodes)
	}
	if _, ok := extractSection(MarkdownToADF("## Otra"), "Observación"); ok {
		t.Error("un título ausente no se encuentra")
	}
}

func TestReplaceSection(t *testing.T) {
	adf := MarkdownToADF(sample)
	if !replaceSection(adf, "A considerar", []any{map[string]any{"type": "rule"}}) {
		t.Fatal("el título existe")
	}
	got := asJSON(adf)
	if strings.Contains(got, "Reiniciar colas") || !strings.Contains(got, `{"type":"rule"}`) || !strings.Contains(got, "fuera") {
		t.Errorf("reemplazo = %s", got)
	}
	if replaceSection(adf, "Inexistente", nil) {
		t.Error("sin el título no hay reemplazo")
	}
}
