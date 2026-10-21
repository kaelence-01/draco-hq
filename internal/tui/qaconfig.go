package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/kaelence-01/draco-hq/internal/config"
)

// qaConfig edits what the QA report needs: qa_email, the names of its header
// (qa_report) and the Jira instance mapping (qa_jira). With qa_jira empty,
// that section is not saved: QA is optional.
type qaConfig struct {
	parent  *settings
	email   string
	report  config.QAReportConfig
	project string
	// Lists are typed comma-separated.
	statuses, reviews       string
	testing, sprint, marker string
	fields                  config.QAJiraFields
	form                    *huh.Form
	err                     string
}

func newQAConfig(parent *settings) *qaConfig {
	q := &qaConfig{parent: parent}
	file, err := parent.st.deps.LoadFile()
	if err != nil {
		q.err, file = err.Error(), &config.Config{}
	}
	q.email = file.QAEmail
	if file.QAReport != nil {
		q.report = *file.QAReport
	}
	if j := file.QAJira; j != nil {
		q.project, q.testing, q.sprint, q.marker, q.fields = j.Project, j.TestingStatus, j.SprintName, j.ObservationMarker, j.Fields
		q.statuses, q.reviews = strings.Join(j.Statuses, ", "), strings.Join(j.ReviewPrefixes, ", ")
	}
	q.form = q.build()
	return q
}

func (q *qaConfig) build() *huh.Form {
	input := func(title, desc, placeholder string, v *string) *huh.Input {
		return huh.NewInput().Title(title).Description(desc).Placeholder(placeholder).Value(v)
	}
	return newPagedForm(
		huh.NewGroup(
			input("Email de la cuenta de QA", "Solo ella publica el consolidado (no es un permiso)", "qa@example.test", &q.email),
			input("Líder Técnico", "Nombres del encabezado del consolidado", "", &q.report.LiderTecnico),
			input("PMO", "", "", &q.report.PMO),
			input("QA", "", "", &q.report.QA),
		),
		huh.NewGroup(
			input("Proyecto de Jira", "Clave del proyecto de las tareas", "OPS", &q.project),
			input("Estados que ya llegaron a QA", "Id o nombre, separados por comas", "10001, En QA", &q.statuses),
			input("Estado de Testing", "Abre la ventana Kanban. Vacío: "+config.DefaultTestingStatus, "", &q.testing),
			input("Nombre del sprint en Jira", "Con {module} y {sprint}. Vacío: "+config.DefaultSprintName, "", &q.sprint),
		),
		huh.NewGroup(
			input("Campo de errores de codificación", "Id con la forma customfield_12345", "customfield_10001", &q.fields.CodingErrors),
			input("Campo de devoluciones a desarrollo", "", "customfield_10002", &q.fields.DevReturns),
			input("Campo del estado del PR", "Panel de desarrollo", "customfield_10003", &q.fields.DevStatus),
		),
		huh.NewGroup(
			input("Prefijos de las tareas de revisión", "Separados por comas. Vacío: "+strings.Join(config.DefaultReviewPrefixes, ", "), "", &q.reviews),
			input("Marcador de observación", "Inicio del comentario que se toma como observación. Vacío: "+config.DefaultObservationMarker, "", &q.marker),
		),
	)
}

func (q *qaConfig) title() string       { return "QA" }
func (q *qaConfig) keys() []key.Binding { return []key.Binding{nextFieldKey, prevFieldKey} }

func (q *qaConfig) captures(k tea.KeyPressMsg) bool { return !key.Matches(k, backKey) }

func (q *qaConfig) init() tea.Cmd {
	if q.form.State != huh.StateNormal {
		q.form = q.build()
	}
	return q.form.Init()
}

func (q *qaConfig) update(msg tea.Msg) (screen, tea.Cmd) {
	m, cmd := q.form.Update(msg)
	q.form = m.(*huh.Form)
	if q.form.State != huh.StateCompleted {
		return q, cmd
	}
	email, report, jira := strings.TrimSpace(q.email), q.trimmedReport(), q.jira()
	if !jira.Empty() {
		if err := jira.Validate(); err != nil {
			q.err = "No se guardó: " + err.Error()
			q.form = q.build()
			return q, q.form.Init()
		}
	}
	err := q.parent.st.save(func(c *config.Config) error {
		c.QAEmail, c.QAReport, c.QAJira = email, nil, nil
		if report != (config.QAReportConfig{}) {
			c.QAReport = &report
		}
		if !jira.Empty() {
			c.QAJira = jira
		}
		return nil
	})
	q.parent.done("QA guardado", err)
	return q, back
}

func (q *qaConfig) trimmedReport() config.QAReportConfig {
	r := q.report
	return config.QAReportConfig{LiderTecnico: strings.TrimSpace(r.LiderTecnico), PMO: strings.TrimSpace(r.PMO), QA: strings.TrimSpace(r.QA)}
}

func (q *qaConfig) jira() *config.QAJiraConfig {
	f := q.fields
	return &config.QAJiraConfig{
		Project:       strings.TrimSpace(q.project),
		Statuses:      config.SplitList(q.statuses),
		TestingStatus: strings.TrimSpace(q.testing),
		SprintName:    strings.TrimSpace(q.sprint),
		Fields: config.QAJiraFields{
			CodingErrors: strings.TrimSpace(f.CodingErrors), DevReturns: strings.TrimSpace(f.DevReturns), DevStatus: strings.TrimSpace(f.DevStatus),
		},
		ReviewPrefixes:    config.SplitList(q.reviews),
		ObservationMarker: strings.TrimSpace(q.marker),
	}
}

func (q *qaConfig) view(width, height int) string { return formView(q.form, q.err, width) }
