// Package fetch exports a Confluence page that mentions a Jira issue as plain
// text, with no terminal I/O. Steps: Search, then Export of the page picked.
package fetch

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/atlassian"
	"github.com/kaelence-01/draco-hq/internal/config"
)

// Atlassian is the part of the Confluence client the workflow uses.
type Atlassian interface {
	FindPagesByText(query, spaceKey string) ([]atlassian.Page, error)
	GetPageWithContent(pageID string) (*atlassian.PageContent, error)
}

var _ Atlassian = (*atlassian.Client)(nil)

// ErrNoPages means no Confluence page mentions the issue.
var ErrNoPages = errors.New("no se encontraron páginas en Confluence que mencionen el issue")

// Request is what the user asks for, before applying config.
type Request struct {
	Issue string
	Space string // overrides the global space key
}

// Target is a Request resolved against config.
type Target struct {
	Issue    string
	SpaceKey string // empty searches every space
}

// Resolve applies config to the request.
func Resolve(cfg *config.Config, req Request) (*Target, error) {
	issue := strings.ToUpper(strings.TrimSpace(req.Issue))
	if issue == "" {
		return nil, fmt.Errorf("falta la clave del issue de Jira")
	}
	space := req.Space
	if space == "" {
		space = cfg.ConfluenceSpaceKey
	}
	return &Target{Issue: issue, SpaceKey: space}, nil
}

// Session is one run of the workflow; the steps fill its exported fields.
type Session struct {
	api Atlassian

	Target *Target
	Pages  []atlassian.Page // most recently modified first
}

// NewSession starts a run for an already resolved target.
func NewSession(api Atlassian, t *Target) *Session {
	return &Session{api: api, Target: t}
}

// Search lists the pages that mention the issue.
func (s *Session) Search() error {
	pages, err := s.api.FindPagesByText(s.Target.Issue, s.Target.SpaceKey)
	if err != nil {
		return err
	}
	if len(pages) == 0 {
		return fmt.Errorf("%w: %s", ErrNoPages, s.Target.Issue)
	}
	s.Pages = pages
	return nil
}

// Export is a page converted to text, ready to save.
type Export struct {
	Page     *atlassian.PageContent
	Text     string
	FileName string // suggested name, without directory
}

// Export downloads the page and converts it to text.
func (s *Session) Export(page *atlassian.Page) (*Export, error) {
	if page == nil {
		return nil, fmt.Errorf("falta elegir la página a exportar")
	}
	content, err := s.api.GetPageWithContent(page.ID)
	if err != nil {
		return nil, err
	}
	return &Export{
		Page:     content,
		Text:     atlassian.BuildIssueTxt(s.Target.Issue, content),
		FileName: fileName(s.Target.Issue, content.Title),
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
