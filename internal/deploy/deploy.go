// Package deploy is the deploy-document workflow, with no terminal I/O.
// Steps: FetchIssue, FindExisting, ReadCommits, DraftCreate or DraftUpdate,
// Locations (new page only) and Publish; the caller decides between them.
package deploy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/atlassian"
	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/document"
	"github.com/kaelence-01/draco-hq/internal/git"
)

// Atlassian is the part of the Jira and Confluence client the workflow uses.
type Atlassian interface {
	GetIssue(issueKey string) (*atlassian.Issue, error)
	FindPageByTitle(title, spaceKey string) (*atlassian.Page, error)
	FindDeployDocByIssue(issueKey, spaceKey string) (*atlassian.Page, error)
	GetPageADF(pageID string) (*atlassian.PageADF, error)
	GetPage(pageID string) (*atlassian.Page, error)
	Whoami() (*atlassian.JiraUser, error)
	FindLastDeployDoc(spaceKey, accountID string) (pages []atlassian.Page, fellBack bool, err error)
	CreatePage(spaceID, parentID, title string, adfBody map[string]any) (*atlassian.Page, error)
	UpdatePage(pageID, title string, currentVersion int, adfBody map[string]any) (*atlassian.Page, error)
	CreateJiraRemoteLink(issueKey, pageID, pageURL, pageTitle string) error
}

var _ Atlassian = (*atlassian.Client)(nil)

// ChangedFilesFunc lists the files touched by the commits in the repo at
// workDir; empty means the current directory.
type ChangedFilesFunc func(hashes []string, workDir string) ([]string, error)

var (
	// ErrNoSpace means no space key came from the request, project or global config.
	ErrNoSpace = errors.New("Confluence space key no configurado")

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
}

// Repo is one side (backend or frontend) of a deploy.
type Repo struct {
	Name    string
	WorkDir string   // where git runs; empty is the current directory
	Hashes  []string // commits of this side; none means the side is not deployed
}

// Target is a Request resolved against config.
type Target struct {
	Issue       string
	ProjectName string // empty when no project applies
	SpaceKey    string
	VCSHost     string
	VCSOrg      string
	Backend     Repo
	Frontend    Repo
	Checklist   []string

	// Warnings are degraded but workable situations to show the user.
	Warnings []string
}

// Resolve applies config to the request: request > project > global config.
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

	t := &Target{
		Issue:       req.Issue,
		ProjectName: projName,
		VCSHost:     req.VCSHost,
		VCSOrg:      req.VCSOrg,
		Backend:     Repo{Hashes: req.BackendHashes},
		Frontend:    Repo{Hashes: req.FrontendHashes},
		Checklist:   cfg.ResolveDeployChecklist(proj),
	}

	if proj != nil {
		t.Backend.WorkDir, t.Backend.Name = proj.BackendPath, proj.BackendRepo
		t.Frontend.WorkDir, t.Frontend.Name = proj.FrontendPath, proj.FrontendRepo
		if t.VCSHost == "" {
			t.VCSHost = proj.VCSHost
		}
		if t.VCSOrg == "" {
			t.VCSOrg = proj.VCSOrg
		}
	}

	t.SpaceKey = req.Space
	if t.SpaceKey == "" && proj != nil {
		t.SpaceKey = proj.ConfluenceSpaceKey
	}
	if t.SpaceKey == "" {
		t.SpaceKey = cfg.ConfluenceSpaceKey
	}
	if t.SpaceKey == "" {
		return nil, ErrNoSpace
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

// Commits is the result of reading one side's commits.
type Commits struct {
	Hashes []string
	Files  map[string][]string // directory → file names
	Count  int                 // files in total
	Err    error               // set when this side could not be read
}

// Session is one run of the workflow; the steps fill its exported fields.
type Session struct {
	api   Atlassian
	files ChangedFilesFunc

	Target *Target

	Issue    *atlassian.Issue
	Title    string // deterministic from the issue; Confluence's uniqueness key
	Existing *atlassian.Page
	Backend  *Commits // nil when the side is not deployed
	Frontend *Commits
}

// NewSession starts a run for an already resolved target.
func NewSession(api Atlassian, files ChangedFilesFunc, t *Target) *Session {
	return &Session{api: api, files: files, Target: t}
}

// FetchIssue reads the Jira issue and computes the document title.
func (s *Session) FetchIssue() error {
	issue, err := s.api.GetIssue(s.Target.Issue)
	if err != nil {
		return err
	}
	s.Issue = issue
	s.Title = document.BuildTitle(issue.Key, issue.Summary)
	return nil
}

// FindExisting looks for a deploy document already published for the issue:
// by exact title (no index lag), then best effort by issue key for a renamed one.
func (s *Session) FindExisting() error {
	page, err := s.api.FindPageByTitle(s.Title, s.Target.SpaceKey)
	if err != nil {
		return err
	}
	if page == nil {
		if p, ferr := s.api.FindDeployDocByIssue(s.Target.Issue, s.Target.SpaceKey); ferr == nil {
			page = p
		}
	}
	s.Existing = page
	return nil
}

// ReadCommits lists the files of every deployed side. A failed side keeps its
// error in Commits.Err; it is an error only when every side fails.
func (s *Session) ReadCommits() error {
	s.Backend = s.readSide(s.Target.Backend)
	s.Frontend = s.readSide(s.Target.Frontend)

	ok := (s.Backend != nil && s.Backend.Err == nil) || (s.Frontend != nil && s.Frontend.Err == nil)
	if !ok {
		return fmt.Errorf("no se pudo leer ningún commit; en la mayoría de los casos basta con hacer 'git fetch --all' dentro del repo correspondiente")
	}
	return nil
}

func (s *Session) readSide(r Repo) *Commits {
	if len(r.Hashes) == 0 {
		return nil
	}
	c := &Commits{Hashes: r.Hashes}
	files, err := s.files(r.Hashes, r.WorkDir)
	if err != nil {
		c.Err = err
		return c
	}
	c.Files = git.GroupByDirectory(files)
	c.Count = len(files)
	return c
}

// PartialFailure reports whether one side failed while the other was read.
func (s *Session) PartialFailure() bool {
	return (s.Backend != nil && s.Backend.Err != nil) || (s.Frontend != nil && s.Frontend.Err != nil)
}

// Draft is a document ready to publish, for the user to review first.
type Draft struct {
	Title string
	ADF   map[string]any

	// Replaces is the page an update overwrites; nil for a new document.
	Replaces *atlassian.PageADF

	// Preserved reports that "A considerar" was read back; when false on an
	// update the section resets to the template, so the caller must ask.
	Preserved bool

	Warnings []string
}

// DraftCreate builds a new document with the configured checklist.
func (s *Session) DraftCreate() *Draft {
	return &Draft{Title: s.Title, ADF: s.build(nil)}
}

// DraftUpdate builds the replacement for s.Existing, carrying "A considerar"
// over. It fails if the current page cannot be read, rather than overwrite it.
func (s *Session) DraftUpdate() (*Draft, error) {
	if s.Existing == nil {
		return nil, fmt.Errorf("no hay un documento existente que actualizar")
	}
	current, err := s.api.GetPageADF(s.Existing.ID)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el documento actual; se cancela la actualización para no sobrescribir su contenido: %w", err)
	}

	d := &Draft{Title: s.Title, Replaces: current}
	var preserved []any
	if current.ADF == nil {
		d.Warnings = append(d.Warnings, "el documento actual no está en formato ADF; no se pudo leer la sección \"A considerar\".")
	} else {
		preserved = document.ExtractSection(current.ADF, document.ConsiderHeading)
	}
	d.Preserved = len(preserved) > 0
	d.ADF = s.build(preserved)
	return d, nil
}

func (s *Session) build(preserved []any) map[string]any {
	doc := document.DeployDoc{
		IssueKey:          s.Issue.Key,
		IssueSummary:      s.Issue.Summary,
		IssueURL:          s.Issue.URL,
		VCSHost:           s.Target.VCSHost,
		VCSOrg:            s.Target.VCSOrg,
		PreservedConsider: preserved,
		Checklist:         s.Target.Checklist,
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
	return document.Build(doc)
}

// Locations are candidate siblings: a new document goes under the picked one's parent.
type Locations struct {
	Pages []atlassian.Page

	// TeamWide means the list is not limited to the user's own documents.
	TeamWide     bool
	OwnerUnknown bool
}

// Locations lists recent deploy documents, the user's own first choice.
func (s *Session) Locations() (*Locations, error) {
	loc := &Locations{}
	var accountID string
	if me, err := s.api.Whoami(); err == nil {
		accountID = me.AccountID
	} else {
		loc.OwnerUnknown = true
	}

	pages, fellBack, err := s.api.FindLastDeployDoc(s.Target.SpaceKey, accountID)
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, ErrNoLocations
	}
	loc.Pages = pages
	loc.TeamWide = fellBack || loc.OwnerUnknown
	return loc, nil
}

// Result is a published document.
type Result struct {
	Page    *atlassian.Page
	Updated bool

	// LinkErr is a failed Jira remote link: not fatal, but the caller must
	// report it, since the QA report checks that link.
	LinkErr error
}

// Publish creates or updates the page and links it from the Jira issue.
// sibling is the page picked from Locations; it is ignored for an update.
func (s *Session) Publish(d *Draft, sibling *atlassian.Page) (*Result, error) {
	var page *atlassian.Page
	var err error
	res := &Result{}

	if d.Replaces != nil {
		page, err = s.api.UpdatePage(d.Replaces.ID, d.Title, d.Replaces.Version, d.ADF)
		res.Updated = true
	} else {
		if sibling == nil {
			return nil, fmt.Errorf("falta elegir la ubicación del documento nuevo")
		}
		var ref *atlassian.Page
		if ref, err = s.api.GetPage(sibling.ID); err != nil {
			return nil, err
		}
		page, err = s.api.CreatePage(ref.SpaceID, ref.ParentID, d.Title, d.ADF)
	}
	if err != nil {
		return nil, err
	}

	res.Page = page
	res.LinkErr = s.api.CreateJiraRemoteLink(s.Target.Issue, page.ID, page.WebURL, d.Title)
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
