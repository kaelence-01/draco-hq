package atlassian

import (
	"context"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

func status(id, name, category string) map[string]any {
	return map[string]any{"id": id, "name": name, "statusCategory": map[string]any{"key": category}}
}

func TestProjectStatusesOncePerIDDoneLast(t *testing.T) {
	c := fakeConfluence(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /rest/api/3/project/OPS/statuses": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, []any{
				map[string]any{"name": "Tarea", "statuses": []any{
					status("3", "Finalizado", "done"), status("1", "Por hacer", "new"), status("2", "Testing", "indeterminate"),
				}},
				map[string]any{"name": "Error", "statuses": []any{
					status("2", "Testing", "indeterminate"), status("4", "Aprobado", "indeterminate"),
				}},
			})
		},
	})
	got, err := (&Tracker{c: c}).ProjectStatuses(context.Background(), "OPS")
	if err != nil {
		t.Fatal(err)
	}
	want := []provider.Status{{ID: "1", Name: "Por hacer"}, {ID: "2", Name: "Testing"}, {ID: "4", Name: "Aprobado"}, {ID: "3", Name: "Finalizado", Done: true}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("un estado por id, por categoría y los finalizados al final:\n%+v", got)
	}
}

func TestProjectStatusesNamesTheProjectOnError(t *testing.T) {
	c := fakeConfluence(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /rest/api/3/project/XYZ/statuses": func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) },
	})
	if _, err := (&Tracker{c: c}).ProjectStatuses(context.Background(), "XYZ"); err == nil || !strings.Contains(err.Error(), "'XYZ'") {
		t.Errorf("el error dice qué proyecto: %v", err)
	}
}

func TestQAFieldsKeepsNumbersAndDevelopment(t *testing.T) {
	c := fakeConfluence(t, map[string]func(http.ResponseWriter, *http.Request){
		"GET /rest/api/3/field": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, []any{
				map[string]any{"id": "summary", "name": "Resumen", "custom": false, "schema": map[string]any{"type": "string"}},
				map[string]any{"id": "timespent", "name": "Tiempo", "custom": false, "schema": map[string]any{"type": "number"}},
				map[string]any{"id": "customfield_3", "name": "Reprocesos", "custom": true, "schema": map[string]any{"type": "number"}},
				map[string]any{"id": "customfield_1", "name": "Desarrollo", "custom": true, "schema": map[string]any{"type": "any", "custom": devSummaryType}},
				map[string]any{"id": "customfield_2", "name": "errores", "custom": true, "schema": map[string]any{"type": "number"}},
				map[string]any{"id": "customfield_4", "name": "Notas", "custom": true, "schema": map[string]any{"type": "string"}},
			})
		},
	})
	got, err := (&Tracker{c: c}).QAFields(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []provider.Field{
		{ID: "customfield_1", Name: "Desarrollo", Kind: provider.FieldPRStatus},
		{ID: "customfield_2", Name: "errores", Kind: provider.FieldNumber},
		{ID: "customfield_3", Name: "Reprocesos", Kind: provider.FieldNumber},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("solo campos personalizados numéricos y el de desarrollo, por nombre:\n%+v", got)
	}
}
