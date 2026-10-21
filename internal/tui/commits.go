package tui

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/deploy"
	"github.com/kaelence-01/draco-hq/internal/profile"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

// recentCommits is how many commits each side lists.
const recentCommits = 30

var (
	markKey = key.NewBinding(key.WithKeys("space"), key.WithHelp("espacio", "marcar"))
	sideKey = key.NewBinding(key.WithKeys("tab", "left", "right"), key.WithHelp("tab", "backend/frontend"))
	hashKey = key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "hash a mano"))
	goKey   = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "continuar"))
)

// side is one repo of the deploy and the commits the user marks in it.
type side struct {
	label   string
	repo    deploy.Repo
	loading bool
	branch  string
	commits []provider.Commit
	err     error
	cursor  int
	picked  map[string]bool
	manual  []string // hashes typed by hand, in order
	given   []string // hashes from the command bar, marked when the side loads
}

// mark marks the given hashes: one in the list (by prefix, either way) is
// picked; any other is added as typed by hand.
func (s *side) mark() {
	for _, h := range s.given {
		i := slices.IndexFunc(s.commits, func(c provider.Commit) bool {
			return strings.HasPrefix(c.ShortHash, h) || strings.HasPrefix(h, c.ShortHash)
		})
		switch {
		case i >= 0:
			s.picked[s.commits[i].ShortHash] = true
		case !slices.Contains(s.manual, h):
			s.manual = append(s.manual, h)
		}
	}
	s.given = nil
}

// hashes are the marked commits in the order of the list (newest first, the
// one the document links), then the ones typed by hand.
func (s *side) hashes() []string {
	var out []string
	for _, c := range s.commits {
		if s.picked[c.ShortHash] {
			out = append(out, c.ShortHash)
		}
	}
	return append(out, s.manual...)
}

// rows are what the list shows: the commits, then the hashes typed by hand.
func (s *side) rows() int { return len(s.commits) + len(s.manual) }

type commitsLoaded struct {
	owner   *commits
	side    int
	branch  string
	commits []provider.Commit
	err     error
}

// commits lets the user mark the commits of each side.
type commits struct {
	st      *state
	bundle  *profile.Bundle
	project string
	issue   string
	sides   []*side
	cur     int
	typing  bool
	hash    string
	note    string
	spinner spinner.Model
	ctx     context.Context
	cancel  context.CancelFunc
	started bool
}

func newCommits(st *state, bundle *profile.Bundle, sides *deploy.Sides, issue string) *commits {
	c := &commits{st: st, bundle: bundle, project: sides.Project, issue: issue, spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot))}
	back, front := sides.Deployable()
	if back {
		c.sides = append(c.sides, &side{label: "Backend", repo: sides.Backend, picked: map[string]bool{}})
	}
	if front {
		c.sides = append(c.sides, &side{label: "Frontend", repo: sides.Frontend, picked: map[string]bool{}})
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	return c
}

// preset takes the hashes of the command bar (-b, -f); each side marks its
// own when it loads.
func (c *commits) preset(back, front []string) {
	for _, g := range []struct {
		label, flag string
		hashes      []string
	}{{"Backend", "-b", back}, {"Frontend", "-f", front}} {
		if len(g.hashes) == 0 {
			continue
		}
		i := slices.IndexFunc(c.sides, func(s *side) bool { return s.label == g.label })
		if i < 0 {
			c.note = "El proyecto no tiene " + strings.ToLower(g.label) + ": se ignoró " + g.flag
			continue
		}
		c.sides[i].given = g.hashes
	}
}

func (c *commits) title() string { return "Commits de " + c.issue }

func (c *commits) keys() []key.Binding {
	keys := []key.Binding{upKey, downKey, markKey}
	if len(c.sides) > 1 {
		keys = append(keys, sideKey)
	}
	return append(keys, hashKey, goKey)
}

func (c *commits) captures(tea.KeyPressMsg) bool { return c.typing }

func (c *commits) close() { c.cancel() }

func (c *commits) init() tea.Cmd {
	if c.started {
		return nil
	}
	c.started = true
	cmds := []tea.Cmd{c.spinner.Tick}
	for i, s := range c.sides {
		s.loading = true
		cmds = append(cmds, c.load(i, s.repo.WorkDir))
	}
	return tea.Batch(cmds...)
}

func (c *commits) load(i int, workDir string) tea.Cmd {
	changes, ctx := c.bundle.Changes, c.ctx
	return func() tea.Msg {
		msg := commitsLoaded{owner: c, side: i}
		if msg.branch, msg.err = changes.CurrentBranch(ctx, workDir); msg.err == nil {
			msg.commits, msg.err = changes.RecentCommits(ctx, workDir, recentCommits)
		}
		return msg
	}
}

func (c *commits) update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case commitsLoaded:
		if msg.owner != c || c.ctx.Err() != nil {
			return c, nil
		}
		s := c.sides[msg.side]
		s.loading, s.branch, s.commits, s.err = false, msg.branch, msg.commits, msg.err
		s.mark()
		return c, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		c.spinner, cmd = c.spinner.Update(msg)
		return c, cmd
	case tea.KeyPressMsg:
		if c.typing {
			c.typeHash(msg)
			return c, nil
		}
		return c, c.press(msg)
	}
	return c, nil
}

func (c *commits) press(k tea.KeyPressMsg) tea.Cmd {
	s := c.sides[c.cur]
	c.note = ""
	switch {
	case key.Matches(k, upKey):
		s.cursor = max(s.cursor-1, 0)
	case key.Matches(k, downKey):
		s.cursor = min(s.cursor+1, max(s.rows()-1, 0))
	case key.Matches(k, sideKey) && len(c.sides) > 1:
		c.cur = 1 - c.cur
	case key.Matches(k, markKey):
		c.toggle(s)
	case key.Matches(k, hashKey):
		c.typing, c.hash = true, ""
	case key.Matches(k, goKey):
		back, front := c.hashes()
		if len(back) == 0 && len(front) == 0 {
			c.note = "Marca al menos un commit con espacio, o agrega uno con h"
			return nil
		}
		return open(newProgress(c.st, c.bundle, c.project, c.issue, back, front))
	}
	return nil
}

func (c *commits) toggle(s *side) {
	if s.cursor < len(s.commits) {
		h := s.commits[s.cursor].ShortHash
		s.picked[h] = !s.picked[h]
		return
	}
	if i := s.cursor - len(s.commits); i < len(s.manual) {
		s.manual = slices.Delete(s.manual, i, i+1)
		s.cursor = min(s.cursor, max(s.rows()-1, 0))
	}
}

func (c *commits) typeHash(k tea.KeyPressMsg) {
	switch {
	case key.Matches(k, backKey):
		c.typing = false
	case k.Code == tea.KeyEnter:
		c.typing = false
		h := strings.TrimSpace(c.hash)
		s := c.sides[c.cur]
		if h == "" || slices.Contains(s.hashes(), h) {
			return
		}
		s.manual = append(s.manual, h)
		s.cursor = s.rows() - 1
	case k.Code == tea.KeyBackspace:
		if r := []rune(c.hash); len(r) > 0 {
			c.hash = string(r[:len(r)-1])
		}
	case k.Text != "" && k.Text != " ":
		c.hash += k.Text
	}
}

// hashes are the commits of each side, ready for the request.
func (c *commits) hashes() (back, front []string) {
	for _, s := range c.sides {
		if s.label == "Backend" {
			back = s.hashes()
		} else {
			front = s.hashes()
		}
	}
	return back, front
}

func (c *commits) view(width, height int) string {
	s := c.sides[c.cur]
	var tabs []string
	for i, sd := range c.sides {
		label := sd.label
		if n := len(sd.hashes()); n > 0 {
			label += " (" + strconv.Itoa(n) + ")"
		}
		if i == c.cur {
			tabs = append(tabs, accentStyle.Bold(true).Render("▸ "+label))
		} else {
			tabs = append(tabs, faintStyle.Render("  "+label))
		}
	}
	lines := []string{strings.Join(tabs, "   "), c.repoLine(s), ""}

	bottom := []string{""}
	switch {
	case c.typing:
		bottom = append(bottom, "Hash ("+strings.ToLower(s.label)+"): "+c.hash+accentStyle.Render("█")+faintStyle.Render("   enter agrega · esc cancela"))
	case c.note != "":
		bottom = append(bottom, warnStyle.Render(c.note))
	default:
		bottom = append(bottom, faintStyle.Render("Un commit de otra rama se agrega con h."))
	}

	switch {
	case s.loading:
		lines = append(lines, c.spinner.View()+" Leyendo los commits…")
	case s.err != nil:
		lines = append(lines, errStyle.Render("✗ "+s.err.Error()), faintStyle.Render("Puedes agregar el hash a mano con h."))
	}
	lines = append(lines, c.list(s, max(height-len(lines)-len(bottom)-2, 3), width)...)
	return lipgloss.NewStyle().Padding(1, 1, 0).Render(lipgloss.JoinVertical(lipgloss.Left, append(lines, bottom...)...))
}

func (c *commits) repoLine(s *side) string {
	where := s.repo.WorkDir
	if where == "" {
		where = "directorio actual"
	}
	line := faintStyle.Render(where)
	if s.repo.Name != "" {
		line = s.repo.Name + "  " + line
	}
	if s.branch != "" {
		line += "  " + faintStyle.Render("rama ") + s.branch
	}
	return line
}

// list shows the window of rows around the cursor that fits in height.
func (c *commits) list(s *side, height, width int) []string {
	if s.rows() == 0 {
		if !s.loading && s.err == nil {
			return []string{faintStyle.Render("Sin commits en esta rama.")}
		}
		return nil
	}
	start := window(s.cursor, s.rows(), height)
	var out []string
	for i := start; i < min(start+height, s.rows()); i++ {
		var mark, text string
		if i < len(s.commits) {
			cm := s.commits[i]
			mark = box(s.picked[cm.ShortHash])
			text = warnStyle.Render(cm.ShortHash) + " " + cm.Subject + faintStyle.Render("  "+cm.Author+" · "+cm.Date.Format("2006-01-02"))
		} else {
			mark = box(true)
			text = warnStyle.Render(s.manual[i-len(s.commits)]) + faintStyle.Render(" (a mano)")
		}
		cursor := "  "
		if i == s.cursor {
			cursor = accentStyle.Render("▸ ")
		}
		out = append(out, lipgloss.NewStyle().MaxWidth(width-4).Render(cursor+mark+" "+text))
	}
	return out
}

func box(on bool) string {
	if on {
		return okStyle.Render("[x]")
	}
	return faintStyle.Render("[ ]")
}
