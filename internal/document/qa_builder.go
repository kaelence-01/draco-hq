package document

import (
	"fmt"
	"strings"
	"time"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

// QADoc is the data of the QA report. Sprint 0 with Period set is Kanban mode.
type QADoc struct {
	Sprint  int
	Period  string // used in Kanban mode, e.g. "08/05 al 22/05/2026"
	Module  string
	Tasks   []provider.QAIssue // Tabla Consolidada (Testing / En Revisión)
	QATasks []provider.QAIssue // Resumen — tasks assigned to the QA user

	// Header names from qa_report; empty renders as "—".
	LiderTecnico string
	PMO          string
	QA           string

	// Preserved is zero for a new report.
	Preserved QAPreserved

	Date time.Time // review date; zero is today
}

const (
	qaSummaryHeading        = "Resumen"
	qaConsiderationsHeading = "Consideraciones"
)

// QAPreserved is what QA writes by hand: the summary's "Observación" cell and
// the "Consideraciones" section. An update replaces the whole body, so both are
// read back and emitted again. A nil field renders empty.
type QAPreserved struct {
	Observation    []any // content of the "Observación" cell
	Considerations []any // nodes under the "Consideraciones" heading
}

// ExtractQAPreserved reads them back; a nil field means not found or empty.
func ExtractQAPreserved(adf map[string]any) QAPreserved {
	return QAPreserved{
		Observation:    summaryObservation(adf),
		Considerations: ExtractSection(adf, qaConsiderationsHeading),
	}
}

// summaryObservation reads the second cell of the data row of the summary table.
func summaryObservation(adf map[string]any) []any {
	for _, raw := range ExtractSection(adf, qaSummaryHeading) {
		node, ok := raw.(map[string]any)
		if !ok || node["type"] != "table" {
			continue
		}
		rows, _ := node["content"].([]any)
		if len(rows) < 2 {
			return nil
		}
		row, _ := rows[1].(map[string]any)
		cells, _ := row["content"].([]any)
		if len(cells) < 2 {
			return nil
		}
		cell, _ := cells[1].(map[string]any)
		content, _ := cell["content"].([]any)
		return copyNodes(content)
	}
	return nil
}

// BuildQA builds the ADF of the QA report.
func BuildQA(doc QADoc) map[string]any {
	var periodHeader, periodValue string
	if doc.Sprint > 0 {
		periodHeader = "Sprint"
		periodValue = fmt.Sprintf("%d", doc.Sprint)
	} else {
		periodHeader = "Período"
		periodValue = doc.Period
	}
	return map[string]any{
		"type":    "doc",
		"version": 1,
		"content": append([]any{
			heading(2, "Información General"),
			qaInfoTable(periodHeader, periodValue, doc),
			heading(2, "Tabla Consolidada"),
			qaConsolidatedTable(doc.Tasks),
			heading(2, qaSummaryHeading),
			qaResumenTable(len(doc.Tasks), doc.Preserved.Observation),
			heading(3, qaConsiderationsHeading),
		}, qaConsiderations(doc.Preserved.Considerations)...),
	}
}

func qaConsiderations(preserved []any) []any {
	if len(preserved) > 0 {
		return preserved
	}
	return []any{qaEmptyBullet()}
}

// QATitlePrefix starts every QA consolidated report title.
const QATitlePrefix = "Consolidado de Pruebas QA"

// BuildQATitle returns "Consolidado de Pruebas QA - Pagos - Sprint 17".
func BuildQATitle(module string, sprint int) string {
	return fmt.Sprintf("%s - %s - Sprint %d", QATitlePrefix, module, sprint)
}

// BuildQAKanbanTitle returns "Consolidado de Pruebas QA - 08/05 al 22/05/2026".
func BuildQAKanbanTitle(period string) string {
	return fmt.Sprintf("%s - %s", QATitlePrefix, period)
}

func qaInfoTable(periodHeader, periodValue string, doc QADoc) map[string]any {
	date := doc.Date
	if date.IsZero() {
		date = time.Now()
	}
	today := date.Format("02 Jan 2006")
	return table("full-width", 760, []any{
		tableRow([]any{
			tableHeader(200, textNode("Fechas de Revisión")),
			tableCell(560, textNode(today)),
		}),
		tableRow([]any{
			tableHeader(200, textNode("Líder Técnico")),
			tableCell(560, textNode(orDash(doc.LiderTecnico))),
		}),
		tableRow([]any{
			tableHeader(200, textNode("PMO")),
			tableCell(560, textNode(orDash(doc.PMO))),
		}),
		tableRow([]any{
			tableHeader(200, textNode("QA")),
			tableCell(560, textNode(orDash(doc.QA))),
		}),
		tableRow([]any{
			tableHeader(200, textNode(periodHeader)),
			tableCell(560, textNode(periodValue)),
		}),
	})
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func qaConsolidatedTable(tasks []provider.QAIssue) map[string]any {
	rows := []any{
		tableRow([]any{
			tableHeader(150, textNode("Tarea")),
			tableHeader(120, textNode("Sin errores Codificación")),
			tableHeader(120, textNode("Sin devolución desarrollo")),
			tableHeader(120, textNode("Documentación requerida")),
			tableHeader(120, textNode("Pull Request Aprobado")),
			tableHeader(130, textNode("Observaciones")),
			tableHeader(130, textNode("Tarea de Revisión")),
		}),
	}
	for _, t := range tasks {
		var reviewCell map[string]any
		if t.ReviewTaskURL != "" {
			reviewCell = tableCell(130, inlineCard(t.ReviewTaskURL))
		} else {
			reviewCell = qaTableCellEmpty(130)
		}
		var obsCell map[string]any
		if t.Observations != "" {
			obsCell = tableCell(130, textNode(t.Observations))
		} else {
			obsCell = qaTableCellEmpty(130)
		}
		rows = append(rows, tableRow([]any{
			tableCell(150, inlineCard(t.URL)),
			tableCell(120, qaEmoji(!t.HasCodingErrors)),
			tableCell(120, qaEmoji(!t.HasDevReturns)),
			tableCell(120, qaCheck(t.HasDeployDoc, t.DeployDocUnknown)),
			tableCell(120, qaEmoji(t.PRMerged)),
			obsCell,
			reviewCell,
		}))
	}
	return table("full-width", 800, rows)
}

func qaResumenTable(totalTasks int, observation []any) map[string]any {
	obsCell := qaTableCellEmpty(560)
	if len(observation) > 0 {
		obsCell["content"] = observation
	}
	return table("full-width", 760, []any{
		tableRow([]any{
			tableHeader(200, textNode("Total de Tareas")),
			tableHeader(560, textNode("Observación")),
		}),
		tableRow([]any{
			tableCell(200, textNode(fmt.Sprintf("%d", totalTasks))),
			obsCell,
		}),
	})
}

// qaCheck renders an unknown answer as "?", never as a cross: the table is QA
// evidence. Plain text, so it renders with any emoji set.
func qaCheck(ok, unknown bool) map[string]any {
	if unknown {
		return textNode("?")
	}
	return qaEmoji(ok)
}

func qaEmoji(ok bool) map[string]any {
	if ok {
		return map[string]any{
			"type":  "emoji",
			"attrs": map[string]any{"shortName": ":check_mark:", "id": "atlassian-check_mark", "text": "✔"},
		}
	}
	return map[string]any{
		"type":  "emoji",
		"attrs": map[string]any{"shortName": ":cross_mark:", "id": "atlassian-cross_mark", "text": "✖"},
	}
}

func qaTableCellEmpty(colwidth int) map[string]any {
	return map[string]any{
		"type":    "tableCell",
		"attrs":   map[string]any{"colwidth": []int{colwidth}},
		"content": []any{emptyParagraph()},
	}
}

func qaEmptyBullet() map[string]any {
	return map[string]any{
		"type": "bulletList",
		"content": []any{
			map[string]any{
				"type":    "listItem",
				"content": []any{emptyParagraph()},
			},
		},
	}
}
