package atlassian

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

// QASchema maps the QA report onto one Jira instance. It comes from config
// (qa_jira), never from code (P-008).
type QASchema struct {
	Project       string
	Statuses      []string // ids or names
	TestingStatus string   // status whose transition opens the Kanban window

	ObservationMarker string // comment text that starts an observation

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

type jiraSearchResponse struct {
	Issues []struct {
		Key    string         `json:"key"`
		Fields map[string]any `json:"fields"`
	} `json:"issues"`
}

// GetQATasksForReview returns the module's sprint tasks that reached QA.
func (c *Client) GetQATasksForReview(ctx context.Context, s QASchema, sprintName, module string) ([]provider.QAIssue, error) {
	return c.searchQAIssues(ctx, s, sprintReviewJQL(s, sprintName, module))
}

func sprintReviewJQL(s QASchema, sprintName, module string) string {
	return fmt.Sprintf(
		`project = %s AND sprint = %s AND status in (%s) AND component = %s ORDER BY key ASC`,
		quoteLiteral(s.Project), quoteLiteral(sprintName), s.statusList(), quoteLiteral(module),
	)
}

// GetQATasksAsAssignee returns the sprint tasks assigned to qaEmail, or to
// currentUser() when it is empty or cannot be resolved.
func (c *Client) GetQATasksAsAssignee(ctx context.Context, s QASchema, sprintName, qaEmail string) ([]provider.QAIssue, error) {
	return c.searchQAIssues(ctx, s, sprintAssigneeJQL(s, sprintName, c.assigneeClause(ctx, qaEmail)))
}

func sprintAssigneeJQL(s QASchema, sprintName, assignee string) string {
	return fmt.Sprintf(
		`project = %s AND sprint = %s AND assignee = %s ORDER BY key ASC`,
		quoteLiteral(s.Project), quoteLiteral(sprintName), assignee,
	)
}

// assigneeClause returns qaEmail's accountId quoted (Jira Cloud JQL does not
// accept emails), or currentUser().
func (c *Client) assigneeClause(ctx context.Context, qaEmail string) string {
	if qaEmail != "" {
		if accountID, err := c.resolveAccountID(ctx, qaEmail); err == nil && accountID != "" {
			return quoteLiteral(accountID)
		}
	}
	return "currentUser()"
}

func (c *Client) resolveAccountID(ctx context.Context, email string) (string, error) {
	body, err := c.Get(ctx, "/rest/api/3/user/search?query="+url.QueryEscape(email))
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

func (c *Client) searchQAIssues(ctx context.Context, s QASchema, jql string) ([]provider.QAIssue, error) {
	payload := map[string]any{
		"jql":        jql,
		"fields":     []string{"key", "summary", s.FieldCodingErrors, s.FieldDevReturns, s.FieldDevStatus},
		"maxResults": 100,
	}

	body, err := c.Post(ctx, "/rest/api/3/search/jql", payload)
	if err != nil {
		return nil, fmt.Errorf("error buscando issues: %w", err)
	}

	var resp jiraSearchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parseando respuesta: %w", err)
	}

	issues := make([]provider.QAIssue, 0, len(resp.Issues))
	for _, raw := range resp.Issues {
		qi := provider.QAIssue{
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

// GetObservations returns the text after marker in the issue's comments,
// joined with "; ". An error means unknown, not "no observations".
func (c *Client) GetObservations(ctx context.Context, issueKey, marker string) (string, error) {
	body, err := c.Get(ctx, fmt.Sprintf("/rest/api/3/issue/%s/comment?orderBy=-created&maxResults=50", issueKey))
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

	m := strings.ToLower(marker)
	var parts []string
	for _, comment := range resp.Comments {
		text := adfExtractText(comment.Body)
		if idx := strings.Index(strings.ToLower(text), m); idx != -1 {
			if observation := strings.TrimSpace(text[idx+len(m):]); observation != "" {
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

// GetQATasksForReviewKanban returns the tasks of any module that moved to the
// testing status after sinceDate (YYYY-MM-DD).
func (c *Client) GetQATasksForReviewKanban(ctx context.Context, s QASchema, sinceDate string) ([]provider.QAIssue, error) {
	return c.searchQAIssues(ctx, s, kanbanReviewJQL(s, sinceDate))
}

func kanbanReviewJQL(s QASchema, sinceDate string) string {
	return fmt.Sprintf(
		`project = %s AND status in (%s) AND status CHANGED TO %s AFTER %s ORDER BY key ASC`,
		quoteLiteral(s.Project), s.statusList(), quoteLiteral(s.TestingStatus), quoteLiteral(sinceDate),
	)
}

// GetQATasksAsAssigneeKanban returns the QA user's tasks updated since sinceDate.
func (c *Client) GetQATasksAsAssigneeKanban(ctx context.Context, s QASchema, sinceDate, qaEmail string) ([]provider.QAIssue, error) {
	return c.searchQAIssues(ctx, s, kanbanAssigneeJQL(s, sinceDate, c.assigneeClause(ctx, qaEmail)))
}

func kanbanAssigneeJQL(s QASchema, sinceDate, assignee string) string {
	return fmt.Sprintf(
		`project = %s AND assignee = %s AND updated >= %s ORDER BY key ASC`,
		quoteLiteral(s.Project), assignee, quoteLiteral(sinceDate),
	)
}

// HasRemoteLink reports whether the issue has a remote link whose title starts
// with prefix. An error means "could not check", not "no link".
func (c *Client) HasRemoteLink(ctx context.Context, issueKey, prefix string) (bool, error) {
	body, err := c.Get(ctx, fmt.Sprintf("/rest/api/3/issue/%s/remotelink", issueKey))
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
		if strings.HasPrefix(l.Object.Title, prefix) {
			return true, nil
		}
	}
	return false, nil
}
