// Package markdown is a local folder of Markdown files as a provider.DocStore.
// Each document is one file whose first line is "# <title>"; its ID is the
// path relative to the folder, with "/" separators.
package markdown

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

// Source names this destination in exported text.
const Source = "Markdown local"

// Store is the folder at root.
type Store struct{ root string }

var _ provider.DocStore = (*Store)(nil)

// New returns the store for dir; the folder is created on the first write.
func New(dir string) (*Store, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("ruta del destino inválida %q: %w", dir, err)
	}
	return &Store{root: abs}, nil
}

type doc struct {
	id    string
	title string
	body  string
	mod   time.Time
}

func (s *Store) ref(d doc) provider.DocRef {
	return provider.DocRef{ID: d.id, Title: d.title, URL: filepath.Join(s.root, filepath.FromSlash(d.id))}
}

func (s *Store) refs(ds []doc) []provider.DocRef {
	out := make([]provider.DocRef, len(ds))
	for i, d := range ds {
		out[i] = s.ref(d)
	}
	return out
}

// all returns every document, newest first. A missing folder has none.
func (s *Store) all() ([]doc, error) {
	var out []doc
	err := filepath.WalkDir(s.root, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && p == s.root {
				return fs.SkipAll
			}
			return err
		}
		if e.IsDir() {
			if p != s.root && strings.HasPrefix(e.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if !e.Type().IsRegular() || !strings.EqualFold(filepath.Ext(p), ".md") {
			return nil
		}
		rel, _ := filepath.Rel(s.root, p)
		d, err := s.read(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		out = append(out, d)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error leyendo %s: %w", s.root, err)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].mod.After(out[j].mod) })
	return out, nil
}

func (s *Store) read(id string) (doc, error) {
	p, err := s.path(id)
	if err != nil {
		return doc{}, err
	}
	info, err := os.Stat(p)
	if err != nil {
		return doc{}, err
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return doc{}, err
	}
	content := strings.ReplaceAll(string(raw), "\r\n", "\n")
	first, rest, _ := strings.Cut(content, "\n")
	d := doc{id: id, mod: info.ModTime()}
	if title, ok := strings.CutPrefix(first, "# "); ok {
		d.title, d.body = strings.TrimSpace(title), strings.TrimPrefix(rest, "\n")
	} else {
		d.title, d.body = strings.TrimSuffix(path.Base(id), path.Ext(id)), content
	}
	return d, nil
}

// path resolves an ID inside the folder; nothing outside it is ever touched.
func (s *Store) path(id string) (string, error) {
	p := filepath.Join(s.root, filepath.FromSlash(id))
	rel, err := filepath.Rel(s.root, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("ruta fuera del destino: %q", id)
	}
	return p, nil
}

// write replaces the file through a temporary one, so a failure never leaves
// half a document.
func (s *Store) write(id, title, markdown string) error {
	p, err := s.path(id)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return fmt.Errorf("error creando %s: %w", filepath.Dir(p), err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".draco-*.md")
	if err != nil {
		return fmt.Errorf("error escribiendo %s: %w", p, err)
	}
	defer os.Remove(tmp.Name())
	content := "# " + title + "\n\n" + strings.TrimRight(markdown, "\n") + "\n"
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return fmt.Errorf("error escribiendo %s: %w", p, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("error escribiendo %s: %w", p, err)
	}
	if err := os.Rename(tmp.Name(), p); err != nil {
		return fmt.Errorf("error escribiendo %s: %w", p, err)
	}
	return nil
}

func (s *Store) find(match func(doc) bool) ([]doc, error) {
	all, err := s.all()
	if err != nil {
		return nil, err
	}
	var out []doc
	for _, d := range all {
		if match(d) {
			out = append(out, d)
		}
	}
	return out, nil
}

func (s *Store) first(match func(doc) bool) (*provider.DocRef, error) {
	ds, err := s.find(match)
	if err != nil || len(ds) == 0 {
		return nil, err
	}
	r := s.ref(ds[0])
	return &r, nil
}

func (s *Store) FindByTitle(_ context.Context, title string) (*provider.DocRef, error) {
	return s.first(func(d doc) bool { return d.title == title })
}

func (s *Store) FindByTitleKey(_ context.Context, prefix, key string) (*provider.DocRef, error) {
	re := regexp.MustCompile(`(?i)(^|[^A-Za-z0-9])` + regexp.QuoteMeta(key) + `($|[^A-Za-z0-9])`)
	return s.first(func(d doc) bool { return strings.HasPrefix(d.title, prefix) && re.MatchString(d.title) })
}

func (s *Store) Sections(_ context.Context, r provider.DocRef, names []string) ([]string, error) {
	d, err := s.read(r.ID)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer %s: %w", r.ID, err)
	}
	lines := strings.Split(d.body, "\n")
	var found []string
	for _, name := range names {
		if _, _, ok := section(lines, name); ok {
			found = append(found, name)
		}
	}
	return found, nil
}

// Locations are the folders holding documents whose title starts with the
// prefix, newest first, and the root folder, so an empty destination can take
// its first document. Files carry no author: AuthorID is ignored.
func (s *Store) Locations(_ context.Context, q provider.LocationQuery) (*provider.Locations, error) {
	ds, err := s.find(func(d doc) bool { return strings.HasPrefix(d.title, q.TitlePrefix) })
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []provider.DocRef
	add := func(dir string) {
		if seen[dir] {
			return
		}
		seen[dir] = true
		title := dir + "/"
		if dir == "." {
			title = "./ (raíz de la carpeta)"
		}
		out = append(out, provider.DocRef{ID: dir + "/", Title: title, URL: filepath.Join(s.root, filepath.FromSlash(dir))})
	}
	for _, d := range ds {
		add(path.Dir(d.id))
	}
	add(".")
	return &provider.Locations{Docs: out}, nil
}

// Create writes the document in the folder of near: near itself when it is a
// folder from Locations (its ID ends in "/"), else the folder of that document.
func (s *Store) Create(_ context.Context, near provider.DocRef, d provider.Document) (*provider.DocRef, error) {
	dir := path.Dir(near.ID)
	if strings.HasSuffix(near.ID, "/") {
		dir = path.Clean(near.ID)
	}
	if _, err := s.path(dir); err != nil {
		return nil, err
	}
	id := path.Join(dir, slug(d.Title)+".md")
	for n := 2; ; n++ {
		p, err := s.path(id)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(p); errors.Is(err, fs.ErrNotExist) {
			break
		}
		id = path.Join(dir, fmt.Sprintf("%s-%d.md", slug(d.Title), n))
	}
	if err := s.write(id, d.Title, d.Markdown); err != nil {
		return nil, err
	}
	return &provider.DocRef{ID: id, Title: d.Title, URL: filepath.Join(s.root, filepath.FromSlash(id))}, nil
}

// Update rewrites the document keeping the current content of d.Preserve; it
// writes nothing when the current file cannot be read.
func (s *Store) Update(_ context.Context, r provider.DocRef, d provider.Document) (*provider.DocRef, error) {
	current, err := s.read(r.ID)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el documento actual; no se sobrescribe: %w", err)
	}
	old := strings.Split(current.body, "\n")
	next := strings.Split(d.Markdown, "\n")
	for _, name := range d.Preserve {
		from, to, ok := section(old, name)
		if !ok {
			continue
		}
		start, end, ok := section(next, name)
		if !ok {
			continue
		}
		next = append(append(append([]string{}, next[:start]...), old[from:to]...), next[end:]...)
	}
	if err := s.write(r.ID, d.Title, strings.Join(next, "\n")); err != nil {
		return nil, err
	}
	return &provider.DocRef{ID: r.ID, Title: d.Title, URL: filepath.Join(s.root, filepath.FromSlash(r.ID))}, nil
}

func (s *Store) Search(_ context.Context, text string) ([]provider.DocRef, error) {
	t := strings.ToLower(text)
	ds, err := s.find(func(d doc) bool { return strings.Contains(strings.ToLower(d.title+"\n"+d.body), t) })
	return s.refs(ds), err
}

func (s *Store) SearchTitle(_ context.Context, text string) ([]provider.DocRef, error) {
	t := strings.ToLower(text)
	ds, err := s.find(func(d doc) bool { return strings.Contains(strings.ToLower(d.title), t) })
	return s.refs(ds), err
}

func (s *Store) Export(_ context.Context, r provider.DocRef) (*provider.Exported, error) {
	d, err := s.read(r.ID)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer %s: %w", r.ID, err)
	}
	return &provider.Exported{
		DocRef:      s.ref(d),
		Source:      Source,
		Space:       s.root,
		UpdatedDate: d.mod.Format("2006-01-02"),
		Text:        strings.TrimRight(d.body, "\n") + "\n",
	}, nil
}

var headingRe = regexp.MustCompile(`^(#{1,6})[ \t]+(.*?)[ \t#]*$`)

// section returns the lines under the named heading, up to the next heading of
// the same or higher rank; headings inside code fences do not count.
func section(lines []string, name string) (start, end int, ok bool) {
	want := provider.NormalizeHeading(name)
	start, level, fence := -1, 0, ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fence != "" {
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fence = trimmed[:3]
			continue
		}
		m := headingRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if start < 0 {
			if provider.NormalizeHeading(m[2]) == want {
				start, level = i+1, len(m[1])
			}
			continue
		}
		if len(m[1]) <= level {
			return start, i, true
		}
	}
	if start < 0 {
		return 0, 0, false
	}
	return start, len(lines), true
}

var accents = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n")

// slug turns a title into a file name: lowercase ASCII letters, digits and
// dashes, at most 80 characters.
func slug(title string) string {
	var b strings.Builder
	dash := false
	for _, r := range accents.Replace(strings.ToLower(title)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	s := strings.Trim(b.String(), "-")
	if len(s) > 80 {
		s = strings.TrimRight(s[:80], "-")
	}
	if s == "" {
		return "documento"
	}
	return s
}
