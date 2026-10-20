// Package provider holds the platform-neutral model and the ports the flows
// use; each platform implements them in an adapter (internal/provider/<name>).
// Design: docs/arquitectura/proveedores.md.
package provider

import "context"

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
	Version int // as last read; Update needs it where the destination checks it
}

// Doc is a document read back with its body.
type Doc struct {
	DocRef
	// Body is the destination's native body (ADF for Confluence) until documents
	// move to Markdown; nil when the page uses a format draco cannot read.
	Body map[string]any
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
	Read(ctx context.Context, ref DocRef) (*Doc, error)
	Locations(ctx context.Context, q LocationQuery) (*Locations, error)
	// Create places the document next to near (same parent).
	Create(ctx context.Context, near DocRef, title string, body map[string]any) (*DocRef, error)
	// Update replaces the whole body; ref.Version is the version read.
	Update(ctx context.Context, ref DocRef, title string, body map[string]any) (*DocRef, error)
	Search(ctx context.Context, text string) ([]DocRef, error)
	Export(ctx context.Context, ref DocRef) (*Exported, error)
}
