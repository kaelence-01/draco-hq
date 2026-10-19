package atlassian

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Issue is the part of a Jira issue draco uses.
type Issue struct {
	Key     string
	Summary string
	URL     string
}

type jiraIssueResponse struct {
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
	} `json:"fields"`
}

// JiraUser is the authenticated user, as /rest/api/3/myself returns it.
type JiraUser struct {
	AccountID    string `json:"accountId"`
	EmailAddress string `json:"emailAddress"`
	DisplayName  string `json:"displayName"`
}

// Whoami returns the authenticated user. Basic Auth with a wrong email returns
// empty results instead of failing, so callers verify the account early.
func (c *Client) Whoami() (*JiraUser, error) {
	body, err := c.Get("/rest/api/3/myself")
	if err != nil {
		return nil, fmt.Errorf("no se pudo verificar credenciales: %w", err)
	}
	var u JiraUser
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, fmt.Errorf("respuesta inválida de /myself: %w", err)
	}
	return &u, nil
}

// VerifyCredentialsMatch calls Whoami and reports through warn when the
// authenticated email differs from configuredEmail; it errors only if the call fails.
func (c *Client) VerifyCredentialsMatch(configuredEmail string, warn func(string)) (*JiraUser, error) {
	u, err := c.Whoami()
	if err != nil {
		return nil, err
	}
	if configuredEmail != "" && !strings.EqualFold(u.EmailAddress, configuredEmail) {
		if warn != nil {
			warn(fmt.Sprintf(
				"el email del config (%s) no coincide con la cuenta autenticada (%s). "+
					"Búsquedas pueden devolver resultados vacíos. Revisa ~/.config/draco/config.yaml o ejecuta: draco init",
				configuredEmail, u.EmailAddress,
			))
		}
	}
	return u, nil
}

// CreateJiraRemoteLink links the issue to a Confluence page. The globalId makes
// it idempotent: a rerun updates the link instead of duplicating it.
func (c *Client) CreateJiraRemoteLink(issueKey, pageID, pageURL, pageTitle string) error {
	payload := map[string]any{
		"globalId":     "confluence-page:" + pageID,
		"relationship": "mentioned in",
		"object": map[string]any{
			"url":   pageURL,
			"title": pageTitle,
		},
	}
	_, err := c.Post(fmt.Sprintf("/rest/api/3/issue/%s/remotelink", issueKey), payload)
	return err
}

// GetIssue returns a Jira issue by key.
func (c *Client) GetIssue(issueKey string) (*Issue, error) {
	path := fmt.Sprintf("/rest/api/3/issue/%s?fields=summary", issueKey)

	body, err := c.Get(path)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo issue %s: %w", issueKey, err)
	}

	var resp jiraIssueResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parseando issue: %w", err)
	}

	return &Issue{
		Key:     resp.Key,
		Summary: resp.Fields.Summary,
		URL:     fmt.Sprintf("%s/browse/%s", c.BaseURL, resp.Key),
	}, nil
}
