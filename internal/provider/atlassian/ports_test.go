package atlassian

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

func fakeConfluence(t *testing.T, routes map[string]func(w http.ResponseWriter, r *http.Request)) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, ok := routes[r.Method+" "+r.URL.Path]
		if !ok {
			t.Errorf("ruta inesperada: %s %s", r.Method, r.URL.Path)
			http.NotFound(w, r)
			return
		}
		h(w, r)
	}))
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, "qa@example.test", "token")
}

func writeJSON(w http.ResponseWriter, v any) { _ = json.NewEncoder(w).Encode(v) }

func TestDocStoreCreateUsesSiblingSpaceAndParent(t *testing.T) {
	var created map[string]any
	c := fakeConfluence(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /wiki/api/v2/pages/9": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, map[string]any{"id": "9", "spaceId": "S1", "parentId": "P1"})
		},
		"POST /wiki/api/v2/pages": func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &created)
			writeJSON(w, map[string]any{"id": "new", "title": "Doc", "_links": map[string]any{"webui": "/spaces/S/pages/new"}})
		},
	})

	ref, err := NewDocStore(c, "DEV").Create(context.Background(), provider.DocRef{ID: "9"}, "Doc", map[string]any{"type": "doc"})
	if err != nil {
		t.Fatal(err)
	}
	if created["spaceId"] != "S1" || created["parentId"] != "P1" || created["title"] != "Doc" {
		t.Errorf("creada con %v, esperaba S1/P1", created)
	}
	if ref.ID != "new" || ref.URL == "" {
		t.Errorf("referencia = %+v", ref)
	}
}

func TestDocStoreLocationsFiltersPrefixAndAuthor(t *testing.T) {
	pages := []map[string]any{
		{"id": "1", "title": "Documento de Despliegue - OPS-1 - a", "authorId": "otro"},
		{"id": "2", "title": "Otra página", "authorId": "yo"},
		{"id": "3", "title": "Documento de Despliegue - OPS-3 - b", "authorId": "yo"},
	}
	c := fakeConfluence(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /wiki/api/v2/spaces": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, map[string]any{"results": []any{map[string]any{"id": "100"}}})
		},
		"GET /wiki/api/v2/pages": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("space-id") != "100" {
				t.Errorf("v2 debe filtrar por space-id numérico, obtuve %q", r.URL.RawQuery)
			}
			writeJSON(w, map[string]any{"results": pages})
		},
	})
	docs := NewDocStore(c, "DEV")
	ctx := context.Background()

	mine, err := docs.Locations(ctx, provider.LocationQuery{TitlePrefix: "Documento de Despliegue", AuthorID: "yo"})
	if err != nil {
		t.Fatal(err)
	}
	if len(mine.Docs) != 1 || mine.Docs[0].ID != "3" || mine.TeamWide {
		t.Errorf("propios = %+v", mine)
	}

	team, err := docs.Locations(ctx, provider.LocationQuery{TitlePrefix: "Documento de Despliegue", AuthorID: "nadie"})
	if err != nil {
		t.Fatal(err)
	}
	if len(team.Docs) != 2 || !team.TeamWide {
		t.Errorf("sin documentos propios esperaba los del equipo, obtuve %+v", team)
	}
}

func TestDocStoreExportConvertsStorage(t *testing.T) {
	c := fakeConfluence(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /wiki/rest/api/content/42": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, map[string]any{
				"id": "42", "title": "Doc",
				"space":   map[string]any{"name": "Desarrollo"},
				"version": map[string]any{"number": 3},
				"body":    map[string]any{"storage": map[string]any{"value": "<p>Hola</p>"}},
			})
		},
	})
	e, err := NewDocStore(c, "").Export(context.Background(), provider.DocRef{ID: "42"})
	if err != nil {
		t.Fatal(err)
	}
	if e.Source != "Confluence" || e.Space != "Desarrollo" || e.Version != 3 || e.Text != "Hola\n" {
		t.Errorf("exportado = %+v", e)
	}
}
