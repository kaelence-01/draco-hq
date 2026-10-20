package document

import (
	"fmt"
	"strings"
	"time"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

// QADoc holds the data of the QA consolidated report. Sprint > 0 is Sprint
// mode; otherwise Period labels the Kanban window.
type QADoc struct {
	Sprint int
	Period string // Kanban mode, e.g. "08/05 al 22/05/2026"
	Module string
	Tasks  []provider.QAIssue

	// Header names from qa_report; empty renders as "—".
	LiderTecnico string
	PMO          string
	QA           string

	Date time.Time // review date; zero is today
}

// Headings of the sections QA writes by hand; the destination keeps them on
// update.
const (
	QAObservationHeading    = "Observación"
	QAConsiderationsHeading = "Consideraciones"
)

// QATitlePrefix starts every QA consolidated report title.
const QATitlePrefix = "Consolidado de Pruebas QA"

// QAMarkdown builds the QA consolidated report.
func QAMarkdown(doc QADoc) string {
	periodHeader, periodValue := "Período", mdText(doc.Period)
	if doc.Sprint > 0 {
		periodHeader, periodValue = "Sprint", fmt.Sprintf("%d", doc.Sprint)
	}
	date := doc.Date
	if date.IsZero() {
		date = time.Now()
	}

	var b strings.Builder
	b.WriteString("## Información General\n\n|  |  |\n|---|---|\n")
	for _, row := range [][2]string{
		{"Fechas de Revisión", date.Format("02 Jan 2006")},
		{"Líder Técnico", orDash(doc.LiderTecnico)},
		{"PMO", orDash(doc.PMO)},
		{"QA", orDash(doc.QA)},
		{periodHeader, periodValue},
	} {
		fmt.Fprintf(&b, "| **%s** | %s |\n", row[0], row[1])
	}

	b.WriteString("\n## Tabla Consolidada\n\n")
	b.WriteString("| Tarea | Sin errores Codificación | Sin devolución desarrollo | Documentación requerida | Pull Request Aprobado | Observaciones | Tarea de Revisión |\n")
	b.WriteString("|---|---|---|---|---|---|---|\n")
	for _, t := range doc.Tasks {
		review := ""
		if t.ReviewTaskURL != "" {
			review = card(t.ReviewTaskURL)
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n",
			card(t.URL), check(!t.HasCodingErrors), check(!t.HasDevReturns),
			deployDocCheck(t.HasDeployDoc, t.DeployDocUnknown), check(t.PRMerged), mdText(t.Observations), review)
	}

	fmt.Fprintf(&b, "\n## Resumen\n\n| Total de Tareas |\n|---|\n| %d |\n\n", len(doc.Tasks))
	fmt.Fprintf(&b, "### %s\n\n", QAObservationHeading)
	fmt.Fprintf(&b, "### %s\n\n- \n", QAConsiderationsHeading)
	return b.String()
}

// BuildQATitle returns "Consolidado de Pruebas QA - FAC - Sprint 17".
func BuildQATitle(module string, sprint int) string {
	return fmt.Sprintf("%s - %s - Sprint %d", QATitlePrefix, module, sprint)
}

// BuildQAKanbanTitle returns "Consolidado de Pruebas QA - 08/05 al 22/05/2026".
func BuildQAKanbanTitle(period string) string {
	return fmt.Sprintf("%s - %s", QATitlePrefix, period)
}

func check(ok bool) string {
	if ok {
		return "✔"
	}
	return "✖"
}

// deployDocCheck renders an unverified lookup as "?", never as a failed check:
// the report is QA evidence.
func deployDocCheck(ok, unknown bool) string {
	if unknown {
		return "?"
	}
	return check(ok)
}
