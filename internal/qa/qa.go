// Package qa is the QA consolidated report workflow, with no terminal I/O.
// Steps: CheckAccount, Collect, EvaluateTask per task, FindExisting, DraftCreate
// or DraftUpdate, Locations (new page only) and Publish.
package qa

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/document"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

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

	Title    string // deterministic; the destination's uniqueness key
	SpaceKey string // scope of the DocStore
	QAEmail  string
	Report   config.QAReportConfig
	Date     time.Time // review date printed in the report

	// Warnings are degraded but workable situations to show the user.
	Warnings []string
}

// Resolve applies config to the request; now sets the Kanban window. The space
// key comes from the request or the global config, not from a project.
func Resolve(cfg *config.Config, req Request, now time.Time) (*Target, error) {
	if req.Sprint < 0 {
		return nil, fmt.Errorf("el sprint debe ser un número positivo: %d", req.Sprint)
	}
	t := &Target{Kanban: req.Sprint == 0, QAEmail: cfg.QAEmail, Date: now}
	module := strings.TrimSpace(req.Module)
	if !t.Kanban && module == "" {
		return nil, fmt.Errorf("falta el módulo: es obligatorio en modo Sprint")
	}

	if err := cfg.QAJira.Validate(); err != nil {
		return nil, err
	}

	if t.Kanban {
		since, period := kanbanWindow(now)
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

func (t *Target) scope() provider.QAScope {
	if t.Kanban {
		return provider.QAScope{Since: t.Since}
	}
	return provider.QAScope{SprintName: t.SprintName, Module: t.Module}
}

// Session is one run of the workflow; the steps fill its exported fields.
type Session struct {
	tracker provider.QATracker
	docs    provider.DocStore

	Target *Target

	Account  *provider.User
	Tasks    []provider.QAIssue // under review: the rows of the report
	QATasks  []provider.QAIssue // assigned to the QA account
	Existing *provider.DocRef
}

// NewSession starts a run for an already resolved target.
func NewSession(tracker provider.QATracker, docs provider.DocStore, t *Target) *Session {
	return &Session{tracker: tracker, docs: docs, Target: t}
}

// CheckAccount verifies that the authenticated account is qa_email. It guards
// against accidents, not access: authorization is the platform's permissions.
func (s *Session) CheckAccount(ctx context.Context) error {
	me, err := s.tracker.Me(ctx)
	if err != nil {
		return err
	}
	s.Account = me
	if s.Target.QAEmail == "" || !strings.EqualFold(me.Email, s.Target.QAEmail) {
		return ErrNotQAAccount
	}
	return nil
}

// Collect reads the dev tasks under review and the QA account's own tasks, and
// links each task to its review task.
func (s *Session) Collect(ctx context.Context) error {
	sc := s.Target.scope()
	all, err := s.tracker.ReviewTasks(ctx, sc)
	if err != nil {
		return err
	}
	s.Tasks = devTasksOnly(all)
	if s.QATasks, err = s.tracker.AssignedTasks(ctx, sc, s.Target.QAEmail); err != nil {
		return err
	}

	reviews := buildReviewMap(s.QATasks)
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
func (s *Session) EvaluateTask(ctx context.Context, i int) []string {
	t := &s.Tasks[i]
	var warnings []string

	hasDoc, err := s.tracker.HasDocumentLink(ctx, t.Key, document.DeployTitlePrefix)
	t.HasDeployDoc = err == nil && hasDoc
	t.DeployDocUnknown = err != nil
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("%s: no se pudo verificar el documento de despliegue (%v)", t.Key, err))
	}

	if t.HasCodingErrors {
		obs, err := s.tracker.Observations(ctx, t.Key)
		if err != nil {
			obs = "(no se pudieron leer las observaciones)"
			warnings = append(warnings, fmt.Sprintf("%s: %v", t.Key, err))
		}
		t.Observations = obs
	}
	return warnings
}

// FindExisting looks for a report already published with this exact title.
func (s *Session) FindExisting(ctx context.Context) error {
	page, err := s.docs.FindByTitle(ctx, s.Target.Title)
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
	Replaces *provider.Doc

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
func (s *Session) DraftUpdate(ctx context.Context) (*Draft, error) {
	if s.Existing == nil {
		return nil, fmt.Errorf("no hay un consolidado existente que actualizar")
	}
	current, err := s.docs.Read(ctx, *s.Existing)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el consolidado actual; se cancela la actualización para no sobrescribir lo escrito a mano: %w", err)
	}

	d := &Draft{Title: s.Target.Title, Replaces: current}
	var kept document.QAPreserved
	if current.Body == nil {
		d.Warnings = append(d.Warnings, "el consolidado actual no está en formato ADF; no se pudo leer lo escrito a mano.")
	} else {
		kept = document.ExtractQAPreserved(current.Body)
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
		Date:         t.Date,
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
func (s *Session) Locations(ctx context.Context) ([]provider.DocRef, error) {
	title := document.QATitlePrefix
	if s.Target.Module != "" {
		title += " - " + s.Target.Module
	}
	pages, err := s.docs.SearchTitle(ctx, title)
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
	Page    *provider.DocRef
	Updated bool
}

// Publish creates or updates the report. sibling is the document picked from
// Locations; it is ignored for an update.
func (s *Session) Publish(ctx context.Context, d *Draft, sibling *provider.DocRef) (*Result, error) {
	if d.Replaces != nil {
		page, err := s.docs.Update(ctx, d.Replaces.DocRef, d.Title, d.ADF)
		if err != nil {
			return nil, err
		}
		return &Result{Page: page, Updated: true}, nil
	}

	if sibling == nil {
		return nil, fmt.Errorf("falta elegir la ubicación del consolidado nuevo")
	}
	page, err := s.docs.Create(ctx, *sibling, d.Title, d.ADF)
	if err != nil {
		return nil, err
	}
	return &Result{Page: page}, nil
}
