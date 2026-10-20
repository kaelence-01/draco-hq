// Package document builds draco's documents as Markdown from templates; each
// destination adapter converts it to its own format.
package document

import (
	"fmt"
	"sort"
	"strings"
)

// DeployDoc is the data of the deploy document.
type DeployDoc struct {
	IssueKey       string
	IssueSummary   string
	IssueURL       string
	VCSHost        string // e.g. "https://bitbucket.org"
	VCSOrg         string // e.g. "mi-organizacion"
	BackendRepo    string
	BackendCommit  string
	BackendFiles   map[string][]string // dir -> []filename
	FrontendRepo   string
	FrontendCommit string
	FrontendFiles  map[string][]string // dir -> []filename

	// Checklist comes from config; empty means defaultChecklist.
	Checklist []string
}

// deployView is what the deploy template receives: every value is already
// escaped Markdown.
type deployView struct {
	Issue     string // the issue as a card, or "—"
	Sides     []sideView
	Checklist []string
}

type sideView struct {
	Label string
	Rows  []fileRow
}

type fileRow struct {
	Server, App, Dir string
	Files, Links     []string
}

// Deploy renders the deploy document.
func Deploy(t *Template, doc DeployDoc) (string, error) {
	v := deployView{Issue: card(doc.IssueURL)}
	if len(doc.FrontendFiles) > 0 {
		v.Sides = append(v.Sides, side("Frontend", doc.FrontendRepo, doc.FrontendCommit, doc.FrontendFiles, doc.VCSHost, doc.VCSOrg))
	}
	if len(doc.BackendFiles) > 0 {
		v.Sides = append(v.Sides, side("Backend", doc.BackendRepo, doc.BackendCommit, doc.BackendFiles, doc.VCSHost, doc.VCSOrg))
	}
	checklist := doc.Checklist
	if len(checklist) == 0 {
		checklist = defaultChecklist
	}
	for _, step := range checklist {
		if step = strings.TrimSpace(step); step != "" {
			v.Checklist = append(v.Checklist, mdText(step))
		}
	}
	return t.render(v)
}

// DeployTitle renders the deploy document title, replacing the characters
// Confluence rejects in titles.
func DeployTitle(t *Template, issueKey, summary string) (string, error) {
	replacer := strings.NewReplacer("/", "-", ":", "-", "|", "-", "[", "(", "]", ")")
	return t.renderTitle(struct{ IssueKey, Summary string }{issueKey, strings.TrimSpace(replacer.Replace(summary))})
}

func side(label, repoName, commitHash string, files map[string][]string, vcsHost, vcsOrg string) sideView {
	var repoURL string
	if vcsHost != "" && vcsOrg != "" {
		repoURL = fmt.Sprintf("%s/%s/%s", vcsHost, vcsOrg, repoName)
	}
	server := mdText(repoName)
	if repoURL != "" {
		server = link(repoName, repoURL)
	}

	dirs := make([]string, 0, len(files))
	for dir := range files {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	s := sideView{Label: label}
	for _, dir := range dirs {
		row := fileRow{Server: server, App: mdText(repoName), Dir: mdText(dir)}
		for _, name := range files[dir] {
			p := dir + "/" + name
			if dir == "." {
				p = name
			}
			row.Files = append(row.Files, mdText(name))
			if u := commitFileURL(vcsHost, repoURL, commitHash, p); u != "" {
				row.Links = append(row.Links, link("link", u))
			} else {
				row.Links = append(row.Links, "—")
			}
		}
		s.Rows = append(s.Rows, row)
	}
	return s
}

// commitFileURL links a file inside a commit, or "" for an unknown host (no
// link beats a dead one). GitHub's per-file anchor is a path digest, so it is
// omitted.
func commitFileURL(vcsHost, repoURL, commitHash, filePath string) string {
	if repoURL == "" || commitHash == "" {
		return ""
	}
	host := strings.ToLower(vcsHost)
	switch {
	case strings.Contains(host, "bitbucket"):
		return fmt.Sprintf("%s/commits/%s#chg-%s", repoURL, commitHash, filePath)
	case strings.Contains(host, "github"):
		return fmt.Sprintf("%s/commit/%s", repoURL, commitHash)
	case strings.Contains(host, "gitlab"):
		return fmt.Sprintf("%s/-/commit/%s", repoURL, commitHash)
	default:
		return ""
	}
}

// defaultChecklist is stack-agnostic on purpose; real steps go in deploy_checklist.
var defaultChecklist = []string{
	"Pasar backend al servidor",
	"Ejecutar migraciones",
	"Pasar frontend",
}
