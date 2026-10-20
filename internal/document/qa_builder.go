package document

import (
	"fmt"
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

// qaView is what the QA template receives: every value is already escaped
// Markdown.
type qaView struct {
	Date, LiderTecnico, PMO, QA string
	PeriodLabel, PeriodValue    string
	Tasks                       []qaRow
	Total                       int
}

type qaRow struct {
	Task, Errors, Returns, DeployDoc, PR, Observations, Review string
}

// QA renders the QA consolidated report.
func QA(t *Template, doc QADoc) (string, error) {
	date := doc.Date
	if date.IsZero() {
		date = time.Now()
	}
	v := qaView{
		Date:         fecha(date),
		LiderTecnico: orDash(doc.LiderTecnico),
		PMO:          orDash(doc.PMO),
		QA:           orDash(doc.QA),
		PeriodLabel:  "Período",
		PeriodValue:  mdText(doc.Period),
		Total:        len(doc.Tasks),
	}
	if doc.Sprint > 0 {
		v.PeriodLabel, v.PeriodValue = "Sprint", fmt.Sprintf("%d", doc.Sprint)
	}
	for _, task := range doc.Tasks {
		row := qaRow{
			Task:         card(task.URL),
			Errors:       check(!task.HasCodingErrors),
			Returns:      check(!task.HasDevReturns),
			DeployDoc:    deployDocCheck(task.HasDeployDoc, task.DeployDocUnknown),
			PR:           check(task.PRMerged),
			Observations: mdText(task.Observations),
		}
		if task.ReviewTaskURL != "" {
			row.Review = card(task.ReviewTaskURL)
		}
		v.Tasks = append(v.Tasks, row)
	}
	return t.render(v)
}

// QATitle renders the report title: Sprint mode when sprint > 0, else the
// Kanban period.
func QATitle(t *Template, module string, sprint int, period string) (string, error) {
	return t.renderTitle(struct {
		Module, Period string
		Sprint         int
	}{module, period, sprint})
}

var meses = [...]string{"ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic"}

// fecha formats a date in Spanish: "21 sep 2026".
func fecha(t time.Time) string {
	return fmt.Sprintf("%02d %s %d", t.Day(), meses[t.Month()-1], t.Year())
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
