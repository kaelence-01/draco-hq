package document

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

//go:embed templates
var builtin embed.FS

// Manifest is a template's template.yaml.
type Manifest struct {
	Title    string   `yaml:"title"`    // text/template for the document title
	Prefix   string   `yaml:"prefix"`   // fixed start of every title; searches rely on it
	Preserve []string `yaml:"preserve"` // headings of the sections people edit by hand
}

// Template is a document template ready to render.
type Template struct {
	Kind     string
	Source   string // folder that customized the built-in template; empty if none
	Manifest Manifest
	body     *template.Template
	title    *template.Template
}

// funcs are the only functions a template can call: the values it receives
// are already escaped Markdown, so it needs none to stay safe.
var funcs = template.FuncMap{"join": strings.Join}

const mainFile = "main.md.tmpl"

// LoadTemplate loads the built-in template of kind ("deploy", "qa") and
// customizes it with the first of dirs that exists: its template.yaml fields
// replace the built-in ones, and its *.md.tmpl files are parsed over the
// built-in ones, so a {{define}} replaces a {{block}} and main.md.tmpl
// replaces the whole document.
func LoadTemplate(kind string, dirs ...string) (*Template, error) {
	t := &Template{Kind: kind}
	base := path.Join("templates", kind)
	raw, err := builtin.ReadFile(path.Join(base, "template.yaml"))
	if err != nil {
		return nil, fmt.Errorf("no existe la plantilla incluida %q", kind)
	}
	if err := yaml.Unmarshal(raw, &t.Manifest); err != nil {
		return nil, fmt.Errorf("plantilla incluida %q: %w", kind, err)
	}
	if t.body, err = template.New(mainFile).Funcs(funcs).ParseFS(builtin, path.Join(base, "*.md.tmpl")); err != nil {
		return nil, fmt.Errorf("plantilla incluida %q: %w", kind, err)
	}

	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		info, err := os.Stat(dir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil || !info.IsDir() {
			return nil, fmt.Errorf("la plantilla %q no es una carpeta legible", dir)
		}
		if err := t.customize(dir); err != nil {
			return nil, err
		}
		break
	}

	if strings.TrimSpace(t.Manifest.Title) == "" || strings.TrimSpace(t.Manifest.Prefix) == "" {
		return nil, fmt.Errorf("plantilla %q: template.yaml necesita title y prefix", t.name())
	}
	if t.title, err = template.New("title").Funcs(funcs).Parse(t.Manifest.Title); err != nil {
		return nil, fmt.Errorf("plantilla %q, título: %w", t.name(), err)
	}
	return t, nil
}

// MustBuiltin returns a built-in template; for defaults and tests.
func MustBuiltin(kind string) *Template {
	t, err := LoadTemplate(kind)
	if err != nil {
		panic(err)
	}
	return t
}

func (t *Template) customize(dir string) error {
	t.Source = dir
	if raw, err := os.ReadFile(filepath.Join(dir, "template.yaml")); err == nil {
		var m Manifest
		if err := yaml.Unmarshal(raw, &m); err != nil {
			return fmt.Errorf("%s: %w", filepath.Join(dir, "template.yaml"), err)
		}
		if m.Title != "" {
			t.Manifest.Title = m.Title
		}
		if m.Prefix != "" {
			t.Manifest.Prefix = m.Prefix
		}
		if m.Preserve != nil {
			t.Manifest.Preserve = m.Preserve
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("no se pudo leer %s: %w", filepath.Join(dir, "template.yaml"), err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.md.tmpl"))
	if err != nil || len(files) == 0 {
		return err
	}
	if _, err := t.body.ParseFiles(files...); err != nil {
		return fmt.Errorf("plantilla %q: %w", dir, err)
	}
	return nil
}

func (t *Template) name() string {
	if t.Source != "" {
		return t.Source
	}
	return t.Kind
}

func (t *Template) render(data any) (string, error) {
	var b bytes.Buffer
	if err := t.body.ExecuteTemplate(&b, mainFile, data); err != nil {
		return "", fmt.Errorf("plantilla %q: %w", t.name(), err)
	}
	return b.String(), nil
}

// renderTitle executes the title and checks it keeps the prefix: without it,
// searching for an existing document would fail and duplicates would appear.
func (t *Template) renderTitle(data any) (string, error) {
	var b bytes.Buffer
	if err := t.title.Execute(&b, data); err != nil {
		return "", fmt.Errorf("plantilla %q, título: %w", t.name(), err)
	}
	title := strings.TrimSpace(b.String())
	if !strings.HasPrefix(title, t.Manifest.Prefix) {
		return "", fmt.Errorf("plantilla %q: el título %q no empieza con el prefijo %q", t.name(), title, t.Manifest.Prefix)
	}
	return title, nil
}

// Kinds are the built-in templates.
var Kinds = []string{"deploy", "qa"}

// Dirs lists the folders that may customize the template of kind, in the
// order LoadTemplate tries them: explicit, the repo's .draco/templates/<kind>
// (repoDir empty: the current directory), then userRoot/<kind>.
func Dirs(kind, explicit, repoDir, userRoot string) []string {
	d := []string{explicit, filepath.Join(repoDir, ".draco", "templates", kind)}
	if userRoot != "" {
		d = append(d, filepath.Join(userRoot, kind))
	}
	return d
}

// BuiltinFiles returns the files of the built-in template of kind by name, to
// copy them as the start of a custom one.
func BuiltinFiles(kind string) (map[string][]byte, error) {
	entries, err := builtin.ReadDir(path.Join("templates", kind))
	if err != nil {
		return nil, fmt.Errorf("no existe la plantilla incluida %q", kind)
	}
	files := map[string][]byte{}
	for _, e := range entries {
		if files[e.Name()], err = builtin.ReadFile(path.Join("templates", kind, e.Name())); err != nil {
			return nil, err
		}
	}
	return files, nil
}

var headingRe = regexp.MustCompile(`^#{1,6}[ \t]+(.*?)[ \t#]*$`)

// MissingPreserved returns the Manifest.Preserve sections md has no heading
// for: what people write there by hand would be lost on every update.
func (t *Template) MissingPreserved(md string) []string {
	found := map[string]bool{}
	fence := ""
	for _, line := range strings.Split(md, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case fence != "":
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
		case strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~"):
			fence = trimmed[:3]
		default:
			if m := headingRe.FindStringSubmatch(line); m != nil {
				found[provider.NormalizeHeading(m[1])] = true
			}
		}
	}
	var missing []string
	for _, name := range t.Manifest.Preserve {
		if !found[provider.NormalizeHeading(name)] {
			missing = append(missing, name)
		}
	}
	return missing
}
