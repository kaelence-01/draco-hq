// Package provider holds the platform-neutral model and the ports the flows
// use; each platform implements them in an adapter (internal/provider/<name>).
// Design: docs/arquitectura/proveedores.md.
package provider

import (
	"context"
	"strings"
	"time"
)

// Issue is a task in a tracker.
type Issue struct {
	Key     string
	Summary string
	URL     string
}

// User is the account the credentials authenticate.
type User struct {
	ID    string
	Email string
	Name  string
}

// DocRef identifies a document in a destination.
type DocRef struct {
	ID      string
	Title   string
	URL     string
	Version int
}

// Document is what a flow publishes.
type Document struct {
	Title    string
	Markdown string
	// Preserve names the sections, by heading, that people edit by hand: on
	// Update the destination keeps their current content instead of the
	// template's.
	Preserve []string
}

// LocationQuery selects the documents a new one can be created next to.
type LocationQuery struct {
	TitlePrefix string
	AuthorID    string // empty: anyone's
}

// Locations are the candidate siblings for a new document.
type Locations struct {
	Docs []DocRef
	// TeamWide is true when AuthorID was set but that user has no documents,
	// so the list is everyone's.
	TeamWide bool
}

// Exported is a document converted to plain text, with its metadata.
type Exported struct {
	DocRef
	Source      string // platform name, e.g. "Confluence"
	Space       string
	CreatedDate string // YYYY-MM-DD
	UpdatedDate string
	CreatedBy   string
	UpdatedBy   string
	Text        string
}

// Tracker is where the issues live.
type Tracker interface {
	Issue(ctx context.Context, key string) (*Issue, error)
	Me(ctx context.Context) (*User, error)
	// LinkDocument links the document from the issue; linking again updates
	// the link instead of duplicating it.
	LinkDocument(ctx context.Context, issueKey string, doc DocRef) error
}

// DocStore is where the documents are published. A nil *DocRef with a nil
// error means "not found".
type DocStore interface {
	FindByTitle(ctx context.Context, title string) (*DocRef, error)
	// FindByTitleKey finds a document whose title has both prefix and key; it
	// may lag behind recent changes, unlike FindByTitle.
	FindByTitleKey(ctx context.Context, prefix, key string) (*DocRef, error)
	// Sections reports which of the named sections the current document has.
	Sections(ctx context.Context, ref DocRef, names []string) ([]string, error)
	Locations(ctx context.Context, q LocationQuery) (*Locations, error)
	// Create places the document next to near (same parent).
	Create(ctx context.Context, near DocRef, doc Document) (*DocRef, error)
	// Update replaces the document, keeping doc.Preserve. It fails without
	// writing anything when the current document cannot be read.
	Update(ctx context.Context, ref DocRef, doc Document) (*DocRef, error)
	Search(ctx context.Context, text string) ([]DocRef, error)
	// SearchTitle finds documents whose title matches text, newest first,
	// across the whole destination; it may lag behind recent changes.
	SearchTitle(ctx context.Context, text string) ([]DocRef, error)
	Export(ctx context.Context, ref DocRef) (*Exported, error)
}

// QAIssue is one task evaluated in the QA consolidated report.
type QAIssue struct {
	Key             string
	URL             string
	Summary         string
	HasCodingErrors bool
	HasDevReturns   bool
	HasDeployDoc    bool
	PRMerged        bool
	Observations    string
	ReviewTaskKey   string // the QA review task of this one
	ReviewTaskURL   string

	// DeployDocUnknown means the link lookup failed: the report says
	// "unverified" instead of a failed check, because it is QA evidence.
	DeployDocUnknown bool
}

// QAScope selects the tasks of one report: a sprint of a module, or (Kanban)
// everything that reached testing since a date.
type QAScope struct {
	SprintName string
	Module     string
	Since      string // YYYY-MM-DD; used when SprintName is empty
}

// QATracker is the optional tracker capability behind the QA consolidated
// report. Its schema (project, statuses, fields) is given to the adapter when
// it is built, never by the flow.
type QATracker interface {
	Tracker
	// ReviewTasks returns the tasks of the scope that reached QA or passed it.
	ReviewTasks(ctx context.Context, scope QAScope) ([]QAIssue, error)
	// AssignedTasks returns the tasks of the scope assigned to email.
	AssignedTasks(ctx context.Context, scope QAScope, email string) ([]QAIssue, error)
	// HasDocumentLink reports a link from the issue whose title starts with
	// prefix. An error means "could not check", never "no link".
	HasDocumentLink(ctx context.Context, issueKey, prefix string) (bool, error)
	// Observations joins the text after the configured marker ("Novedad::")
	// in the issue's comments; empty with a nil error means there are none.
	Observations(ctx context.Context, issueKey string) (string, error)
}

// Commit is one commit of a repository.
type Commit struct {
	Hash      string
	ShortHash string
	Subject   string
	Author    string
	Date      time.Time
}

// ChangeSource is where the changes of a deploy come from. workDir is the
// local clone; empty is the current directory.
type ChangeSource interface {
	// ChangedFiles returns the deduplicated files of the commits, in order of
	// first appearance.
	ChangedFiles(ctx context.Context, workDir string, hashes []string) ([]string, error)
	RecentCommits(ctx context.Context, workDir string, n int) ([]Commit, error)
}

// NormalizeHeading is how every destination compares section titles: case,
// surrounding space and a trailing colon (draco emits "A considerar:") are
// ignored.
func NormalizeHeading(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.TrimSpace(strings.TrimSuffix(s, ":"))
}
