// Package history is the local record of the deploy documents draco
// published, newest first: it is read without the network.
package history

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/deploy"
)

// Limit is how many documents the history keeps.
const Limit = 200

// Entry is a published document, as it was the last time draco published it.
type Entry struct {
	Issue   string    `json:"issue"`
	Title   string    `json:"title"`
	URL     string    `json:"url"`
	DocID   string    `json:"doc_id"`
	Dest    string    `json:"dest"` // the destination provider: confluence, markdown
	Project string    `json:"project,omitempty"`
	Profile string    `json:"profile,omitempty"`
	Updated bool      `json:"updated"`
	At      time.Time `json:"at"`

	// LinkFailed means the task does not link to the document, so the QA
	// report will mark it as undocumented.
	LinkFailed bool `json:"link_failed,omitempty"`
	// NotPreserved names the hand-written sections that went out as the
	// template has them.
	NotPreserved []string `json:"not_preserved,omitempty"`
}

// Deploy is the entry of a document published by the deploy flow.
func Deploy(t *deploy.Target, d *deploy.Draft, res *deploy.Result, profile, dest string, at time.Time) Entry {
	return Entry{Issue: t.Issue, Title: d.Title, URL: res.Page.URL, DocID: res.Page.ID, Dest: dest, Project: t.ProjectName,
		Profile: profile, Updated: res.Updated, At: at, LinkFailed: res.LinkErr != nil, NotPreserved: d.NotPreserved}
}

// Path is history.json, next to config.yaml.
func Path() (string, error) {
	cfg, err := config.ConfigPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(cfg), "history.json"), nil
}

// Load reads the history; without a file it is empty.
func Load(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el historial: %w", err)
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("el historial está dañado; bórralo para empezar de nuevo (%s): %w", path, err)
	}
	return entries, nil
}

// Record adds e to the history in path. A history that cannot be read is not
// overwritten.
func Record(path string, e Entry) error {
	entries, err := Load(path)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(Add(entries, e), "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("no se pudo guardar el historial: %w", err)
	}
	// A temporary file renamed over the history: a cut never leaves it half written.
	tmp, err := os.CreateTemp(filepath.Dir(path), "history-*.json")
	if err != nil {
		return fmt.Errorf("no se pudo guardar el historial: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("no se pudo guardar el historial: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("no se pudo guardar el historial: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("no se pudo guardar el historial: %w", err)
	}
	return nil
}

// Add puts e first, drops the older entry of the same document and keeps the
// newest Limit.
func Add(entries []Entry, e Entry) []Entry {
	out := []Entry{e}
	for _, x := range entries {
		if !x.sameDocument(e) {
			out = append(out, x)
		}
	}
	return out[:min(len(out), Limit)]
}

func (e Entry) sameDocument(o Entry) bool {
	if e.Dest != o.Dest {
		return false
	}
	if e.DocID != "" || o.DocID != "" {
		return e.DocID == o.DocID
	}
	return e.URL == o.URL
}

// Of returns the entries of a project; empty is every one.
func Of(entries []Entry, project string) []Entry {
	if project == "" {
		return entries
	}
	var out []Entry
	for _, e := range entries {
		if e.Project == project {
			out = append(out, e)
		}
	}
	return out
}

// Projects names the projects in entries, sorted.
func Projects(entries []Entry) []string {
	var out []string
	for _, e := range entries {
		if e.Project != "" && !slices.Contains(out, e.Project) {
			out = append(out, e.Project)
		}
	}
	slices.Sort(out)
	return out
}
