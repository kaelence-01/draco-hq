package atlassian

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Page is the part of a Confluence page draco uses.
type Page struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	ParentID string `json:"parentId"`
	SpaceID  string `json:"spaceId"`
	Version  int
	WebURL   string
}

type searchResponse struct {
	Results []struct {
		Content struct {
			ID    string `json:"id"`
			Title string `json:"title"`
			Links struct {
				WebUI string `json:"webui"`
			} `json:"_links"`
			Expandable struct {
				Container string `json:"container"`
			} `json:"_expandable"`
		} `json:"content"`
	} `json:"results"`
}

type pageResponse struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	ParentID string `json:"parentId"`
	SpaceID  string `json:"spaceId"`
	AuthorID string `json:"authorId"`
	Version  struct {
		Number int `json:"number"`
	} `json:"version"`
	Links struct {
		WebUI string `json:"webui"`
	} `json:"_links"`
}

// recentPagesLimit caps the pages offered as a location.
const recentPagesLimit = 10

// ResolveSpaceID returns the numeric space ID of a space key.
func (c *Client) ResolveSpaceID(ctx context.Context, spaceKey string) (string, error) {
	path := fmt.Sprintf("/wiki/api/v2/spaces?keys=%s&limit=1", url.QueryEscape(spaceKey))
	body, err := c.Get(ctx, path)
	if err != nil {
		return "", fmt.Errorf("error resolviendo space %q: %w", spaceKey, err)
	}

	var result struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("error parseando space: %w", err)
	}
	if len(result.Results) == 0 {
		return "", fmt.Errorf("no se encontró el space %q en Confluence", spaceKey)
	}
	return result.Results[0].ID, nil
}

// FindRecentPages returns the newest pages of the space whose title starts with
// prefix, read from the v2 pages API (no CQL index lag). With an accountID it
// returns only that user's, falling back to the team's with fellBack set when
// the user has none.
func (c *Client) FindRecentPages(ctx context.Context, spaceKey, prefix, accountID string) (pages []Page, fellBack bool, err error) {
	spaceID, err := c.ResolveSpaceID(ctx, spaceKey)
	if err != nil {
		return nil, false, err
	}

	// v2 cannot filter by title prefix: over-fetch and filter here.
	path := fmt.Sprintf("/wiki/api/v2/pages?space-id=%s&sort=-created-date&limit=100", url.QueryEscape(spaceID))

	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, false, fmt.Errorf("error buscando documentos: %w", err)
	}

	var result struct {
		Results []pageResponse `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, false, fmt.Errorf("error parseando respuesta: %w", err)
	}

	all := make([]Page, 0, recentPagesLimit)
	mine := make([]Page, 0, recentPagesLimit)
	for _, r := range result.Results {
		if !strings.HasPrefix(r.Title, prefix) {
			continue
		}
		p := Page{
			ID:       r.ID,
			Title:    r.Title,
			ParentID: r.ParentID,
			SpaceID:  r.SpaceID,
			Version:  r.Version.Number,
			WebURL:   c.BaseURL + "/wiki" + r.Links.WebUI,
		}
		if len(all) < recentPagesLimit {
			all = append(all, p)
		}
		if accountID != "" && r.AuthorID == accountID && len(mine) < recentPagesLimit {
			mine = append(mine, p)
		}
	}

	if accountID != "" && len(mine) > 0 {
		return mine, false, nil
	}
	if accountID != "" && len(mine) == 0 {
		return all, true, nil
	}
	return all, false, nil
}

// PageADF is a page with its body parsed as ADF. ADF is nil when the body is in
// another representation (legacy editor), so the caller can warn.
type PageADF struct {
	Page
	ADF map[string]any
}

// pageADFResponse carries the ADF as a JSON string, which needs a second unmarshal.
type pageADFResponse struct {
	pageResponse
	Body struct {
		ADF struct {
			Value string `json:"value"`
		} `json:"atlas_doc_format"`
	} `json:"body"`
}

// GetPageADF returns a page by ID with its body parsed as ADF, to carry
// hand-edited sections across an update (which replaces the whole body).
func (c *Client) GetPageADF(ctx context.Context, pageID string) (*PageADF, error) {
	path := fmt.Sprintf("/wiki/api/v2/pages/%s?body-format=atlas_doc_format", pageID)

	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo página %s: %w", pageID, err)
	}

	var resp pageADFResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parseando página: %w", err)
	}

	out := &PageADF{
		Page: Page{
			ID:       resp.ID,
			Title:    resp.Title,
			ParentID: resp.ParentID,
			SpaceID:  resp.SpaceID,
			Version:  resp.Version.Number,
			WebURL:   c.BaseURL + "/wiki" + resp.Links.WebUI,
		},
	}

	if raw := strings.TrimSpace(resp.Body.ADF.Value); raw != "" {
		var doc map[string]any
		if err := json.Unmarshal([]byte(raw), &doc); err != nil {
			return nil, fmt.Errorf("error parseando el ADF de la página %s: %w", pageID, err)
		}
		out.ADF = doc
	}

	return out, nil
}

// GetPage returns a page by ID, including its parent and space.
func (c *Client) GetPage(ctx context.Context, pageID string) (*Page, error) {
	path := fmt.Sprintf("/wiki/api/v2/pages/%s", pageID)

	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo página %s: %w", pageID, err)
	}

	var resp pageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parseando página: %w", err)
	}

	return &Page{
		ID:       resp.ID,
		Title:    resp.Title,
		ParentID: resp.ParentID,
		SpaceID:  resp.SpaceID,
		Version:  resp.Version.Number,
		WebURL:   c.BaseURL + "/wiki" + resp.Links.WebUI,
	}, nil
}

// FindPageByTitleKey searches a page whose title contains both prefix and key.
// It uses CQL, so it is subject to index lag. An empty spaceKey searches all
// spaces.
func (c *Client) FindPageByTitleKey(ctx context.Context, prefix, key, spaceKey string) (*Page, error) {
	cqlBase := fmt.Sprintf(`title ~ %s AND title ~ %s`,
		quoteLiteral(prefix), quoteLiteral(key))
	if spaceKey != "" {
		cqlBase += fmt.Sprintf(` AND space = %s`, quoteLiteral(spaceKey))
	}
	path := fmt.Sprintf("/wiki/rest/api/search?cql=%s&limit=1", url.QueryEscape(cqlBase))

	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("error buscando documento existente: %w", err)
	}

	var result searchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parseando respuesta: %w", err)
	}

	if len(result.Results) == 0 {
		return nil, nil
	}

	r := result.Results[0]
	return &Page{
		ID:     r.Content.ID,
		Title:  r.Content.Title,
		WebURL: c.BaseURL + "/wiki" + r.Content.Links.WebUI,
	}, nil
}

// findPageByExactTitle looks up a page by exact title in the v2 API: no index
// lag, and the same uniqueness rule Confluence applies on create. An empty
// spaceKey searches all spaces; otherwise it is resolved to a space-id, since v2
// silently ignores an unknown filter and would match another space's page.
func (c *Client) findPageByExactTitle(ctx context.Context, title, spaceKey string) (*Page, error) {
	path := fmt.Sprintf("/wiki/api/v2/pages?title=%s&limit=1", url.QueryEscape(title))
	if spaceKey != "" {
		spaceID, err := c.ResolveSpaceID(ctx, spaceKey)
		if err != nil {
			return nil, err
		}
		path += "&space-id=" + url.QueryEscape(spaceID)
	}

	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("error buscando documento existente: %w", err)
	}

	var result struct {
		Results []pageResponse `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parseando respuesta: %w", err)
	}

	if len(result.Results) == 0 {
		return nil, nil
	}

	r := result.Results[0]
	return &Page{
		ID:       r.ID,
		Title:    r.Title,
		ParentID: r.ParentID,
		SpaceID:  r.SpaceID,
		Version:  r.Version.Number,
		WebURL:   c.BaseURL + "/wiki" + r.Links.WebUI,
	}, nil
}

// FindPageByTitle looks up a page by its exact title, with no index lag.
func (c *Client) FindPageByTitle(ctx context.Context, title, spaceKey string) (*Page, error) {
	return c.findPageByExactTitle(ctx, title, spaceKey)
}

// CreatePage creates a new Confluence page under the given parent.
func (c *Client) CreatePage(ctx context.Context, spaceID, parentID, title string, adfBody map[string]any) (*Page, error) {
	adfStr, err := marshalADF(adfBody)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"spaceId":  spaceID,
		"parentId": parentID,
		"title":    title,
		"status":   "current",
		"body": map[string]any{
			"representation": "atlas_doc_format",
			"value":          adfStr,
		},
	}

	body, err := c.Post(ctx, "/wiki/api/v2/pages", payload)
	if err != nil {
		return nil, fmt.Errorf("error creando página: %w", err)
	}

	var resp pageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parseando respuesta de creación: %w", err)
	}

	return &Page{
		ID:     resp.ID,
		Title:  resp.Title,
		WebURL: c.BaseURL + "/wiki" + resp.Links.WebUI,
	}, nil
}

// UpdatePage replaces the whole body of a page; currentVersion is the one read.
func (c *Client) UpdatePage(ctx context.Context, pageID, title string, currentVersion int, adfBody map[string]any) (*Page, error) {
	adfStr, err := marshalADF(adfBody)
	if err != nil {
		return nil, err
	}

	payload := map[string]any{
		"id":     pageID,
		"status": "current",
		"title":  title,
		"version": map[string]any{
			"number":  currentVersion + 1,
			"message": "Actualizado vía draco",
		},
		"body": map[string]any{
			"representation": "atlas_doc_format",
			"value":          adfStr,
		},
	}

	body, err := c.Put(ctx, fmt.Sprintf("/wiki/api/v2/pages/%s", pageID), payload)
	if err != nil {
		return nil, fmt.Errorf("error actualizando página: %w", err)
	}

	var resp pageResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("error parseando respuesta de actualización: %w", err)
	}

	return &Page{
		ID:     resp.ID,
		Title:  resp.Title,
		WebURL: c.BaseURL + "/wiki" + resp.Links.WebUI,
	}, nil
}

// FindQAPagesForModule returns recent QA reports of a module, or of any module
// when module is empty. An empty spaceKey searches all spaces.
func (c *Client) FindQAPagesForModule(ctx context.Context, module, spaceKey string) ([]Page, error) {
	title := "Consolidado de Pruebas QA"
	if module != "" {
		title += " - " + module
	}
	titleFilter := quoteLiteral(title)
	cqlBase := fmt.Sprintf(`title ~ %s ORDER BY created DESC`, titleFilter)
	if spaceKey != "" {
		cqlBase = fmt.Sprintf(`title ~ %s AND space = %s ORDER BY created DESC`,
			titleFilter, quoteLiteral(spaceKey))
	}
	path := fmt.Sprintf("/wiki/rest/api/search?cql=%s&limit=5", url.QueryEscape(cqlBase))

	body, err := c.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("error buscando páginas QA de referencia: %w", err)
	}

	var result searchResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error parseando respuesta: %w", err)
	}

	pages := make([]Page, 0, len(result.Results))
	for _, r := range result.Results {
		pages = append(pages, Page{
			ID:     r.Content.ID,
			Title:  r.Content.Title,
			WebURL: c.BaseURL + "/wiki" + r.Content.Links.WebUI,
		})
	}
	return pages, nil
}

func marshalADF(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("error serializando ADF: %w", err)
	}
	return string(b), nil
}
