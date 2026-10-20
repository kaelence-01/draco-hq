// Package document builds draco's documents as Markdown; each destination
// adapter converts it to its own format.
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

// ConsiderHeading opens the section of the deploy document the deployment team
// edits by hand; the destination keeps it on update.
const ConsiderHeading = "A considerar"

// DeployTitlePrefix starts every deploy document title; searches rely on it.
const DeployTitlePrefix = "Documento de Despliegue"

// DeployMarkdown builds the deploy document.
func DeployMarkdown(doc DeployDoc) string {
	var b strings.Builder
	b.WriteString("|  |  |\n|---|---|\n")
	b.WriteString("| **Épica** | _Enlace a épica o función de Jira relacionada_ |\n")
	fmt.Fprintf(&b, "| **Tarea(s)** | %s |\n\n", card(doc.IssueURL))

	b.WriteString("## Arquitecturas e interfaces\n\n")
	if len(doc.FrontendFiles) > 0 {
		b.WriteString("### Proyectos y formularios - Frontend:\n\n")
		filesTable(&b, doc.FrontendRepo, doc.FrontendCommit, doc.FrontendFiles, doc.VCSHost, doc.VCSOrg)
	}
	if len(doc.BackendFiles) > 0 {
		b.WriteString("### Proyectos y formularios - Backend:\n\n")
		filesTable(&b, doc.BackendRepo, doc.BackendCommit, doc.BackendFiles, doc.VCSHost, doc.VCSOrg)
	}

	fmt.Fprintf(&b, "## %s:\n\n", ConsiderHeading)
	checklist := doc.Checklist
	if len(checklist) == 0 {
		checklist = defaultChecklist
	}
	for _, step := range checklist {
		if step = strings.TrimSpace(step); step != "" {
			fmt.Fprintf(&b, "- [ ] %s\n", mdText(step))
		}
	}
	return b.String()
}

// BuildTitle builds the page title, replacing characters Confluence rejects.
func BuildTitle(issueKey, summary string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		":", "-",
		"|", "-",
		"[", "(",
		"]", ")",
	)
	sanitized := strings.TrimSpace(replacer.Replace(summary))
	return fmt.Sprintf("%s - %s - %s", DeployTitlePrefix, issueKey, sanitized)
}

func filesTable(b *strings.Builder, repoName, commitHash string, files map[string][]string, vcsHost, vcsOrg string) {
	b.WriteString("| Servidor | Aplicación web | Ubicación | Nombre del archivo | Observación |\n|---|---|---|---|---|\n")

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

	for _, dir := range dirs {
		names := make([]string, 0, len(files[dir]))
		links := make([]string, 0, len(files[dir]))
		for _, name := range files[dir] {
			path := dir + "/" + name
			if dir == "." {
				path = name
			}
			names = append(names, mdText(name))
			if u := commitFileURL(vcsHost, repoURL, commitHash, path); u != "" {
				links = append(links, link("link", u))
			} else {
				links = append(links, "—")
			}
		}
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n", server, mdText(repoName), mdText(dir), strings.Join(names, "<br>"), strings.Join(links, "<br>"))
	}
	b.WriteString("\n")
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
