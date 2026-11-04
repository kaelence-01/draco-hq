package atlassian

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

// devSummaryType is the schema of Jira Cloud's Development field, whatever
// its name in the site's language.
const devSummaryType = "com.atlassian.jira.plugins.jira-development-integration-plugin:devsummarycf"

var categoryRank = map[string]int{"new": 0, "indeterminate": 1, "done": 2}

// ProjectStatuses returns the statuses of every issue type of the project,
// once each, ordered by category (to do, in progress, done).
func (t *Tracker) ProjectStatuses(ctx context.Context, project string) ([]provider.Status, error) {
	body, err := t.c.Get(ctx, "/rest/api/3/project/"+url.PathEscape(project)+"/statuses")
	if err != nil {
		return nil, fmt.Errorf("no se pudieron leer los estados del proyecto '%s': %w", project, err)
	}
	return parseProjectStatuses(body)
}

func parseProjectStatuses(body []byte) ([]provider.Status, error) {
	var types []struct {
		Statuses []struct {
			ID             string `json:"id"`
			Name           string `json:"name"`
			StatusCategory struct {
				Key string `json:"key"`
			} `json:"statusCategory"`
		} `json:"statuses"`
	}
	if err := json.Unmarshal(body, &types); err != nil {
		return nil, fmt.Errorf("respuesta inesperada de Jira con los estados: %w", err)
	}
	var out []provider.Status
	rank := map[string]int{}
	for _, it := range types {
		for _, s := range it.Statuses {
			if _, seen := rank[s.ID]; seen {
				continue
			}
			rank[s.ID] = categoryRank[s.StatusCategory.Key]
			out = append(out, provider.Status{ID: s.ID, Name: s.Name, Done: s.StatusCategory.Key == "done"})
		}
	}
	slices.SortStableFunc(out, func(a, b provider.Status) int { return rank[a.ID] - rank[b.ID] })
	return out, nil
}

// QAFields returns the custom number fields and the Development field.
func (t *Tracker) QAFields(ctx context.Context) ([]provider.Field, error) {
	body, err := t.c.Get(ctx, "/rest/api/3/field")
	if err != nil {
		return nil, fmt.Errorf("no se pudieron leer los campos de Jira: %w", err)
	}
	return parseQAFields(body)
}

func parseQAFields(body []byte) ([]provider.Field, error) {
	var fields []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Custom bool   `json:"custom"`
		Schema struct {
			Type   string `json:"type"`
			Custom string `json:"custom"`
		} `json:"schema"`
	}
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, fmt.Errorf("respuesta inesperada de Jira con los campos: %w", err)
	}
	var out []provider.Field
	for _, f := range fields {
		switch {
		case !f.Custom:
		case f.Schema.Custom == devSummaryType:
			out = append(out, provider.Field{ID: f.ID, Name: f.Name, Kind: provider.FieldPRStatus})
		case f.Schema.Type == "number":
			out = append(out, provider.Field{ID: f.ID, Name: f.Name, Kind: provider.FieldNumber})
		}
	}
	slices.SortStableFunc(out, func(a, b provider.Field) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	return out, nil
}
