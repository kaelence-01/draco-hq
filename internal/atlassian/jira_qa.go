package atlassian

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// QASchema maps the QA consolidated report onto one Jira instance: the project
// key, the workflow statuses of a task that reached QA, and the custom fields
// the report reads. All of it belongs to the user's instance and comes from
// config (qa_jira); nothing here is compiled in. See
// docs/security/patrones-seguros.md (P-008).
type QASchema struct {
	Project       string
	Statuses      []string // ids or names
	TestingStatus string   // status whose transition opens the Kanban window

	FieldCodingErrors string // number: errors found in review
	FieldDevReturns   string // number: times sent back to dev
	FieldDevStatus    string // Development panel: PR state
}

// statusList renders the statuses for a JQL "status in (...)" clause. A
// numeric id goes bare, as Jira expects; anything else is a status name and is
// quoted as a literal.
func (s QASchema) statusList() string {
	parts := make([]string, len(s.Statuses))
	for i, st := range s.Statuses {
		st = strings.TrimSpace(st)
		if numericRe.MatchString(st) {
			parts[i] = st
		} else {
			parts[i] = quoteLiteral(st)
		}
	}
	return strings.Join(parts, ", ")
}

var numericRe = regexp.MustCompile(`^\d+$`)

// QAIssue holds the evaluation data for one task in the QA consolidated report.
type QAIssue struct {
	Key             string
	URL             string
	Summary         string
	HasCodingErrors bool   // FieldCodingErrors > 0
	HasDevReturns   bool   // FieldDevReturns > 0
	HasDeployDoc    bool   // has a remote link titled "Documento de Despliegue..."
	PRMerged        bool   // PR state = MERGED (FieldDevStatus)
	Observations    string // text after "::" in comments matching "Novedad::"
	ReviewTaskKey   string // QA review task key (e.g. APP-2160)
	ReviewTaskURL   string // QA review task URL

	// DeployDocUnknown is set when the remote-link lookup failed, so the
	// report can say "unverified" instead of claiming the document is
	// missing. This table is published as QA evidence: a network or
	// permission error must never be rendered as a failed check.
	DeployDocUnknown bool
}

type jiraSearchResponse struct {
	Issues []struct {
		Key    string         `json:"key"`
		Fields map[string]any `json:"fields"`
	} `json:"issues"`
}

// GetQATasksForReview returns sprint dev tasks that have reached or passed
// through QA for the given module. QA review tasks themselves are excluded.
func (c *Client) GetQATasksForReview(s QASchema, sprintName, module string) ([]QAIssue, error) {
	all, err := c.searchQAIssues(s, sprintReviewJQL(s, sprintName, module))
	if err != nil {
		return nil, err
	}
	return devTasksOnly(all), nil
}

func sprintReviewJQL(s QASchema, sprintName, module string) string {
	return fmt.Sprintf(
		`project = %s AND sprint = %s AND status in (%s) AND component = %s ORDER BY key ASC`,
		quoteLiteral(s.Project), quoteLiteral(sprintName), s.statusList(), quoteLiteral(module),
	)
}

// devTasksOnly drops the QA review tasks ("Revisión de Tarea - …", "QA …")
// from a search result, leaving the development tasks under review.
func devTasksOnly(all []QAIssue) []QAIssue {
	dev := make([]QAIssue, 0, len(all))
	for _, t := range all {
		lower := strings.ToLower(t.Summary)
		if !strings.HasPrefix(lower, "revisión de tarea") &&
			!strings.HasPrefix(lower, "revision de tarea") &&
			!strings.HasPrefix(lower, "qa") {
			dev = append(dev, t)
		}
	}
	return dev
}

// GetQATasksAsAssignee returns sprint tasks assigned to the given email (or currentUser() if empty).
// Resolves the email to an accountId first since Jira Cloud JQL requires accountId, not email.
func (c *Client) GetQATasksAsAssignee(s QASchema, sprintName, qaEmail string) ([]QAIssue, error) {
	return c.searchQAIssues(s, sprintAssigneeJQL(s, sprintName, c.assigneeClause(qaEmail)))
}

func sprintAssigneeJQL(s QASchema, sprintName, assignee string) string {
	return fmt.Sprintf(
		`project = %s AND sprint = %s AND assignee = %s ORDER BY key ASC`,
		quoteLiteral(s.Project), quoteLiteral(sprintName), assignee,
	)
}

// assigneeClause returns the JQL value for the QA user: the accountId of
// qaEmail, already quoted, or currentUser() when there is no email or it
// cannot be resolved.
func (c *Client) assigneeClause(qaEmail string) string {
	if qaEmail != "" {
		if accountID, err := c.resolveAccountID(qaEmail); err == nil && accountID != "" {
			return quoteLiteral(accountID)
		}
	}
	return "currentUser()"
}

// BuildReviewMap builds a map of devTaskKey → QAIssue from a list of QA review tasks.
// QA review tasks follow the pattern "Revisión de Tarea - APP-XXXX".
func BuildReviewMap(qaTasks []QAIssue) map[string]QAIssue {
	m := make(map[string]QAIssue, len(qaTasks))
	for _, t := range qaTasks {
		if key := parseDevTaskKey(t.Summary); key != "" {
			m[key] = t
		}
	}
	return m
}

// issueKeyRe matches a Jira issue key: a project key followed by a number.
var issueKeyRe = regexp.MustCompile(`[A-Z][A-Z0-9]*-\d+`)

// parseDevTaskKey extracts the dev task key from a QA review task summary,
// e.g. "Revisión de Tarea - APP-1257" → "APP-1257".
//
// The first key in the summary wins: the descriptive prefix never contains one,
// so the first match is the task under review even when a trailing note
// mentions another key. Splitting on "-" instead — as this used to — broke on
// any summary with a suffix ("… - APP-1257 - ajuste" yielded "1257 - ajuste")
// and on an em dash.
func parseDevTaskKey(summary string) string {
	return issueKeyRe.FindString(strings.ToUpper(summary))
}

// resolveAccountID looks up the Jira accountId for a given email address.
func (c *Client) resolveAccountID(email string) (string, error) {
	body, err := c.Get("/rest/api/3/user/search?query=" + url.QueryEscape(email))
	if err != nil {
		return "", err
	}
	var users []struct {
		AccountID string `json:"accountId"`
	}
	if err := json.Unmarshal(body, &users); err != nil || len(users) == 0 {
		return "", fmt.Errorf("usuario no encontrado: %s", email)
	}
	return users[0].AccountID, nil
}

func (c *Client) searchQAIssues(s QASchema, jql string) ([]QAIssue, error) {
	payload := map[string]any{
		"jql":        jql,
		"fields":     []string{"key", "summary", s.FieldCodingErrors, s.FieldDevReturns, s.FieldDevStatus},
		"maxResults": 100,
	}

	body, err := c.Post("/rest/api/3/search/jql", payload)
	if err != nil {
		return nil, fmt.Errorf("error buscando issues: %w", err)
	}

	var resp jiraSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parseando respuesta: %w", err)
	}

	issues := make([]QAIssue, 0, len(resp.Issues))
	for _, raw := range resp.Issues {
		qi := QAIssue{
			Key: raw.Key,
			URL: fmt.Sprintf("%s/browse/%s", c.BaseURL, raw.Key),
		}
		if v, ok := raw.Fields["summary"]; ok && v != nil {
			if s, ok := v.(string); ok {
				qi.Summary = s
			}
		}
		if v, ok := raw.Fields[s.FieldCodingErrors]; ok && v != nil {
			if n, ok := v.(float64); ok && n > 0 {
				qi.HasCodingErrors = true
			}
		}
		if v, ok := raw.Fields[s.FieldDevReturns]; ok && v != nil {
			if n, ok := v.(float64); ok && n > 0 {
				qi.HasDevReturns = true
			}
		}
		if v, ok := raw.Fields[s.FieldDevStatus]; ok && v != nil {
			if s, ok := v.(string); ok {
				qi.PRMerged = strings.Contains(s, "state=MERGED")
			}
		}
		issues = append(issues, qi)
	}
	return issues, nil
}

// GetNovedadComment returns the observation text from comments matching "Novedad::texto".
// If multiple such comments exist they are joined with "; ".
// Returns an empty string and a nil error when the issue simply has none; an
// error means the comments could not be read and the result says nothing about
// whether observations exist.
func (c *Client) GetNovedadComment(issueKey string) (string, error) {
	body, err := c.Get(fmt.Sprintf("/rest/api/3/issue/%s/comment?orderBy=-created&maxResults=50", issueKey))
	if err != nil {
		return "", fmt.Errorf("error leyendo comentarios de %s: %w", issueKey, err)
	}

	var resp struct {
		Comments []struct {
			Body map[string]any `json:"body"`
		} `json:"comments"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("error parseando comentarios de %s: %w", issueKey, err)
	}

	var parts []string
	for _, comment := range resp.Comments {
		text := adfExtractText(comment.Body)
		if idx := strings.Index(strings.ToLower(text), "novedad::"); idx != -1 {
			observation := strings.TrimSpace(text[idx+len("novedad::"):])
			if observation != "" {
				parts = append(parts, observation)
			}
		}
	}
	return strings.Join(parts, "; "), nil
}

// adfExtractText recursively collects all plain text nodes from an ADF document.
func adfExtractText(node map[string]any) string {
	if node["type"] == "text" {
		if t, ok := node["text"].(string); ok {
			return t
		}
	}
	var sb strings.Builder
	if content, ok := node["content"].([]any); ok {
		for _, child := range content {
			if m, ok := child.(map[string]any); ok {
				sb.WriteString(adfExtractText(m))
			}
		}
	}
	return sb.String()
}

// businessDaysAgo returns the date N business days before from, skipping weekends.
func businessDaysAgo(n int, from time.Time) time.Time {
	d := from
	for count := 0; count < n; {
		d = d.AddDate(0, 0, -1)
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			count++
		}
	}
	return d
}

// KanbanWindow returns the start date (10 business days ago) and a display label like "08/05 al 22/05/2026".
func KanbanWindow() (since time.Time, period string) {
	now := time.Now()
	since = businessDaysAgo(10, now)
	period = fmt.Sprintf("%s al %s", since.Format("02/01"), now.Format("02/01/2006"))
	return
}

// GetQATasksForReviewKanban returns all Kanban dev tasks (any component) that
// transitioned to the testing status on or after sinceDate (format YYYY-MM-DD).
func (c *Client) GetQATasksForReviewKanban(s QASchema, sinceDate string) ([]QAIssue, error) {
	all, err := c.searchQAIssues(s, kanbanReviewJQL(s, sinceDate))
	if err != nil {
		return nil, err
	}
	return devTasksOnly(all), nil
}

func kanbanReviewJQL(s QASchema, sinceDate string) string {
	return fmt.Sprintf(
		`project = %s AND status in (%s) AND status CHANGED TO %s AFTER %s ORDER BY key ASC`,
		quoteLiteral(s.Project), s.statusList(), quoteLiteral(s.TestingStatus), quoteLiteral(sinceDate),
	)
}

// GetQATasksAsAssigneeKanban returns tasks assigned to the QA user updated on or after sinceDate.
func (c *Client) GetQATasksAsAssigneeKanban(s QASchema, sinceDate, qaEmail string) ([]QAIssue, error) {
	return c.searchQAIssues(s, kanbanAssigneeJQL(s, sinceDate, c.assigneeClause(qaEmail)))
}

func kanbanAssigneeJQL(s QASchema, sinceDate, assignee string) string {
	return fmt.Sprintf(
		`project = %s AND assignee = %s AND updated >= %s ORDER BY key ASC`,
		quoteLiteral(s.Project), assignee, quoteLiteral(sinceDate),
	)
}

// HasDeployDocLink returns true if the issue has a remote link titled
// "Documento de Despliegue...".
//
// A false with a non-nil error means "could not check", not "no document":
// the caller must distinguish them, because the answer lands in a QA report
// published as evidence. Returning false on error — as this used to — marked
// compliant tasks as missing their document whenever the API hiccuped.
func (c *Client) HasDeployDocLink(issueKey string) (bool, error) {
	body, err := c.Get(fmt.Sprintf("/rest/api/3/issue/%s/remotelink", issueKey))
	if err != nil {
		return false, fmt.Errorf("error leyendo enlaces remotos de %s: %w", issueKey, err)
	}

	var links []struct {
		Object struct {
			Title string `json:"title"`
		} `json:"object"`
	}
	if err := json.Unmarshal(body, &links); err != nil {
		return false, fmt.Errorf("error parseando enlaces remotos de %s: %w", issueKey, err)
	}

	for _, l := range links {
		if strings.HasPrefix(l.Object.Title, "Documento de Despliegue") {
			return true, nil
		}
	}
	return false, nil
}
