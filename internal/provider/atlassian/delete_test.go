package atlassian

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

func TestDeleteMovesThePageToTheTrash(t *testing.T) {
	var deleted bool
	c := fakeConfluence(t, map[string]func(http.ResponseWriter, *http.Request){
		"DELETE /wiki/api/v2/pages/42": func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("purge") != "" {
				t.Error("borrar no purga: la página queda en la papelera")
			}
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		},
		"DELETE /wiki/api/v2/pages/404": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
		"DELETE /wiki/api/v2/pages/403": func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) },
	})
	docs := NewDocStore(c, "")
	if err := docs.Delete(context.Background(), provider.DocRef{ID: "42"}); err != nil || !deleted {
		t.Fatalf("borrar: %v, borrado %v", err, deleted)
	}
	if err := docs.Delete(context.Background(), provider.DocRef{ID: "404"}); !errors.Is(err, provider.ErrNotFound) {
		t.Errorf("una página que ya no existe es ErrNotFound: %v", err)
	}
	if err := docs.Delete(context.Background(), provider.DocRef{ID: "403"}); err == nil || errors.Is(err, provider.ErrNotFound) {
		t.Errorf("sin permiso es un error, no «ya no existe»: %v", err)
	}
}

func TestUnlinkRemovesOnlyDracosLink(t *testing.T) {
	var got string
	status := http.StatusNoContent
	c := fakeConfluence(t, map[string]func(http.ResponseWriter, *http.Request){
		"DELETE /rest/api/3/issue/OPS-7/remotelink": func(w http.ResponseWriter, r *http.Request) {
			got = r.URL.Query().Get("globalId")
			w.WriteHeader(status)
		},
	})
	trk := NewTracker(c)
	if err := trk.UnlinkDocument(context.Background(), "OPS-7", provider.DocRef{ID: "42"}); err != nil || got != "confluence-page:42" {
		t.Fatalf("quita el enlace por su globalId: %v %q", err, got)
	}

	status = http.StatusNotFound
	if err := trk.UnlinkDocument(context.Background(), "OPS-7", provider.DocRef{ID: "42"}); err != nil {
		t.Errorf("sin enlace no hay nada que quitar: %v", err)
	}

	status = http.StatusForbidden
	if err := trk.UnlinkDocument(context.Background(), "OPS-7", provider.DocRef{ID: "42"}); err == nil {
		t.Error("sin permiso para quitarlo es un error")
	}
}
