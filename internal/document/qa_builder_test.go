package document

import (
	"testing"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

func qaReport(preserved QAPreserved) map[string]any {
	return BuildQA(QADoc{
		Sprint:    12,
		Module:    "Pagos",
		Tasks:     []provider.QAIssue{{Key: "OPS-7", URL: "https://example.test/browse/OPS-7"}},
		Preserved: preserved,
	})
}

func textOf(nodes []any) string {
	var s string
	for _, raw := range nodes {
		if n, ok := raw.(map[string]any); ok {
			s += nodeText(n)
		}
	}
	return s
}

func TestQAPreservedSurvivesRegeneration(t *testing.T) {
	published := roundTrip(t, qaReport(QAPreserved{
		Observation: []any{paragraph(textNode("Sin bloqueos"))},
		Considerations: []any{
			bulletItem(textNode("Reprobar OPS-7 en staging")),
			paragraph(textNode("Ventana de despliegue el viernes")),
		},
	}))

	got := ExtractQAPreserved(published)
	if txt := textOf(got.Observation); txt != "Sin bloqueos" {
		t.Errorf("observación = %q, esperaba %q", txt, "Sin bloqueos")
	}
	if len(got.Considerations) != 2 {
		t.Fatalf("esperaba 2 nodos en Consideraciones, obtuve %d", len(got.Considerations))
	}

	again := ExtractQAPreserved(roundTrip(t, qaReport(got)))
	if textOf(again.Observation) != "Sin bloqueos" || len(again.Considerations) != 2 {
		t.Errorf("la segunda regeneración perdió contenido: %#v", again)
	}
}

func TestQAPreservedOfNewReportIsTemplate(t *testing.T) {
	got := ExtractQAPreserved(roundTrip(t, qaReport(QAPreserved{})))
	if got.Observation == nil {
		t.Error("la celda vacía de un reporte nuevo debe encontrarse")
	}
	if textOf(got.Observation) != "" {
		t.Errorf("la observación de un reporte nuevo debe estar vacía: %q", textOf(got.Observation))
	}
	if len(got.Considerations) != 1 {
		t.Errorf("esperaba la viñeta vacía de la plantilla, obtuve %d nodos", len(got.Considerations))
	}
}

func TestQAPreservedMissingParts(t *testing.T) {
	// A page whose summary table was deleted by hand.
	doc := roundTrip(t, map[string]any{
		"type":    "doc",
		"version": 1,
		"content": []any{heading(2, "Resumen"), paragraph(textNode("texto suelto"))},
	})
	got := ExtractQAPreserved(doc)
	if got.Observation != nil || got.Considerations != nil {
		t.Errorf("sin tabla ni Consideraciones esperaba nil, obtuve %#v", got)
	}
}
