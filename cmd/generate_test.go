package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/profile"
	"github.com/kaelence-01/draco-hq/internal/provider"
	"github.com/kaelence-01/draco-hq/internal/provider/markdown"
)

type fakeTracker struct {
	linkErr error
	linked  []string
}

func (*fakeTracker) Issue(_ context.Context, key string) (*provider.Issue, error) {
	if key != "OPS-7" {
		return nil, errors.New("recurso no encontrado (404)")
	}
	return &provider.Issue{Key: "OPS-7", Summary: "Filtro de pagos", URL: "https://example.test/browse/OPS-7"}, nil
}
func (*fakeTracker) Me(context.Context) (*provider.User, error) { return &provider.User{ID: "yo"}, nil }
func (t *fakeTracker) LinkDocument(_ context.Context, key string, d provider.DocRef) error {
	t.linked = append(t.linked, key+" → "+d.Title)
	return t.linkErr
}

type fakeChanges struct{}

func (fakeChanges) ChangedFiles(_ context.Context, _ string, hashes []string) ([]string, error) {
	if hashes[0] == "malo" {
		return nil, errors.New("el commit malo no existe en el repo local")
	}
	return []string{"app/Http/Pagos/FiltroController.php", "routes/api.php"}, nil
}
func (fakeChanges) RecentCommits(context.Context, string, int) ([]provider.Commit, error) {
	return nil, nil
}
func (fakeChanges) CurrentBranch(context.Context, string) (string, error) { return "develop", nil }

// fakeEnv makes every command use an empty config, the fake tracker and a
// Markdown folder; it returns the folder.
func fakeEnv(t *testing.T, trk *fakeTracker) string {
	return fakeEnvWith(t, &config.Config{}, trk)
}

// fakeEnvWith is fakeEnv with a given config and tracker.
func fakeEnvWith(t *testing.T, cfg *config.Config, trk provider.Tracker) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "despliegues")
	docs, err := markdown.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	prevLoad, prevOpen, prevDialog := loadConfig, openProfile, saveDialog
	t.Cleanup(func() { loadConfig, openProfile, saveDialog = prevLoad, prevOpen, prevDialog })
	loadConfig = func() (*config.Config, error) { return cfg, nil }
	openProfile = func(*config.Config, profile.Options) (*profile.Bundle, error) {
		return &profile.Bundle{
			Name:    "prueba",
			Profile: config.Profile{Tracker: config.ProviderConfig{Provider: "jira"}, Docs: config.ProviderConfig{Provider: "markdown", Path: dir}},
			Tracker: trk, Docs: docs, Changes: fakeChanges{},
		}, nil
	}
	saveDialog = func(string) string {
		t.Error("el diálogo de guardar no se abre fuera de una terminal")
		return ""
	}
	t.Chdir(t.TempDir()) // no .draco/templates of the developer's repo
	return dir
}

func onlyFile(t *testing.T, dir string) string {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join(dir, "*.md"))
	if len(files) != 1 {
		t.Fatalf("esperaba un documento en %s, hay %v", dir, files)
	}
	b, _ := os.ReadFile(files[0])
	return string(b)
}

func TestGenerateCreatesThenUpdatesKeepingHandEdits(t *testing.T) {
	trk := &fakeTracker{}
	dir := fakeEnv(t, trk)

	// Location [1] (the empty folder), Enter to confirm.
	code, out, errOut := runWith("1\n\n", "g", "-i", "OPS-7", "-b", "a1b2c3d")
	if code != exitOK {
		t.Fatalf("código %d\n%s\n%s", code, out, errOut)
	}
	for _, want := range []string{"[1/4]", "OPS-7 — Filtro de pagos", "Ninguno encontrado", "backend  a1b2c3d → 2 archivos", "Documento creado"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en la salida:\n%s", want, out)
		}
	}
	doc := onlyFile(t, dir)
	if !strings.Contains(doc, "# Documento de Despliegue - OPS-7 - Filtro de pagos") || len(trk.linked) != 1 {
		t.Fatalf("documento o enlace mal creados:\n%s\n%v", doc, trk.linked)
	}

	path, _ := filepath.Glob(filepath.Join(dir, "*.md"))
	edited := strings.Replace(doc, "- [ ] Pasar backend al servidor", "- [x] Pasar backend al servidor", 1)
	if err := os.WriteFile(path[0], []byte(edited), 0644); err != nil {
		t.Fatal(err)
	}

	// [1] Actualizar, Enter to confirm.
	code, out, errOut = runWith("1\n\n", "g", "-i", "OPS-7", "-b", "a1b2c3d")
	if code != exitOK || !strings.Contains(out, "Ya existe un documento") || !strings.Contains(out, "se conserva") || !strings.Contains(out, "Documento actualizado") {
		t.Fatalf("actualización: código %d\n%s\n%s", code, out, errOut)
	}
	if !strings.Contains(onlyFile(t, dir), "- [x] Pasar backend al servidor") {
		t.Error("la actualización perdió lo editado a mano")
	}
}

func TestGenerateUsesTheProjectProfile(t *testing.T) {
	cfg := &config.Config{
		DefaultProfile: "equipo",
		Profiles:       map[string]*config.Profile{"equipo": {}, "servicios": {}},
		Projects:       map[string]*config.ProjectConfig{"api-pagos": {BackendPath: t.TempDir(), Profile: "servicios"}},
	}
	fakeEnvWith(t, cfg, &fakeTracker{})
	open, opened := openProfile, []string{}
	openProfile = func(c *config.Config, o profile.Options) (*profile.Bundle, error) {
		opened = append(opened, o.Name)
		return open(c, o)
	}
	for _, args := range [][]string{{"-p", "api-pagos"}, {"-p", "api-pagos", "--profile", "equipo"}} {
		if code, out, errOut := run(append([]string{"g", "-i", "OPS-7", "-b", "a1b2c3d", "--dry-run"}, args...)...); code != exitOK {
			t.Fatalf("%v: %d\n%s\n%s", args, code, out, errOut)
		}
	}
	if strings.Join(opened, ",") != "servicios,equipo" {
		t.Errorf("el proyecto trae su perfil y --profile gana: %v", opened)
	}
}

func TestGenerateDryRunPublishesNothing(t *testing.T) {
	trk := &fakeTracker{}
	dir := fakeEnv(t, trk)
	code, out, _ := run("g", "-i", "OPS-7", "-b", "a1b2c3d", "--dry-run")
	if code != exitOK || !strings.Contains(out, "## Arquitecturas e interfaces") || !strings.Contains(out, "no se publicó nada") {
		t.Fatalf("código %d\n%s", code, out)
	}
	if files, _ := filepath.Glob(filepath.Join(dir, "*.md")); len(files) != 0 || len(trk.linked) != 0 {
		t.Errorf("--dry-run no debe escribir ni enlazar: %v %v", files, trk.linked)
	}
}

func TestGenerateCancelAndBadAnswers(t *testing.T) {
	dir := fakeEnv(t, &fakeTracker{})
	if code, out, _ := runWith("1\nn\n", "g", "-i", "OPS-7", "-b", "a1b2c3d"); code != exitOK || !strings.Contains(out, "Cancelado") {
		t.Errorf("responder n cancela: %d\n%s", code, out)
	}
	if code, _, errOut := runWith("9\n", "g", "-i", "OPS-7", "-b", "a1b2c3d"); code != exitError || !strings.Contains(errOut, "opción inválida") {
		t.Errorf("una ubicación fuera de rango es un error: %d %s", code, errOut)
	}
	if files, _ := filepath.Glob(filepath.Join(dir, "*.md")); len(files) != 0 {
		t.Errorf("nada debía crearse: %v", files)
	}
}

func TestGenerateReportsFailures(t *testing.T) {
	trk := &fakeTracker{linkErr: errors.New("sin permisos")}
	fakeEnv(t, trk)
	code, out, _ := runWith("1\n\n", "g", "-i", "OPS-7", "-b", "a1b2c3d", "-f", "malo")
	if code != exitOK {
		t.Fatalf("un lado que falla y un enlace que falla no impiden publicar: %d\n%s", code, out)
	}
	for _, want := range []string{"✗ frontend [malo]", "uno de los lados falló", "no se pudo enlazar el documento desde OPS-7"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q en:\n%s", want, out)
		}
	}
	if code, _, errOut := run("g", "-i", "OPS-404", "-b", "a1b2c3d"); code != exitError || !strings.Contains(errOut, "404") {
		t.Errorf("una tarea inexistente es un error: %d %s", code, errOut)
	}
}

func TestExplainProfile(t *testing.T) {
	if err := explainProfile(profile.ErrNoSpace); !errors.Is(err, profile.ErrNoSpace) || !strings.Contains(err.Error(), "--space") {
		t.Errorf("ErrNoSpace debe explicar las opciones: %v", err)
	}
}
