// Package qa is the QA consolidated report workflow, with no terminal I/O.
// Steps: CheckAccount, Collect, EvaluateTask per task, FindExisting, DraftCreate
// or DraftUpdate, Locations (new page only) and Publish.
package qa

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kaelence-01/draco-hq/internal/atlassian"
	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/document"
)

// Atlassian is the part of the Jira and Confluence client the workflow uses.
type Atlassian interface {
	Whoami() (*atlassian.JiraUser, error)
	GetQATasksForReview(s atlassian.QASchema, sprintName, module string) ([]atlassian.QAIssue, error)
	GetQATasksAsAssignee(s atlassian.QASchema, sprintName, qaEmail string) ([]atlassian.QAIssue, error)
	GetQATasksForReviewKanban(s atlassian.QASchema, sinceDate string) ([]atlassian.QAIssue, error)
	GetQATasksAsAssigneeKanban(s atlassian.QASchema, sinceDate, qaEmail string) ([]atlassian.QAIssue, error)
	HasDeployDocLink(issueKey string) (bool, error)
	GetNovedadComment(issueKey string) (string, error)
	FindPageByTitle(title, spaceKey string) (*atlassian.Page, error)
	GetPageADF(pageID string) (*atlassian.PageADF, error)
	FindQAPagesForModule(module, spaceKey string) ([]atlassian.Page, error)
	GetPage(pageID string) (*atlassian.Page, error)
	CreatePage(spaceID, parentID, title string, adfBody map[string]any) (*atlassian.Page, error)
	UpdatePage(pageID, title string, currentVersion int, adfBody map[string]any) (*atlassian.Page, error)
}

var _ Atlassian = (*atlassian.Client)(nil)

var (
	// ErrNoSpace means no space key came from the request or the global config.
	ErrNoSpace = errors.New("Confluence space key no configurado")

	// ErrNotQAAccount means the account is not qa_email, or qa_email is unset;
	// Session.Account holds the authenticated user.
	ErrNotQAAccount = errors.New("este comando está pensado para la cuenta de QA y no coincide con la tuya")

	// ErrNoLocations means no previous consolidated report exists to sit next to.
	ErrNoLocations = errors.New("no se encontró ningún consolidado QA previo. Crea uno manualmente primero como referencia")
)

// Request is what the user asks for, before applying config.
type Request struct {
	Sprint int    // 0: Kanban mode
	Module string // required in Sprint mode
	Space  string // overrides the global space key
}

// Target is a Request resolved against config.
type Target struct {
	Kanban bool

	// Sprint mode.
	Sprint     int
	Module     string
	SprintName string // the sprint as named in Jira (qa_jira.sprint_name)

	// Kanban mode.
	Since  string // start of the window, YYYY-MM-DD
	Period string // display label of the window

	Title    string // deterministic; Confluence's uniqueness key
	SpaceKey string
	Schema   atlassian.QASchema
	QAEmail  string
	Report   config.QAReportConfig

	// Warnings are degraded but workable situations to show the user.
	Warnings []string
}

// Resolve applies config to the request; now sets the Kanban window. The space
// key comes from the request or the global config, not from a project.
func Resolve(cfg *config.Config, req Request, now time.Time) (*Target, error) {
	if req.Sprint < 0 {
		return nil, fmt.Errorf("el sprint debe ser un número positivo: %d", req.Sprint)
	}
	t := &Target{Kanban: req.Sprint == 0, QAEmail: cfg.QAEmail}
	module := strings.TrimSpace(req.Module)
	if !t.Kanban && module == "" {
		return nil, fmt.Errorf("falta el módulo: es obligatorio en modo Sprint")
	}

	if err := cfg.QAJira.Validate(); err != nil {
		return nil, err
	}
	t.Schema = schema(cfg.QAJira)

	if t.Kanban {
		since, period := atlassian.KanbanWindow(now)
		t.Since, t.Period = since.Format("2006-01-02"), period
		t.Title = document.BuildQAKanbanTitle(period)
		if module != "" {
			t.warn("el módulo no aplica en modo Kanban: se incluyen las tareas de todos los módulos.")
		}
	} else {
		t.Sprint, t.Module = req.Sprint, module
		t.SprintName = cfg.QAJira.SprintNameFor(module, req.Sprint)
		t.Title = document.BuildQATitle(module, req.Sprint)
	}

	t.SpaceKey = req.Space
	if t.SpaceKey == "" {
		t.SpaceKey = cfg.ConfluenceSpaceKey
	}
	if t.SpaceKey == "" {
		return nil, ErrNoSpace
	}

	if cfg.QAReport != nil {
		t.Report = *cfg.QAReport
	} else {
		t.warn("no hay 'qa_report' en la configuración; Líder Técnico, PMO y QA saldrán vacíos.")
	}
	return t, nil
}

func (t *Target) warn(msg string) { t.Warnings = append(t.Warnings, msg) }

// schema converts a validated qa_jira section to what the client expects.
func schema(q *config.QAJiraConfig) atlassian.QASchema {
	testing := q.TestingStatus
	if strings.TrimSpace(testing) == "" {
		testing = config.DefaultTestingStatus
	}
	return atlassian.QASchema{
		Project:           q.Project,
		Statuses:          q.Statuses,
		TestingStatus:     testing,
		FieldCodingErrors: q.Fields.CodingErrors,
		FieldDevReturns:   q.Fields.DevReturns,
		FieldDevStatus:    q.Fields.DevStatus,
	}
}

// Session is one run of the workflow; the steps fill its exported fields.
type Session struct {
	api Atlassian

	Target *Target

	Account  *atlassian.JiraUser
	Tasks    []atlassian.QAIssue // under review: the rows of the report
	QATasks  []atlassian.QAIssue // assigned to the QA account
	Existing *atlassian.Page
}

// NewSession starts a run for an already resolved target.
func NewSession(api Atlassian, t *Target) *Session {
	return &Session{api: api, Target: t}
}

// CheckAccount verifies that the authenticated account is qa_email. It guards
// against accidents, not access: authorization is Atlassian's permissions.
func (s *Session) CheckAccount() error {
	me, err := s.api.Whoami()
	if err != nil {
		return err
	}
	s.Account = me
	if s.Target.QAEmail == "" || !strings.EqualFold(me.EmailAddress, s.Target.QAEmail) {
		return ErrNotQAAccount
	}
	return nil
}

// Collect reads the tasks under review and the QA account's own tasks, and
// links each task to its review task.
func (s *Session) Collect() error {
	t := s.Target
	var err error
	if t.Kanban {
		if s.Tasks, err = s.api.GetQATasksForReviewKanban(t.Schema, t.Since); err != nil {
			return err
		}
		s.QATasks, err = s.api.GetQATasksAsAssigneeKanban(t.Schema, t.Since, t.QAEmail)
	} else {
		if s.Tasks, err = s.api.GetQATasksForReview(t.Schema, t.SprintName, t.Module); err != nil {
			return err
		}
		s.QATasks, err = s.api.GetQATasksAsAssignee(t.Schema, t.SprintName, t.QAEmail)
	}
	if err != nil {
		return err
	}

	reviews := atlassian.BuildReviewMap(s.QATasks)
	for i := range s.Tasks {
		if r, ok := reviews[s.Tasks[i].Key]; ok {
			s.Tasks[i].ReviewTaskKey = r.Key
			s.Tasks[i].ReviewTaskURL = r.URL
		}
	}
	return nil
}

// EvaluateTask checks the deploy-document link and observations of s.Tasks[i].
// A failed lookup marks the task unknown, never failed, and returns a warning.
func (s *Session) EvaluateTask(i int) []string {
	t := &s.Tasks[i]
	var warnings []string

	hasDoc, err := s.api.HasDeployDocLink(t.Key)
	t.HasDeployDoc = err == nil && hasDoc
	t.DeployDocUnknown = err != nil
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("%s: no se pudo verificar el documento de despliegue (%v)", t.Key, err))
	}

	if t.HasCodingErrors {
		obs, err := s.api.GetNovedadComment(t.Key)
		if err != nil {
			obs = "(no se pudieron leer las observaciones)"
			warnings = append(warnings, fmt.Sprintf("%s: %v", t.Key, err))
		}
		t.Observations = obs
	}
	return warnings
}

// FindExisting looks for a report already published with this exact title.
func (s *Session) FindExisting() error {
	page, err := s.api.FindPageByTitle(s.Target.Title, s.Target.SpaceKey)
	if err != nil {
		return err
	}
	s.Existing = page
	return nil
}

// Draft is a report ready to publish, for the user to review first.
type Draft struct {
	Title string
	ADF   map[string]any

	// Replaces is the page an update overwrites; nil for a new report.
	Replaces *atlassian.PageADF

	// NotPreserved names hand-written parts that were not read back and go out
	// empty; when set, the caller must ask before going on.
	NotPreserved []string

	Warnings []string
}

// Names of the hand-written parts, as Draft.NotPreserved reports them.
const (
	PartObservation    = "Observación del Resumen"
	PartConsiderations = "Consideraciones"
)

// DraftCreate builds a new report.
func (s *Session) DraftCreate() *Draft {
	return &Draft{Title: s.Target.Title, ADF: s.build(document.QAPreserved{})}
}

// DraftUpdate builds the replacement for s.Existing, carrying over what QA wrote
// by hand. It fails if the current page cannot be read, rather than overwrite it.
func (s *Session) DraftUpdate() (*Draft, error) {
	if s.Existing == nil {
		return nil, fmt.Errorf("no hay un consolidado existente que actualizar")
	}
	current, err := s.api.GetPageADF(s.Existing.ID)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el consolidado actual; se cancela la actualización para no sobrescribir lo escrito a mano: %w", err)
	}

	d := &Draft{Title: s.Target.Title, Replaces: current}
	var kept document.QAPreserved
	if current.ADF == nil {
		d.Warnings = append(d.Warnings, "el consolidado actual no está en formato ADF; no se pudo leer lo escrito a mano.")
	} else {
		kept = document.ExtractQAPreserved(current.ADF)
	}
	if kept.Observation == nil {
		d.NotPreserved = append(d.NotPreserved, PartObservation)
	}
	if kept.Considerations == nil {
		d.NotPreserved = append(d.NotPreserved, PartConsiderations)
	}
	d.ADF = s.build(kept)
	return d, nil
}

func (s *Session) build(kept document.QAPreserved) map[string]any {
	t := s.Target
	doc := document.QADoc{
		Tasks:        s.Tasks,
		QATasks:      s.QATasks,
		LiderTecnico: t.Report.LiderTecnico,
		PMO:          t.Report.PMO,
		QA:           t.Report.QA,
		Preserved:    kept,
	}
	if t.Kanban {
		doc.Period = t.Period
	} else {
		doc.Sprint, doc.Module = t.Sprint, t.Module
	}
	return document.BuildQA(doc)
}

// Locations lists recent consolidated reports as candidate siblings for a new
// one: the module's in Sprint mode, any in Kanban mode.
func (s *Session) Locations() ([]atlassian.Page, error) {
	pages, err := s.api.FindQAPagesForModule(s.Target.Module, s.Target.SpaceKey)
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, ErrNoLocations
	}
	return pages, nil
}

// Result is a published report.
type Result struct {
	Page    *atlassian.Page
	Updated bool
}

// Publish creates or updates the page. sibling is the page picked from
// Locations; it is ignored for an update.
func (s *Session) Publish(d *Draft, sibling *atlassian.Page) (*Result, error) {
	if d.Replaces != nil {
		page, err := s.api.UpdatePage(d.Replaces.ID, d.Title, d.Replaces.Version, d.ADF)
		if err != nil {
			return nil, err
		}
		return &Result{Page: page, Updated: true}, nil
	}

	if sibling == nil {
		return nil, fmt.Errorf("falta elegir la ubicación del consolidado nuevo")
	}
	ref, err := s.api.GetPage(sibling.ID)
	if err != nil {
		return nil, err
	}
	page, err := s.api.CreatePage(ref.SpaceID, ref.ParentID, d.Title, d.ADF)
	if err != nil {
		return nil, err
	}
	return &Result{Page: page}, nil
}
