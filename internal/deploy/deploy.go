// Package deploy is the deploy-document workflow, with no terminal I/O.
// Steps: FetchIssue, FindExisting, ReadCommits, DraftCreate or DraftUpdate,
// Locations (new page only) and Publish; the caller decides between them.
package deploy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/document"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

var (
	// ErrNoLocations means no previous deploy document exists to sit next to.
	ErrNoLocations = errors.New("no se encontraron documentos de despliegue previos. Crea uno manualmente primero como referencia")
)

// Request is what the user asks for, before applying config.
type Request struct {
	Issue          string
	BackendHashes  []string
	FrontendHashes []string

	Project string // empty: default_project
	Space   string // overrides the project and global space key
	VCSHost string // overrides the project's vcs_host
	VCSOrg  string // overrides the project's vcs_org

	// Template is a folder with a deploy template that overrides the repo's
	// and the user's.
	Template string
}

// Repo is one side (backend or frontend) of a deploy.
type Repo struct {
	Name    string
	WorkDir string   // where git runs; empty is the current directory
	Hashes  []string // commits of this side; none means the side is not deployed
}

// Sides is what a deploy takes from a project before the commits are chosen:
// the visual mode lists the commits of each side with it.
type Sides struct {
	Project  string // empty when no project applies
	Backend  Repo
	Frontend Repo
	SpaceKey string // the project's destination scope; empty: the profile's
}

// ProjectSides returns the sides of a project; empty is default_project.
func ProjectSides(cfg *config.Config, project string) (*Sides, error) {
	proj, name, err := cfg.GetProject(project)
	if err != nil {
		return nil, err
	}
	return sidesOf(proj, name), nil
}

func sidesOf(proj *config.ProjectConfig, name string) *Sides {
	s := &Sides{Project: name}
	if proj != nil {
		s.Backend = Repo{Name: proj.BackendRepo, WorkDir: proj.BackendPath}
		s.Frontend = Repo{Name: proj.FrontendRepo, WorkDir: proj.FrontendPath}
		s.SpaceKey = proj.ConfluenceSpaceKey
	}
	return s
}

// Deployable reports the sides with a clone to read commits from. Without a
// frontend path the backend is always one: its clone, or the current directory.
func (s *Sides) Deployable() (backend, frontend bool) {
	frontend = s.Frontend.WorkDir != ""
	return s.Backend.WorkDir != "" || !frontend, frontend
}

// Target is a Request resolved against config.
type Target struct {
	Issue       string
	ProjectName string // empty when no project applies
	SpaceKey    string // destination scope from the request or the project; empty: the profile's
	VCSHost     string
	VCSOrg      string
	Backend     Repo
	Frontend    Repo
	Checklist   []string

	// Template renders the document; nil is the built-in one.
	Template *document.Template

	// Warnings are degraded but workable situations to show the user.
	Warnings []string
}

// Resolve applies config to the request: request > project > global config.
// The destination itself, and its default scope, come from the profile.
func Resolve(cfg *config.Config, req Request) (*Target, error) {
	if strings.TrimSpace(req.Issue) == "" {
		return nil, fmt.Errorf("falta la clave del issue de Jira")
	}
	if len(req.BackendHashes) == 0 && len(req.FrontendHashes) == 0 {
		return nil, fmt.Errorf("debes indicar al menos un commit de backend o de frontend")
	}

	proj, projName, err := cfg.GetProject(req.Project)
	if err != nil {
		return nil, err
	}

	sides := sidesOf(proj, projName)
	t := &Target{
		Issue:       req.Issue,
		ProjectName: projName,
		SpaceKey:    req.Space,
		VCSHost:     req.VCSHost,
		VCSOrg:      req.VCSOrg,
		Backend:     sides.Backend,
		Frontend:    sides.Frontend,
		Checklist:   cfg.ResolveDeployChecklist(proj),
	}
	t.Backend.Hashes, t.Frontend.Hashes = req.BackendHashes, req.FrontendHashes

	if proj != nil {
		if t.VCSHost == "" {
			t.VCSHost = proj.VCSHost
		}
		if t.VCSOrg == "" {
			t.VCSOrg = proj.VCSOrg
		}
	}
	if t.SpaceKey == "" {
		t.SpaceKey = sides.SpaceKey
	}

	if t.Template, err = loadTemplate(req.Template, t.Backend.WorkDir); err != nil {
		return nil, err
	}

	t.resolveSide(&t.Backend, "backend", proj != nil)
	t.resolveSide(&t.Frontend, "frontend", proj != nil)

	if t.VCSHost == "" || t.VCSOrg == "" {
		t.warn("VCS host/org no configurado: los links del documento no serán clickeables. Configura vcs_host y vcs_org en tu proyecto.")
	}
	return t, nil
}

// resolveSide guesses a missing repo name from the clone directory (P-008).
func (t *Target) resolveSide(r *Repo, side string, hasProject bool) {
	if len(r.Hashes) == 0 {
		return
	}
	if hasProject && r.WorkDir == "" {
		t.warn(fmt.Sprintf("el proyecto no tiene %s_path configurado: git correrá en el directorio actual.", side))
	}
	if r.Name == "" {
		r.Name = repoNameFromDir(r.WorkDir)
		t.warn(fmt.Sprintf("nombre del repo %s no configurado; se usará %q (tomado del directorio).", side, r.Name))
	}
}

func (t *Target) warn(msg string) { t.Warnings = append(t.Warnings, msg) }

func (t *Target) template() *document.Template {
	if t.Template == nil {
		return document.MustBuiltin("deploy")
	}
	return t.Template
}

// loadTemplate resolves the deploy template: the explicit folder, then the
// repo's .draco/templates/deploy (repoDir empty: the current directory), then
// the user's; the built-in one is the base of all of them.
func loadTemplate(explicit, repoDir string) (*document.Template, error) {
	user, _ := config.TemplatesDir()
	return document.LoadTemplate("deploy", document.Dirs("deploy", explicit, repoDir, user)...)
}

// Commits is the result of reading one side's commits.
type Commits struct {
	Hashes []string
	Files  map[string][]string // directory → file names
	Count  int                 // files in total
	Err    error               // set when this side could not be read
}

// Session is one run of the workflow; the steps fill its exported fields.
type Session struct {
	tracker provider.Tracker
	docs    provider.DocStore
	changes provider.ChangeSource

	Target *Target

	Issue    *provider.Issue
	Title    string // deterministic from the issue; the destination's uniqueness key
	Existing *provider.DocRef
	Backend  *Commits // nil when the side is not deployed
	Frontend *Commits
}

// NewSession starts a run for an already resolved target.
func NewSession(tracker provider.Tracker, docs provider.DocStore, changes provider.ChangeSource, t *Target) *Session {
	return &Session{tracker: tracker, docs: docs, changes: changes, Target: t}
}

// FetchIssue reads the issue and computes the document title.
func (s *Session) FetchIssue(ctx context.Context) error {
	issue, err := s.tracker.Issue(ctx, s.Target.Issue)
	if err != nil {
		return err
	}
	s.Issue = issue
	title, err := document.DeployTitle(s.Target.template(), issue.Key, issue.Summary)
	if err != nil {
		return err
	}
	s.Title = title
	return nil
}

// FindExisting looks for a deploy document already published for the issue:
// by exact title (no index lag), then best effort by issue key for a renamed one.
func (s *Session) FindExisting(ctx context.Context) error {
	page, err := s.docs.FindByTitle(ctx, s.Title)
	if err != nil {
		return err
	}
	if page == nil {
		if p, ferr := s.docs.FindByTitleKey(ctx, s.Target.template().Manifest.Prefix, s.Target.Issue); ferr == nil {
			page = p
		}
	}
	s.Existing = page
	return nil
}

// ReadCommits lists the files of every deployed side. A failed side keeps its
// error in Commits.Err; it is an error only when every side fails.
func (s *Session) ReadCommits(ctx context.Context) error {
	s.Backend = s.readSide(ctx, s.Target.Backend)
	s.Frontend = s.readSide(ctx, s.Target.Frontend)

	ok := (s.Backend != nil && s.Backend.Err == nil) || (s.Frontend != nil && s.Frontend.Err == nil)
	if !ok {
		return fmt.Errorf("no se pudo leer ningún commit; en la mayoría de los casos basta con hacer 'git fetch --all' dentro del repo correspondiente")
	}
	return nil
}

func (s *Session) readSide(ctx context.Context, r Repo) *Commits {
	if len(r.Hashes) == 0 {
		return nil
	}
	c := &Commits{Hashes: r.Hashes}
	files, err := s.changes.ChangedFiles(ctx, r.WorkDir, r.Hashes)
	if err != nil {
		c.Err = err
		return c
	}
	c.Files = groupByDirectory(files)
	c.Count = len(files)
	return c
}

// PartialFailure reports whether one side failed while the other was read.
func (s *Session) PartialFailure() bool {
	return (s.Backend != nil && s.Backend.Err != nil) || (s.Frontend != nil && s.Frontend.Err != nil)
}

// Draft is a document ready to publish, for the user to review first.
type Draft struct {
	Title    string
	Markdown string

	// Replaces is the document an update overwrites; nil for a new one.
	Replaces *provider.DocRef

	// NotPreserved names the hand-written sections ("A considerar") the
	// current document lacks: they go out as the template has them, so when
	// set on an update the caller must ask.
	NotPreserved []string
}

// DraftCreate builds a new document with the configured checklist.
func (s *Session) DraftCreate() (*Draft, error) {
	md, err := s.build()
	if err != nil {
		return nil, err
	}
	return &Draft{Title: s.Title, Markdown: md}, nil
}

// DraftUpdate builds the replacement for s.Existing; the destination carries
// "A considerar" over when publishing. It fails if the current document cannot
// be read, rather than overwrite it.
func (s *Session) DraftUpdate(ctx context.Context) (*Draft, error) {
	if s.Existing == nil {
		return nil, fmt.Errorf("no hay un documento existente que actualizar")
	}
	preserve := s.Target.template().Manifest.Preserve
	found, err := s.docs.Sections(ctx, *s.Existing, preserve)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el documento actual; se cancela la actualización para no sobrescribir su contenido: %w", err)
	}
	md, err := s.build()
	if err != nil {
		return nil, err
	}
	d := &Draft{Title: s.Title, Markdown: md, Replaces: s.Existing}
	for _, name := range preserve {
		if !slices.Contains(found, name) {
			d.NotPreserved = append(d.NotPreserved, name)
		}
	}
	return d, nil
}

func (s *Session) build() (string, error) {
	doc := document.DeployDoc{
		IssueKey:     s.Issue.Key,
		IssueSummary: s.Issue.Summary,
		IssueURL:     s.Issue.URL,
		VCSHost:      s.Target.VCSHost,
		VCSOrg:       s.Target.VCSOrg,
		Checklist:    s.Target.Checklist,
	}
	if s.Backend != nil && s.Backend.Err == nil {
		doc.BackendRepo = s.Target.Backend.Name
		doc.BackendCommit = s.Backend.Hashes[0]
		doc.BackendFiles = s.Backend.Files
	}
	if s.Frontend != nil && s.Frontend.Err == nil {
		doc.FrontendRepo = s.Target.Frontend.Name
		doc.FrontendCommit = s.Frontend.Hashes[0]
		doc.FrontendFiles = s.Frontend.Files
	}
	return document.Deploy(s.Target.template(), doc)
}

// Locations are candidate siblings: a new document goes under the picked one's parent.
type Locations struct {
	Docs []provider.DocRef

	// TeamWide means the list is not limited to the user's own documents.
	TeamWide     bool
	OwnerUnknown bool
}

// Locations lists recent deploy documents, the user's own first choice.
func (s *Session) Locations(ctx context.Context) (*Locations, error) {
	loc := &Locations{}
	q := provider.LocationQuery{TitlePrefix: s.Target.template().Manifest.Prefix}
	if me, err := s.tracker.Me(ctx); err == nil {
		q.AuthorID = me.ID
	} else {
		loc.OwnerUnknown = true
	}

	found, err := s.docs.Locations(ctx, q)
	if err != nil {
		return nil, err
	}
	if len(found.Docs) == 0 {
		return nil, ErrNoLocations
	}
	loc.Docs = found.Docs
	loc.TeamWide = found.TeamWide || loc.OwnerUnknown
	return loc, nil
}

// Result is a published document.
type Result struct {
	Page    *provider.DocRef
	Updated bool

	// LinkErr is a failed link from the issue: not fatal, but the caller must
	// report it, since the QA report checks that link.
	LinkErr error
}

// Publish creates or updates the document and links it from the issue.
// sibling is the document picked from Locations; it is ignored for an update.
func (s *Session) Publish(ctx context.Context, d *Draft, sibling *provider.DocRef) (*Result, error) {
	var page *provider.DocRef
	var err error
	res := &Result{}

	doc := provider.Document{Title: d.Title, Markdown: d.Markdown, Preserve: s.Target.template().Manifest.Preserve}
	if d.Replaces != nil {
		page, err = s.docs.Update(ctx, *d.Replaces, doc)
		res.Updated = true
	} else {
		if sibling == nil {
			return nil, fmt.Errorf("falta elegir la ubicación del documento nuevo")
		}
		page, err = s.docs.Create(ctx, *sibling, doc)
	}
	if err != nil {
		return nil, err
	}

	res.Page = page
	link := provider.DocRef{ID: page.ID, Title: d.Title, URL: page.URL}
	res.LinkErr = s.tracker.LinkDocument(ctx, s.Target.Issue, link)
	return res, nil
}

// SplitHashes splits a comma-separated list of commit hashes, dropping blanks.
func SplitHashes(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if h := strings.TrimSpace(p); h != "" {
			out = append(out, h)
		}
	}
	return out
}

// repoNameFromDir guesses the repo name from the clone directory (empty: the current one).
func repoNameFromDir(workDir string) string {
	dir := workDir
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "repo"
		}
		dir = wd
	}
	base := filepath.Base(filepath.Clean(dir))
	if base == "." || base == string(filepath.Separator) {
		return "repo"
	}
	return base
}

// groupByDirectory groups file names by their parent directory; root files go
// under ".".
func groupByDirectory(files []string) map[string][]string {
	groups := make(map[string][]string)
	for _, f := range files {
		dir, name := ".", f
		if i := strings.LastIndex(f, "/"); i >= 0 {
			dir, name = f[:i], f[i+1:]
		}
		groups[dir] = append(groups[dir], name)
	}
	return groups
}
