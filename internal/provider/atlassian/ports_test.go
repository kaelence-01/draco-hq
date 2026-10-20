package atlassian

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/config"
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

	ref, err := NewDocStore(c, "DEV").Create(context.Background(), provider.DocRef{ID: "9"}, provider.Document{Title: "Doc", Markdown: "## Hola"})
	if err != nil {
		t.Fatal(err)
	}
	if created["spaceId"] != "S1" || created["parentId"] != "P1" || created["title"] != "Doc" {
		t.Errorf("creada con %v, esperaba S1/P1", created)
	}
	if body := sentADF(t, created); asJSON(body["content"]) != `[{"attrs":{"level":2},"content":[{"text":"Hola","type":"text"}],"type":"heading"}]` {
		t.Errorf("el Markdown debía llegar convertido a ADF: %s", asJSON(body))
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

func TestNewQATrackerConvertsConfig(t *testing.T) {
	q := &config.QAJiraConfig{
		Project:  "OPS",
		Statuses: []string{"10001"},
		Fields:   config.QAJiraFields{CodingErrors: "customfield_1", DevReturns: "customfield_2", DevStatus: "customfield_3"},
	}
	got := NewQATracker(NewClient("https://example.test", "", ""), q).schema
	want := QASchema{
		Project: "OPS", Statuses: []string{"10001"}, TestingStatus: config.DefaultTestingStatus, ObservationMarker: config.DefaultObservationMarker,
		FieldCodingErrors: "customfield_1", FieldDevReturns: "customfield_2", FieldDevStatus: "customfield_3",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("esquema = %+v, esperaba %+v", got, want)
	}
}

// sentADF decodes the ADF a create or update request carried as a string.
func sentADF(t *testing.T, payload map[string]any) map[string]any {
	t.Helper()
	var adf map[string]any
	value := payload["body"].(map[string]any)["value"].(string)
	if err := json.Unmarshal([]byte(value), &adf); err != nil {
		t.Fatalf("cuerpo enviado no es JSON: %v", err)
	}
	return adf
}

// publishedPage is a page as Confluence returns it, with its ADF as a string.
func publishedPage(t *testing.T, adf map[string]any) map[string]any {
	t.Helper()
	raw, _ := json.Marshal(adf)
	return map[string]any{"id": "42", "title": "Doc", "version": map[string]any{"number": 7},
		"body": map[string]any{"atlas_doc_format": map[string]any{"value": string(raw)}}}
}

func TestDocStoreUpdateKeepsHandEditedSections(t *testing.T) {
	current := MarkdownToADF("## Arquitecturas e interfaces\n\nviejo\n\n## A considerar:\n\n- [x] Paso marcado a mano\n- [ ] Paso agregado a mano\n")
	var sent map[string]any
	c := fakeConfluence(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /wiki/api/v2/pages/42": func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, publishedPage(t, current)) },
		"PUT /wiki/api/v2/pages/42": func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &sent)
			writeJSON(w, map[string]any{"id": "42", "title": "Doc"})
		},
	})
	doc := provider.Document{Title: "Doc", Markdown: "## Arquitecturas e interfaces\n\nnuevo\n\n## A considerar:\n\n- [ ] Paso de la plantilla\n", Preserve: []string{"A considerar"}}
	if _, err := NewDocStore(c, "DEV").Update(context.Background(), provider.DocRef{ID: "42"}, doc); err != nil {
		t.Fatal(err)
	}
	if v := sent["version"].(map[string]any)["number"]; v != float64(8) {
		t.Errorf("debe actualizar sobre la versión leída (7), envió %v", v)
	}
	got := asJSON(sentADF(t, sent))
	if !strings.Contains(got, "nuevo") || strings.Contains(got, "viejo") {
		t.Errorf("lo generado debe venir del Markdown nuevo: %s", got)
	}
	if !strings.Contains(got, `"state":"DONE"`) || !strings.Contains(got, "Paso agregado a mano") || strings.Contains(got, "Paso de la plantilla") {
		t.Errorf("«A considerar» debe conservarse tal cual estaba: %s", got)
	}
}

func TestDocStoreUpdateRefusesWhenPageUnreadable(t *testing.T) {
	c := fakeConfluence(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /wiki/api/v2/pages/42": func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "timeout", http.StatusGatewayTimeout) },
	})
	_, err := NewDocStore(c, "DEV").Update(context.Background(), provider.DocRef{ID: "42"}, provider.Document{Title: "Doc", Markdown: "x"})
	if err == nil || !strings.Contains(err.Error(), "no se sobrescribe") {
		t.Errorf("sin leer la página no se actualiza: %v", err)
	}
}

func TestDocStoreSectionsReadsLegacyObservationCell(t *testing.T) {
	legacy := MarkdownToADF("## Resumen\n\n| Total de Tareas | Observación |\n|---|---|\n| 3 | Sprint estable |\n\n### Consideraciones\n\n- algo\n")
	c := fakeConfluence(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /wiki/api/v2/pages/42": func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, publishedPage(t, legacy)) },
	})
	found, err := NewDocStore(c, "DEV").Sections(context.Background(), provider.DocRef{ID: "42"}, []string{"Observación", "Consideraciones", "Inexistente"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(found, []string{"Observación", "Consideraciones"}) {
		t.Errorf("secciones = %v", found)
	}
	if nodes, _ := preservedSection(legacy, "Observación"); !strings.Contains(asJSON(nodes), "Sprint estable") {
		t.Errorf("la observación vieja debe leerse de la celda: %s", asJSON(nodes))
	}
}

func TestQATrackerObservationsUseTheConfiguredMarker(t *testing.T) {
	comment := func(text string) map[string]any {
		return map[string]any{"body": map[string]any{"type": "doc", "content": []any{
			map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": text}}},
		}}}
	}
	c := fakeConfluence(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /rest/api/3/issue/OPS-7/comment": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, map[string]any{"comments": []any{comment("Obs: falta el monto"), comment("Novedad:: no aplica"), comment("sin marcador")}})
		},
	})
	q := &config.QAJiraConfig{Project: "OPS", ObservationMarker: "obs:"}
	got, err := NewQATracker(c, q).Observations(context.Background(), "OPS-7")
	if err != nil || got != "falta el monto" {
		t.Errorf("observaciones = %q, %v", got, err)
	}
}
