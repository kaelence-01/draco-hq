package atlassian

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// QASchema maps the QA report onto one Jira instance. It comes from config
// (qa_jira), never from code (P-008).
type QASchema struct {
	Project       string
	Statuses      []string // ids or names
	TestingStatus string   // status whose transition opens the Kanban window

	FieldCodingErrors string // number: errors found in review
	FieldDevReturns   string // number: times sent back to dev
	FieldDevStatus    string // Development panel: PR state
}

// statusList renders a JQL "status in (...)" list: ids bare, names quoted.
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

// QAIssue is one task of the QA report with its evaluation.
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

	// DeployDocUnknown is set when the lookup failed: the report is evidence,
	// so an error must never render as a missing document.
	DeployDocUnknown bool
}

type jiraSearchResponse struct {
	Issues []struct {
		Key    string         `json:"key"`
		Fields map[string]any `json:"fields"`
	} `json:"issues"`
}

// GetQATasksForReview returns the module's sprint dev tasks that reached QA,
// excluding the QA review tasks themselves.
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

// devTasksOnly drops the QA review tasks ("Revisión de Tarea - …", "QA …").
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

// GetQATasksAsAssignee returns the sprint tasks assigned to qaEmail, or to
// currentUser() when it is empty or cannot be resolved.
func (c *Client) GetQATasksAsAssignee(s QASchema, sprintName, qaEmail string) ([]QAIssue, error) {
	return c.searchQAIssues(s, sprintAssigneeJQL(s, sprintName, c.assigneeClause(qaEmail)))
}

func sprintAssigneeJQL(s QASchema, sprintName, assignee string) string {
	return fmt.Sprintf(
		`project = %s AND sprint = %s AND assignee = %s ORDER BY key ASC`,
		quoteLiteral(s.Project), quoteLiteral(sprintName), assignee,
	)
}

// assigneeClause returns qaEmail's accountId quoted (Jira Cloud JQL does not
// accept emails), or currentUser().
func (c *Client) assigneeClause(qaEmail string) string {
	if qaEmail != "" {
		if accountID, err := c.resolveAccountID(qaEmail); err == nil && accountID != "" {
			return quoteLiteral(accountID)
		}
	}
	return "currentUser()"
}

// BuildReviewMap maps each dev task key to its QA review task
// ("Revisión de Tarea - OPS-7").
func BuildReviewMap(qaTasks []QAIssue) map[string]QAIssue {
	m := make(map[string]QAIssue, len(qaTasks))
	for _, t := range qaTasks {
		if key := parseDevTaskKey(t.Summary); key != "" {
			m[key] = t
		}
	}
	return m
}

var issueKeyRe = regexp.MustCompile(`[A-Z][A-Z0-9]*-\d+`)

// parseDevTaskKey returns the first issue key in a review task summary: the
// prefix never has one, and a trailing note may mention another key.
func parseDevTaskKey(summary string) string {
	return issueKeyRe.FindString(strings.ToUpper(summary))
}

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

// GetNovedadComment returns the text after "Novedad::" in the issue's comments,
// joined with "; ". An error means unknown, not "no observations".
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

// businessDaysAgo returns the date n weekdays before from.
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

// KanbanWindow returns the start date (10 business days before now) and a
// display label like "08/05 al 22/05/2026".
func KanbanWindow(now time.Time) (since time.Time, period string) {
	since = businessDaysAgo(10, now)
	period = fmt.Sprintf("%s al %s", since.Format("02/01"), now.Format("02/01/2006"))
	return
}

// GetQATasksForReviewKanban returns the dev tasks of any module that moved to
// the testing status after sinceDate (YYYY-MM-DD).
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

// GetQATasksAsAssigneeKanban returns the QA user's tasks updated since sinceDate.
func (c *Client) GetQATasksAsAssigneeKanban(s QASchema, sinceDate, qaEmail string) ([]QAIssue, error) {
	return c.searchQAIssues(s, kanbanAssigneeJQL(s, sinceDate, c.assigneeClause(qaEmail)))
}

func kanbanAssigneeJQL(s QASchema, sinceDate, assignee string) string {
	return fmt.Sprintf(
		`project = %s AND assignee = %s AND updated >= %s ORDER BY key ASC`,
		quoteLiteral(s.Project), assignee, quoteLiteral(sinceDate),
	)
}

// HasDeployDocLink reports whether the issue links a deploy document. An error
// means "could not check", not "no document".
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
