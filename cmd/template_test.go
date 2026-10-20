package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/config"
)

// templateEnv runs the template commands over cfg, in an empty folder and a
// temporary home; it returns the user's templates folder.
func templateEnv(t *testing.T, cfg *config.Config) string {
	t.Helper()
	configHome(t)
	prev := loadConfig
	t.Cleanup(func() { loadConfig = prev })
	loadConfig = func() (*config.Config, error) { return cfg, nil }
	t.Chdir(t.TempDir())
	root, err := config.TemplatesDir()
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func writeTemplate(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for n, c := range files {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTemplateInitCopiesAndNeverOverwrites(t *testing.T) {
	user := templateEnv(t, &config.Config{})

	code, out, errOut := run("template", "init", "deploy")
	repoDir := filepath.Join(".draco", "templates", "deploy")
	if code != exitOK || !strings.Contains(out, "copiada en "+repoDir) {
		t.Fatalf("%d\n%s\n%s", code, out, errOut)
	}
	edited := filepath.Join(repoDir, "template.yaml")
	if err := os.WriteFile(edited, []byte("title: \"Documento de Despliegue {{.IssueKey}}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := run("template", "init", "deploy"); code != exitError || !strings.Contains(errOut, "no se reemplaza lo que editaste") {
		t.Errorf("sin --force no sobrescribe: %d %s", code, errOut)
	}
	if code, _, _ := run("template", "init", "deploy", "--force"); code != exitOK {
		t.Error("--force sobrescribe")
	}
	if b, _ := os.ReadFile(edited); !strings.Contains(string(b), "prefix:") {
		t.Errorf("--force restaura la incluida:\n%s", b)
	}

	if code, _, _ := run("template", "init", "qa", "--user"); code != exitOK {
		t.Fatal("--user")
	}
	if _, err := os.Stat(filepath.Join(user, "qa", "main.md.tmpl")); err != nil {
		t.Errorf("--user copia en la carpeta del usuario: %v", err)
	}
	dir := filepath.Join(t.TempDir(), "equipo", "deploy")
	if code, _, _ := run("template", "init", "deploy", "--dir", dir); code != exitOK {
		t.Fatal("--dir")
	}
	if _, err := os.Stat(filepath.Join(dir, "template.yaml")); err != nil {
		t.Errorf("--dir: %v", err)
	}
}

func TestTemplateListShowsWhichOneIsUsed(t *testing.T) {
	root := filepath.Join(t.TempDir(), "plantillas")
	user := templateEnv(t, &config.Config{DefaultProfile: "equipo", Profiles: map[string]*config.Profile{
		"equipo": {Tracker: config.ProviderConfig{Provider: "jira"}, Docs: config.ProviderConfig{Provider: "markdown", Path: "docs"}, Template: root},
	}})
	writeTemplate(t, filepath.Join(root, "qa"), map[string]string{"template.yaml": ""})
	writeTemplate(t, filepath.Join(user, "qa"), map[string]string{"template.yaml": ""})
	writeTemplate(t, filepath.Join(user, "deploy"), map[string]string{"template.yaml": ""})

	code, out, errOut := run("template", "list")
	if code != exitOK {
		t.Fatalf("%d\n%s\n%s", code, out, errOut)
	}
	deploy, qa, _ := strings.Cut(out, "qa — consolidado QA")
	if !strings.Contains(deploy, filepath.Join(user, "deploy")+"  (en uso)") || !strings.Contains(deploy, "(base de todas)") {
		t.Errorf("deploy usa la del usuario:\n%s", deploy)
	}
	if !strings.Contains(qa, "perfil equipo") || !strings.Contains(qa, filepath.Join(root, "qa")+"  (en uso)") || !strings.Contains(qa, "ignorada") {
		t.Errorf("qa usa la del perfil y la del usuario queda ignorada:\n%s", qa)
	}
}

func TestTemplateValidateAndPreview(t *testing.T) {
	user := templateEnv(t, &config.Config{DeployChecklist: []string{"Limpiar la caché de rutas"}})
	if code, out, _ := run("template", "validate"); code != exitOK || !strings.Contains(out, "deploy (incluida): «Documento de Despliegue - OPS-7 - Filtro de pagos»") {
		t.Errorf("las incluidas son válidas: %d\n%s", code, out)
	}

	code, out, _ := run("template", "preview")
	for _, want := range []string{"Plantilla: incluida", "pagos-api", "- [ ] Limpiar la caché de rutas", "datos de ejemplo"} {
		if code != exitOK || !strings.Contains(out, want) {
			t.Errorf("preview: falta %q (%d)\n%s", want, code, out)
		}
	}

	// A checklist block without its heading would lose «A considerar» on update.
	writeTemplate(t, filepath.Join(user, "deploy"), map[string]string{"main.md.tmpl": `{{define "checklist"}}Pasos:
{{range .Checklist}}- [ ] {{.}}
{{end}}{{end}}`})
	writeTemplate(t, filepath.Join(user, "qa"), map[string]string{"template.yaml": "title: \"Otro título\"\n"})
	code, out, errOut := run("template", "validate")
	if code != exitError || !strings.Contains(out, "el documento no tiene «A considerar»") ||
		!strings.Contains(out, "no empieza con el prefijo") || !strings.Contains(errOut, "plantillas con errores: qa") {
		t.Errorf("validate: %d\n%s\n%s", code, out, errOut)
	}
	if code, out, _ := run("template", "preview", "qa"); code != exitError {
		t.Errorf("preview de una plantilla rota es error: %d\n%s", code, out)
	}
}

func TestTemplateUsageErrors(t *testing.T) {
	templateEnv(t, &config.Config{})
	cases := map[string][]string{
		"subcomando desconocido":   {"template", "borrar"},
		"plantilla desconocida":    {"template", "list", "otro"},
		"indica cuál":              {"template", "validate", "--template", "x"},
		"uso: draco template init": {"template", "init"},
		"--user o --dir":           {"template", "init", "qa", "--user", "--dir", "x"},
	}
	for want, args := range cases {
		if code, _, errOut := run(args...); code != exitUsage || !strings.Contains(errOut, want) {
			t.Errorf("%v: %d %q, esperaba %q", args, code, errOut, want)
		}
	}
}

func TestGenerateUsesTheProfileTemplateOfItsKind(t *testing.T) {
	root := filepath.Join(t.TempDir(), "plantillas")
	cfg := &config.Config{DefaultProfile: "equipo", Profiles: map[string]*config.Profile{
		"equipo": {Tracker: config.ProviderConfig{Provider: "jira"}, Docs: config.ProviderConfig{Provider: "markdown", Path: "docs"}, Template: root},
	}}
	fakeEnvWith(t, cfg, &fakeTracker{})
	writeTemplate(t, filepath.Join(root, "deploy"), map[string]string{"template.yaml": "title: \"Documento de Despliegue {{.IssueKey}} (equipo)\"\n"})
	writeTemplate(t, filepath.Join(root, "qa"), map[string]string{"template.yaml": "title: \"Consolidado de Pruebas QA del equipo\"\n"})

	code, out, errOut := run("g", "-i", "OPS-7", "-b", "a1b2c3d", "--dry-run")
	if code != exitOK || !strings.Contains(out, "Título: Documento de Despliegue OPS-7 (equipo)") {
		t.Errorf("draco g toma <template del perfil>/deploy: %d\n%s\n%s", code, out, errOut)
	}
}
