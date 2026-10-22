package atlassian

import (
	"context"
	"fmt"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/config"
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

// QATracker is Jira with the QA report schema (qa_jira) as a
// provider.QATracker.
type QATracker struct {
	*Tracker
	schema QASchema
}

var (
	_ provider.Tracker   = (*Tracker)(nil)
	_ provider.QATracker = (*QATracker)(nil)
	_ provider.DocStore  = (*DocStore)(nil)
)

// NewTracker returns Jira as a tracker.
func NewTracker(c *Client) *Tracker { return &Tracker{c: c} }

// NewQATracker returns Jira as a QA tracker; q must be valid (q.Validate).
func NewQATracker(c *Client, q *config.QAJiraConfig) *QATracker {
	testing := q.TestingStatus
	if strings.TrimSpace(testing) == "" {
		testing = config.DefaultTestingStatus
	}
	return &QATracker{Tracker: NewTracker(c), schema: QASchema{
		Project:           q.Project,
		Statuses:          q.Statuses,
		TestingStatus:     testing,
		ObservationMarker: q.Marker(),
		FieldCodingErrors: q.Fields.CodingErrors,
		FieldDevReturns:   q.Fields.DevReturns,
		FieldDevStatus:    q.Fields.DevStatus,
	}}
}

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

func (t *QATracker) ReviewTasks(ctx context.Context, sc provider.QAScope) ([]provider.QAIssue, error) {
	if sc.SprintName != "" {
		return t.c.GetQATasksForReview(ctx, t.schema, sc.SprintName, sc.Module)
	}
	return t.c.GetQATasksForReviewKanban(ctx, t.schema, sc.Since)
}

func (t *QATracker) AssignedTasks(ctx context.Context, sc provider.QAScope, email string) ([]provider.QAIssue, error) {
	if sc.SprintName != "" {
		return t.c.GetQATasksAsAssignee(ctx, t.schema, sc.SprintName, email)
	}
	return t.c.GetQATasksAsAssigneeKanban(ctx, t.schema, sc.Since, email)
}

func (t *QATracker) HasDocumentLink(ctx context.Context, issueKey string, prefixes []string) (bool, error) {
	return t.c.HasRemoteLink(ctx, issueKey, prefixes)
}

func (t *QATracker) Observations(ctx context.Context, issueKey string) (string, error) {
	return t.c.GetObservations(ctx, issueKey, t.schema.ObservationMarker)
}

func (d *DocStore) FindByTitle(ctx context.Context, title string) (*provider.DocRef, error) {
	return ref(d.c.FindPageByTitle(ctx, title, d.space))
}

func (d *DocStore) FindByTitleKey(ctx context.Context, prefix, key string) (*provider.DocRef, error) {
	return ref(d.c.FindPageByTitleKey(ctx, prefix, key, d.space))
}

func (d *DocStore) Sections(ctx context.Context, r provider.DocRef, names []string) ([]string, error) {
	p, err := d.c.GetPageADF(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	var found []string
	if p.ADF == nil {
		return found, nil
	}
	for _, name := range names {
		if _, ok := preservedSection(p.ADF, name); ok {
			found = append(found, name)
		}
	}
	return found, nil
}
func (d *DocStore) Locations(ctx context.Context, q provider.LocationQuery) (*provider.Locations, error) {
	pages, fellBack, err := d.c.FindRecentPages(ctx, d.space, q.TitlePrefix, q.AuthorID)
	if err != nil {
		return nil, err
	}
	return &provider.Locations{Docs: refs(pages), TeamWide: fellBack}, nil
}

func (d *DocStore) Create(ctx context.Context, near provider.DocRef, doc provider.Document) (*provider.DocRef, error) {
	sibling, err := d.c.GetPage(ctx, near.ID)
	if err != nil {
		return nil, err
	}
	return ref(d.c.CreatePage(ctx, sibling.SpaceID, sibling.ParentID, doc.Title, MarkdownToADF(doc.Markdown)))
}

// Update reads the current page to carry doc.Preserve over in ADF and to
// update on top of the version it read.
func (d *DocStore) Update(ctx context.Context, r provider.DocRef, doc provider.Document) (*provider.DocRef, error) {
	current, err := d.c.GetPageADF(ctx, r.ID)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el documento actual; no se sobrescribe: %w", err)
	}
	body := MarkdownToADF(doc.Markdown)
	if current.ADF != nil {
		for _, name := range doc.Preserve {
			if nodes, ok := preservedSection(current.ADF, name); ok {
				replaceSection(body, name, nodes)
			}
		}
	}
	return ref(d.c.UpdatePage(ctx, current.ID, doc.Title, current.Version, body))
}

func (d *DocStore) Search(ctx context.Context, text string) ([]provider.DocRef, error) {
	pages, err := d.c.FindPagesByText(ctx, text, d.space)
	if err != nil {
		return nil, err
	}
	return refs(pages), nil
}

func (d *DocStore) SearchTitle(ctx context.Context, text string) ([]provider.DocRef, error) {
	pages, err := d.c.SearchPageTitles(ctx, text, d.space)
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
