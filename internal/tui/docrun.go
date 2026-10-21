package tui

import (
	"context"
	"errors"
	"fmt"

	"github.com/kaelence-01/draco-hq/internal/deploy"
	"github.com/kaelence-01/draco-hq/internal/profile"
	"github.com/kaelence-01/draco-hq/internal/provider"
	"github.com/kaelence-01/draco-hq/internal/qa"
)

// docRun is a document flow at the steps that Generar and QA share: the
// draft, where a new document goes, and publishing. Its implementations only
// translate the flow's types.
type docRun interface {
	about() docAbout
	// updating reports whether the draft replaces a published document.
	updating() bool
	// draft builds the document; publish sends that same draft.
	draft(ctx context.Context) (*docDraft, error)
	locations(ctx context.Context) (*docLocations, error)
	publish(ctx context.Context, sibling *provider.DocRef) (*docResult, error)
}

// docAbout is what the shared screens say about the run.
type docAbout struct {
	noun    string // the document, capitalized: "Documento", "Consolidado"
	recent  string // what Locations looks for
	dest    string // the destination provider
	issue   string // the task publishing links from; empty for none
	tracker string
	command string // the same run in the command mode
}

type docDraft struct {
	title, markdown string
	replaces        *provider.DocRef
	preserve        []string // hand-written sections the destination keeps
	notPreserved    []string // the ones the current document lacks
	files           []string // lines of the Archivos tab; nil: no tab
}

type docLocations struct {
	docs                   []provider.DocRef
	teamWide, ownerUnknown bool
}

type docResult struct {
	page    *provider.DocRef
	updated bool
	linkErr error
}

// genRun is the deploy document after the progress.
type genRun struct {
	bundle  *profile.Bundle
	session *deploy.Session
	replace bool   // update the existing document
	command string // the same run in the command mode
	d       *deploy.Draft
}

func (r *genRun) about() docAbout {
	return docAbout{noun: "Documento", recent: "documentos de despliegue recientes", dest: r.bundle.Profile.Docs.Provider,
		issue: r.session.Target.Issue, tracker: r.bundle.Profile.Tracker.Provider, command: r.command}
}

func (r *genRun) updating() bool { return r.replace }

func (r *genRun) draft(ctx context.Context) (*docDraft, error) {
	var err error
	if r.replace {
		r.d, err = r.session.DraftUpdate(ctx)
	} else {
		r.d, err = r.session.DraftCreate()
	}
	if err != nil {
		return nil, err
	}
	return &docDraft{title: r.d.Title, markdown: r.d.Markdown, replaces: r.d.Replaces, notPreserved: r.d.NotPreserved,
		preserve: r.session.Target.Template.Manifest.Preserve, files: readFiles(r.session)}, nil
}

func (r *genRun) locations(ctx context.Context) (*docLocations, error) {
	loc, err := r.session.Locations(ctx)
	if err != nil {
		return nil, explainLocations(err)
	}
	return &docLocations{docs: loc.Docs, teamWide: loc.TeamWide, ownerUnknown: loc.OwnerUnknown}, nil
}

func (r *genRun) publish(ctx context.Context, sibling *provider.DocRef) (*docResult, error) {
	res, err := r.session.Publish(ctx, r.d, sibling)
	if err != nil {
		return nil, err
	}
	return &docResult{page: res.Page, updated: res.Updated, linkErr: res.LinkErr}, nil
}

// qaRun is the QA report after its evaluation. A report with the same title
// is updated, as draco qa does: the title is the report's key.
type qaRun struct {
	bundle  *profile.Bundle
	session *qa.Session
	command string
	d       *qa.Draft
}

func (r *qaRun) about() docAbout {
	return docAbout{noun: "Consolidado", recent: "consolidados QA previos", dest: r.bundle.Profile.Docs.Provider, command: r.command}
}

func (r *qaRun) updating() bool { return r.session.Existing != nil }

func (r *qaRun) draft(ctx context.Context) (*docDraft, error) {
	var err error
	if r.updating() {
		r.d, err = r.session.DraftUpdate(ctx)
	} else {
		r.d, err = r.session.DraftCreate()
	}
	if err != nil {
		return nil, err
	}
	return &docDraft{title: r.d.Title, markdown: r.d.Markdown, replaces: r.d.Replaces, notPreserved: r.d.NotPreserved,
		preserve: r.session.Target.Template.Manifest.Preserve}, nil
}

func (r *qaRun) locations(ctx context.Context) (*docLocations, error) {
	docs, err := r.session.Locations(ctx)
	if errors.Is(err, qa.ErrNoLocations) {
		return nil, fmt.Errorf("%w (por ejemplo «%s»)", err, r.session.Target.Title)
	}
	if err != nil {
		return nil, err
	}
	return &docLocations{docs: docs}, nil
}

func (r *qaRun) publish(ctx context.Context, sibling *provider.DocRef) (*docResult, error) {
	res, err := r.session.Publish(ctx, r.d, sibling)
	if err != nil {
		return nil, err
	}
	return &docResult{page: res.Page, updated: res.Updated}, nil
}
