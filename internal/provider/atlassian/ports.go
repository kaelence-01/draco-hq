package atlassian

import (
	"context"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

// Tracker is Jira as a provider.Tracker.
type Tracker struct{ c *Client }

// DocStore is one Confluence space as a provider.DocStore; an empty space
// searches every space.
type DocStore struct {
	c     *Client
	space string
}

var (
	_ provider.Tracker  = (*Tracker)(nil)
	_ provider.DocStore = (*DocStore)(nil)
)

// NewTracker returns Jira as a tracker.
func NewTracker(c *Client) *Tracker { return &Tracker{c: c} }

// NewDocStore returns a Confluence space as a document store.
func NewDocStore(c *Client, spaceKey string) *DocStore { return &DocStore{c: c, space: spaceKey} }

func (t *Tracker) Issue(ctx context.Context, key string) (*provider.Issue, error) {
	i, err := t.c.GetIssue(ctx, key)
	if err != nil {
		return nil, err
	}
	return &provider.Issue{Key: i.Key, Summary: i.Summary, URL: i.URL}, nil
}

func (t *Tracker) Me(ctx context.Context) (*provider.User, error) {
	u, err := t.c.Whoami(ctx)
	if err != nil {
		return nil, err
	}
	return &provider.User{ID: u.AccountID, Email: u.EmailAddress, Name: u.DisplayName}, nil
}

func (t *Tracker) LinkDocument(ctx context.Context, issueKey string, doc provider.DocRef) error {
	return t.c.CreateJiraRemoteLink(ctx, issueKey, doc.ID, doc.URL, doc.Title)
}

func (d *DocStore) FindByTitle(ctx context.Context, title string) (*provider.DocRef, error) {
	return ref(d.c.FindPageByTitle(ctx, title, d.space))
}

func (d *DocStore) FindByTitleKey(ctx context.Context, prefix, key string) (*provider.DocRef, error) {
	return ref(d.c.FindPageByTitleKey(ctx, prefix, key, d.space))
}

func (d *DocStore) Read(ctx context.Context, r provider.DocRef) (*provider.Doc, error) {
	p, err := d.c.GetPageADF(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	return &provider.Doc{DocRef: docRef(p.Page), Body: p.ADF}, nil
}

func (d *DocStore) Locations(ctx context.Context, q provider.LocationQuery) (*provider.Locations, error) {
	pages, fellBack, err := d.c.FindRecentPages(ctx, d.space, q.TitlePrefix, q.AuthorID)
	if err != nil {
		return nil, err
	}
	return &provider.Locations{Docs: refs(pages), TeamWide: fellBack}, nil
}

func (d *DocStore) Create(ctx context.Context, near provider.DocRef, title string, body map[string]any) (*provider.DocRef, error) {
	sibling, err := d.c.GetPage(ctx, near.ID)
	if err != nil {
		return nil, err
	}
	return ref(d.c.CreatePage(ctx, sibling.SpaceID, sibling.ParentID, title, body))
}

func (d *DocStore) Update(ctx context.Context, r provider.DocRef, title string, body map[string]any) (*provider.DocRef, error) {
	return ref(d.c.UpdatePage(ctx, r.ID, title, r.Version, body))
}

func (d *DocStore) Search(ctx context.Context, text string) ([]provider.DocRef, error) {
	pages, err := d.c.FindPagesByText(ctx, text, d.space)
	if err != nil {
		return nil, err
	}
	return refs(pages), nil
}

func (d *DocStore) Export(ctx context.Context, r provider.DocRef) (*provider.Exported, error) {
	p, err := d.c.GetPageWithContent(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	e := &provider.Exported{
		DocRef:      provider.DocRef{ID: p.ID, Title: p.Title, URL: p.WebURL, Version: p.Version},
		Source:      "Confluence",
		Space:       p.SpaceName,
		CreatedDate: p.CreatedDate,
		UpdatedDate: p.UpdatedDate,
		CreatedBy:   p.CreatedBy,
		UpdatedBy:   p.UpdatedBy,
	}
	if p.StorageBody != "" {
		e.Text = StorageToText(p.StorageBody)
	}
	return e, nil
}

func docRef(p Page) provider.DocRef {
	return provider.DocRef{ID: p.ID, Title: p.Title, URL: p.WebURL, Version: p.Version}
}

func ref(p *Page, err error) (*provider.DocRef, error) {
	if err != nil || p == nil {
		return nil, err
	}
	r := docRef(*p)
	return &r, nil
}

func refs(pages []Page) []provider.DocRef {
	out := make([]provider.DocRef, len(pages))
	for i, p := range pages {
		out[i] = docRef(p)
	}
	return out
}
