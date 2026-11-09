package markdown

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

func TestDeleteRemovesOnlyTheDocument(t *testing.T) {
	s, dir := newStore(t)
	put(t, dir, "2026/ops-7.md", "# Documento OPS-7\n", 0)
	put(t, filepath.Dir(dir), "fuera.md", "# Fuera\n", 0)

	if err := s.Delete(ctx, provider.DocRef{ID: "2026/ops-7.md"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "2026", "ops-7.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("el archivo sigue ahí: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "2026")); err != nil {
		t.Errorf("la carpeta del documento no se borra: %v", err)
	}

	if err := s.Delete(ctx, provider.DocRef{ID: "2026/ops-7.md"}); !errors.Is(err, provider.ErrNotFound) {
		t.Errorf("un documento que ya no existe es ErrNotFound: %v", err)
	}
	for _, id := range []string{"2026/", "2026", "../fuera.md"} {
		if err := s.Delete(ctx, provider.DocRef{ID: id}); err == nil {
			t.Errorf("%q no se borra: solo documentos dentro de la carpeta", id)
		}
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "fuera.md")); err != nil {
		t.Errorf("un archivo fuera de la carpeta no se toca: %v", err)
	}
}
