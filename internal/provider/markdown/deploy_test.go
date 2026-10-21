package markdown_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/deploy"
	"github.com/kaelence-01/draco-hq/internal/provider"
	"github.com/kaelence-01/draco-hq/internal/provider/markdown"
)

type tracker struct{ linked []provider.DocRef }

func (*tracker) Issue(context.Context, string) (*provider.Issue, error) {
	return &provider.Issue{Key: "OPS-7", Summary: "Filtro de pagos", URL: "https://example.test/browse/OPS-7"}, nil
}
func (*tracker) Me(context.Context) (*provider.User, error) { return &provider.User{ID: "yo"}, nil }
func (t *tracker) LinkDocument(_ context.Context, _ string, d provider.DocRef) error {
	t.linked = append(t.linked, d)
	return nil
}

type changes struct{}

func (changes) ChangedFiles(context.Context, string, []string) ([]string, error) {
	return []string{"app/Http/Pagos/FiltroController.php", "routes/api.php"}, nil
}
func (changes) RecentCommits(context.Context, string, int) ([]provider.Commit, error) {
	return nil, nil
}
func (changes) CurrentBranch(context.Context, string) (string, error) { return "develop", nil }

// publish runs the deploy workflow the way the CLI or the TUI will.
func publish(t *testing.T, s *deploy.Session) *deploy.Result {
	t.Helper()
	ctx := context.Background()
	if err := s.FetchIssue(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.FindExisting(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.ReadCommits(ctx); err != nil {
		t.Fatal(err)
	}
	if s.Existing != nil {
		d, err := s.DraftUpdate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if len(d.NotPreserved) != 0 {
			t.Error("el documento publicado tiene «A considerar»")
		}
		res, err := s.Publish(ctx, d, nil)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}
	loc, err := s.Locations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	d, err := s.DraftCreate()
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.Publish(ctx, d, &loc.Docs[0])
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// TestDeployDocumentWithoutConfluence is the closing criterion of activity 6:
// a whole deploy document published to a local folder, and republished keeping
// what the team edited by hand.
func TestDeployDocumentWithoutConfluence(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "despliegues")
	docs, err := markdown.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	target := &deploy.Target{
		Issue: "OPS-7", SpaceKey: dir, VCSHost: "https://bitbucket.org", VCSOrg: "mi-org",
		Backend:   deploy.Repo{Name: "pagos-api", Hashes: []string{"a1b2c3d"}},
		Checklist: []string{"Respaldar base de datos", "Desplegar backend"},
	}
	trk := &tracker{}

	first := publish(t, deploy.NewSession(trk, docs, changes{}, target))
	if first.Updated || first.LinkErr != nil || len(trk.linked) != 1 {
		t.Fatalf("primera publicación = %+v, enlaces %v", first, trk.linked)
	}
	path := filepath.Join(dir, filepath.FromSlash(first.Page.ID))
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Documento de Despliegue - OPS-7 - Filtro de pagos", "FiltroController.php", "commits/a1b2c3d#chg-app/Http/Pagos/FiltroController.php", "- [ ] Respaldar base de datos"} {
		if !strings.Contains(string(content), want) {
			t.Errorf("falta %q en:\n%s", want, content)
		}
	}

	edited := strings.Replace(string(content), "- [ ] Respaldar base de datos", "- [x] Respaldar base de datos\n- [ ] Avisar a soporte", 1)
	if err := os.WriteFile(path, []byte(edited), 0644); err != nil {
		t.Fatal(err)
	}

	second := publish(t, deploy.NewSession(trk, docs, changes{}, target))
	if !second.Updated || second.Page.ID != first.Page.ID {
		t.Fatalf("la segunda publicación debía actualizar el mismo archivo: %+v", second)
	}
	again, _ := os.ReadFile(path)
	if !strings.Contains(string(again), "- [x] Respaldar base de datos\n- [ ] Avisar a soporte") {
		t.Errorf("se perdió lo editado a mano:\n%s", again)
	}
}
