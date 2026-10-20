// Package fetch exports a document that mentions an issue as plain text, with
// no terminal I/O. Steps: Search, then Export of the document picked.
package fetch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

// ErrNoPages means no document mentions the issue.
var ErrNoPages = errors.New("no se encontraron documentos que mencionen el issue")

// Request is what the user asks for, before applying config.
type Request struct {
	Issue string
	Space string // overrides the global space key
}

// Target is a Request resolved against config.
type Target struct {
	Issue    string
	SpaceKey string // destination scope from the request; empty: the profile's, or everything
}

// Resolve applies config to the request.
func Resolve(cfg *config.Config, req Request) (*Target, error) {
	issue := strings.ToUpper(strings.TrimSpace(req.Issue))
	if issue == "" {
		return nil, fmt.Errorf("falta la clave del issue de Jira")
	}
	return &Target{Issue: issue, SpaceKey: req.Space}, nil
}

// Session is one run of the workflow; the steps fill its exported fields.
type Session struct {
	docs provider.DocStore

	Target *Target
	Pages  []provider.DocRef // most recently modified first
}

// NewSession starts a run for an already resolved target.
func NewSession(docs provider.DocStore, t *Target) *Session {
	return &Session{docs: docs, Target: t}
}

// Search lists the documents that mention the issue.
func (s *Session) Search(ctx context.Context) error {
	pages, err := s.docs.Search(ctx, s.Target.Issue)
	if err != nil {
		return err
	}
	if len(pages) == 0 {
		return fmt.Errorf("%w: %s", ErrNoPages, s.Target.Issue)
	}
	s.Pages = pages
	return nil
}

// Export is a document converted to text, ready to save.
type Export struct {
	Doc      *provider.Exported
	Text     string
	FileName string // suggested name, without directory
}

// Export downloads the document and converts it to text.
func (s *Session) Export(ctx context.Context, page *provider.DocRef) (*Export, error) {
	if page == nil {
		return nil, fmt.Errorf("falta elegir la página a exportar")
	}
	doc, err := s.docs.Export(ctx, *page)
	if err != nil {
		return nil, err
	}
	return &Export{
		Doc:      doc,
		Text:     issueText(s.Target.Issue, doc),
		FileName: fileName(s.Target.Issue, doc.Title),
	}, nil
}

// Save writes the text to path, replacing the file if it exists.
func (e *Export) Save(path string) error {
	if err := os.WriteFile(path, []byte(e.Text), 0644); err != nil {
		return fmt.Errorf("error escribiendo %s: %w", path, err)
	}
	return nil
}

const maxTitleInName = 40

// fileName builds "<ISSUE>_<title>.txt" with the title reduced to ASCII
// letters, digits and underscores.
func fileName(issue, title string) string {
	var sb strings.Builder
	for _, r := range title {
		if a, ok := accents[r]; ok {
			r = a
		}
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			sb.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			sb.WriteByte('_')
		}
	}
	s := sb.String()
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	s = strings.Trim(s, "_")
	if len(s) > maxTitleInName {
		s = strings.TrimRight(s[:maxTitleInName], "_")
	}
	if s == "" {
		return issue + ".txt"
	}
	return issue + "_" + s + ".txt"
}

var accents = map[rune]rune{
	'á': 'a', 'é': 'e', 'í': 'i', 'ó': 'o', 'ú': 'u', 'ü': 'u', 'ñ': 'n',
	'Á': 'A', 'É': 'E', 'Í': 'I', 'Ó': 'O', 'Ú': 'U', 'Ü': 'U', 'Ñ': 'N',
}

// issueText frames the document text with a header of its metadata.
func issueText(issue string, d *provider.Exported) string {
	var sb strings.Builder
	frame := strings.Repeat("=", 80)
	line := func(parts ...string) {
		for _, p := range parts {
			sb.WriteString(p)
		}
		sb.WriteByte('\n')
	}

	line(frame)
	line(issue, " - ", d.Title)
	line(frame)
	line("Fuente: ", d.Source, " - ", d.Space)
	line("URL: ", d.URL)
	if d.CreatedDate != "" {
		sb.WriteString("Creado: " + d.CreatedDate)
		if d.Version > 0 {
			fmt.Fprintf(&sb, " | Versión: %d", d.Version)
		}
		sb.WriteByte('\n')
	}
	if d.UpdatedBy != "" {
		line("Última edición: ", d.UpdatedBy)
	}
	line(strings.Repeat("-", 80))

	if d.Text != "" {
		sb.WriteByte('\n')
		sb.WriteString(d.Text)
	}

	sb.WriteByte('\n')
	line(frame)
	line("FIN DEL DOCUMENTO")
	line(frame)
	return sb.String()
}
