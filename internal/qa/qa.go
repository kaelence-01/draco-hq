// Package qa is the QA consolidated report workflow, with no terminal I/O.
// Steps: CheckAccount, Collect, EvaluateTask per task, FindExisting, DraftCreate
// or DraftUpdate, Locations (new page only) and Publish.
package qa

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/document"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

var (
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

	// Template is a folder with a QA template that overrides the repo's and
	// the user's.
	Template string
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
	SpaceKey string // destination scope from the request; empty: the profile's
	QAEmail  string
	Report   config.QAReportConfig
	Date     time.Time // review date printed in the report

	// Template renders the report; nil is the built-in one. DeployPrefix
	// starts deploy document titles, which is how a task's link to its deploy
	// document is recognized; empty is the built-in one.
	Template     *document.Template
	DeployPrefix string

	// ReviewPrefixes start the summaries of QA review tasks (qa_jira).
	ReviewPrefixes []string

	// Warnings are degraded but workable situations to show the user.
	Warnings []string
}

// Resolve applies config to the request; now sets the Kanban window. The
// destination and its default scope come from the profile.
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
	t.ReviewPrefixes = cfg.QAJira.Reviews()
	if err := t.loadTemplates(req.Template); err != nil {
		return nil, err
	}

	var title string
	var err error
	if t.Kanban {
		since, period := kanbanWindow(now)
		t.Since, t.Period = since.Format("2006-01-02"), period
		title, err = document.QATitle(t.template(), "", 0, period)
		if module != "" {
			t.warn("el módulo no aplica en modo Kanban: se incluyen las tareas de todos los módulos.")
		}
	} else {
		t.Sprint, t.Module = req.Sprint, module
		t.SprintName = cfg.QAJira.SprintNameFor(module, req.Sprint)
		title, err = document.QATitle(t.template(), module, req.Sprint, "")
	}
	if err != nil {
		return nil, err
	}
	t.Title = title

	t.SpaceKey = req.Space

	if cfg.QAReport != nil {
		t.Report = *cfg.QAReport
	} else {
		t.warn("no hay 'qa_report' en la configuración; Líder Técnico, PMO y QA saldrán vacíos.")
	}
	return t, nil
}

func (t *Target) warn(msg string) { t.Warnings = append(t.Warnings, msg) }

func (t *Target) template() *document.Template {
	if t.Template == nil {
		return document.MustBuiltin("qa")
	}
	return t.Template
}

func (t *Target) deployPrefix() string {
	if t.DeployPrefix == "" {
		return document.MustBuiltin("deploy").Manifest.Prefix
	}
	return t.DeployPrefix
}

// loadTemplates resolves the QA template (explicit folder, then the current
// repo's .draco/templates/qa, then the user's) and the deploy prefix the same
// way, without an explicit folder.
func (t *Target) loadTemplates(explicit string) error {
	user, _ := config.TemplatesDir()
	var err error
	if t.Template, err = document.LoadTemplate("qa", document.Dirs("qa", explicit, "", user)...); err != nil {
		return err
	}
	deploy, err := document.LoadTemplate("deploy", document.Dirs("deploy", "", "", user)...)
	if err != nil {
		return err
	}
	t.DeployPrefix = deploy.Manifest.Prefix
	return nil
}

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
	s.Tasks = devTasksOnly(all, s.Target.ReviewPrefixes)
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

	hasDoc, err := s.tracker.HasDocumentLink(ctx, t.Key, s.Target.deployPrefix())
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
	Title    string
	Markdown string

	// Replaces is the document an update overwrites; nil for a new report.
	Replaces *provider.DocRef

	// NotPreserved names hand-written sections the current report lacks: they
	// go out empty, so when set the caller must ask before going on.
	NotPreserved []string
}

// DraftCreate builds a new report.
func (s *Session) DraftCreate() (*Draft, error) {
	md, err := s.build()
	if err != nil {
		return nil, err
	}
	return &Draft{Title: s.Target.Title, Markdown: md}, nil
}

// DraftUpdate builds the replacement for s.Existing; the destination carries
// what QA wrote by hand over when publishing. It fails if the current report
// cannot be read, rather than overwrite it.
func (s *Session) DraftUpdate(ctx context.Context) (*Draft, error) {
	if s.Existing == nil {
		return nil, fmt.Errorf("no hay un consolidado existente que actualizar")
	}
	handWritten := s.Target.template().Manifest.Preserve
	found, err := s.docs.Sections(ctx, *s.Existing, handWritten)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el consolidado actual; se cancela la actualización para no sobrescribir lo escrito a mano: %w", err)
	}
	md, err := s.build()
	if err != nil {
		return nil, err
	}
	d := &Draft{Title: s.Target.Title, Markdown: md, Replaces: s.Existing}
	for _, name := range handWritten {
		if !slices.Contains(found, name) {
			d.NotPreserved = append(d.NotPreserved, name)
		}
	}
	return d, nil
}

func (s *Session) build() (string, error) {
	t := s.Target
	doc := document.QADoc{
		Date:         t.Date,
		Tasks:        s.Tasks,
		LiderTecnico: t.Report.LiderTecnico,
		PMO:          t.Report.PMO,
		QA:           t.Report.QA,
	}
	if t.Kanban {
		doc.Period = t.Period
	} else {
		doc.Sprint, doc.Module = t.Sprint, t.Module
	}
	return document.QA(s.Target.template(), doc)
}

// Locations lists recent consolidated reports as candidate siblings for a new
// one: the module's in Sprint mode, any in Kanban mode. Without any, it falls
// back to the destination's own locations (a Markdown folder offers itself).
func (s *Session) Locations(ctx context.Context) ([]provider.DocRef, error) {
	prefix := s.Target.template().Manifest.Prefix
	title := prefix
	if s.Target.Module != "" {
		title += " - " + s.Target.Module
	}
	pages, err := s.docs.SearchTitle(ctx, title)
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		loc, err := s.docs.Locations(ctx, provider.LocationQuery{TitlePrefix: prefix})
		if err != nil {
			return nil, err
		}
		pages = loc.Docs
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
	doc := provider.Document{Title: d.Title, Markdown: d.Markdown, Preserve: s.Target.template().Manifest.Preserve}
	if d.Replaces != nil {
		page, err := s.docs.Update(ctx, *d.Replaces, doc)
		if err != nil {
			return nil, err
		}
		return &Result{Page: page, Updated: true}, nil
	}

	if sibling == nil {
		return nil, fmt.Errorf("falta elegir la ubicación del consolidado nuevo")
	}
	page, err := s.docs.Create(ctx, *sibling, doc)
	if err != nil {
		return nil, err
	}
	return &Result{Page: page}, nil
}
