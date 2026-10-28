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

// The table: columns that do not grow, and how many items the strip lists.
const (
	hashWidth   = 8
	authorWidth = 10
	dateWidth   = 10
	filesWidth  = 5
	stripItems  = 6
	// whoMinWidth is the narrowest table that keeps 24 columns for the message
	// beside the author and the date.
	whoMinWidth = 67
)

var (
	markKey = key.NewBinding(key.WithKeys("space"), key.WithHelp("espacio", "marcar"))
	sideKey = key.NewBinding(key.WithKeys("tab", "left", "right"), key.WithHelp("tab", "repo"))
	hashKey = key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "hash a mano"))
	goKey   = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "continuar"))
)

// fileRead is what reading the files of one commit gave.
type fileRead struct {
	names []string
	err   error
	done  bool
}

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
	manual  []string             // hashes typed by hand, in order
	given   []string             // hashes from the command bar, marked when the side loads
	files   map[string]*fileRead // by hash, read as the cursor reaches each commit
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

// hashAt is the hash of a row of the list; empty past its end.
func (s *side) hashAt(i int) string {
	if i < len(s.commits) {
		return s.commits[i].ShortHash
	}
	if j := i - len(s.commits); j < len(s.manual) {
		return s.manual[j]
	}
	return ""
}

// subject is the message of a marked commit; one typed by hand has none.
func (s *side) subject(hash string) string {
	if i := slices.IndexFunc(s.commits, func(c provider.Commit) bool { return c.ShortHash == hash }); i >= 0 {
		return s.commits[i].Subject
	}
	return "(a mano)"
}

// repoName is how a repo is called in the tabs.
func (s *side) repoName() string {
	switch {
	case s.repo.Name != "":
		return s.repo.Name
	case s.repo.WorkDir != "":
		return s.repo.WorkDir
	}
	return "directorio actual"
}

type commitsLoaded struct {
	owner   *commits
	side    int
	branch  string
	commits []provider.Commit
	err     error
}

type filesLoaded struct {
	owner *commits
	side  int
	hash  string
	names []string
	err   error
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
		c.sides = append(c.sides, newSide("Backend", sides.Backend))
	}
	if front {
		c.sides = append(c.sides, newSide("Frontend", sides.Frontend))
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	return c
}

func newSide(label string, r deploy.Repo) *side {
	return &side{label: label, repo: r, picked: map[string]bool{}, files: map[string]*fileRead{}}
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
	keys := []key.Binding{moveKey, markKey}
	if len(c.sides) > 1 {
		keys = append(keys, sideKey)
	}
	return append(keys, goKey)
}

func (c *commits) moreKeys() []key.Binding { return []key.Binding{hashKey, upKey, downKey} }

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

// need reads the files of one commit, once: the detail and the totals show
// them without waiting for the document.
func (c *commits) need(i int, hash string) tea.Cmd {
	s := c.sides[i]
	if hash == "" || s.files[hash] != nil {
		return nil
	}
	s.files[hash] = &fileRead{}
	changes, ctx, dir := c.bundle.Changes, c.ctx, s.repo.WorkDir
	return func() tea.Msg {
		names, err := changes.ChangedFiles(ctx, dir, []string{hash})
		return filesLoaded{owner: c, side: i, hash: hash, names: names, err: err}
	}
}

// needCursor reads the files of the commit under the cursor.
func (c *commits) needCursor() tea.Cmd {
	s := c.sides[c.cur]
	return c.need(c.cur, s.hashAt(s.cursor))
}

// needMarked reads the files of every marked commit of a side.
func (c *commits) needMarked(i int) tea.Cmd {
	var cmds []tea.Cmd
	for _, h := range c.sides[i].hashes() {
		cmds = append(cmds, c.need(i, h))
	}
	return tea.Batch(cmds...)
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
		return c, tea.Batch(c.need(msg.side, s.hashAt(s.cursor)), c.needMarked(msg.side))
	case filesLoaded:
		if msg.owner != c || c.ctx.Err() != nil {
			return c, nil
		}
		c.sides[msg.side].files[msg.hash] = &fileRead{names: msg.names, err: msg.err, done: true}
		return c, nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		c.spinner, cmd = c.spinner.Update(msg)
		return c, cmd
	case tea.KeyPressMsg:
		if c.typing {
			return c, c.typeHash(msg)
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
		return c.needCursor()
	case key.Matches(k, downKey):
		s.cursor = min(s.cursor+1, max(s.rows()-1, 0))
		return c.needCursor()
	case key.Matches(k, sideKey) && len(c.sides) > 1:
		c.cur = 1 - c.cur
		return c.needCursor()
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

func (c *commits) typeHash(k tea.KeyPressMsg) tea.Cmd {
	switch {
	case key.Matches(k, backKey):
		c.typing = false
	case k.Code == tea.KeyEnter:
		c.typing = false
		h := strings.TrimSpace(c.hash)
		s := c.sides[c.cur]
		if h == "" || slices.Contains(s.hashes(), h) {
			return nil
		}
		s.manual = append(s.manual, h)
		s.cursor = s.rows() - 1
		return c.need(c.cur, h)
	case k.Code == tea.KeyBackspace:
		if r := []rune(c.hash); len(r) > 0 {
			c.hash = string(r[:len(r)-1])
		}
	case k.Text != "" && k.Text != " ":
		c.hash += k.Text
	}
	return nil
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

// totals count the marked commits of both sides and their files, each file
// once per side as the document lists it.
func (c *commits) totals() (commits, files int) {
	for _, s := range c.sides {
		seen := map[string]bool{}
		for _, h := range s.hashes() {
			commits++
			if fr := s.files[h]; fr != nil {
				for _, f := range fr.names {
					seen[f] = true
				}
			}
		}
		files += len(seen)
	}
	return commits, files
}

func (c *commits) view(width, height int) string {
	if width < panelMinWidth {
		return c.narrow(width, height)
	}
	return c.wide(width, height)
}

// wide is the list of commits and, beside it, the files of the one under the
// cursor; under both, what is marked.
func (c *commits) wide(width, height int) string {
	inner := width - 2
	top := titleLine(boldStyle.Render(c.issue), deploySteps, 1, inner)
	strip := c.strip(inner)
	colsHeight := max(height-2-lipgloss.Height(strip), 6)
	listWidth, detailWidth := columns(inner, 2, 6, 4)
	cols := lipgloss.JoinHorizontal(lipgloss.Top, c.listBox(listWidth, colsHeight), "  ", c.detailBox(detailWidth, colsHeight))
	return lipgloss.NewStyle().PaddingLeft(1).Render(lipgloss.JoinVertical(lipgloss.Left, top, "", cols, strip))
}

func (c *commits) listBox(width, height int) string {
	s := c.sides[c.cur]
	inner := width - 4
	lines := []string{c.tabs(inner), faintStyle.Render(c.header(inner))}
	lines = append(lines, c.state(s, inner)...)
	hint := lipgloss.NewStyle().Width(inner).Render(c.hint(s))
	rows := max(height-2-len(lines)-lipgloss.Height(hint), 1)
	lines = append(lines, c.table(s, rows, inner)...)
	for len(lines) < height-2-lipgloss.Height(hint) {
		lines = append(lines, "")
	}
	return stretch(strings.Join(append(lines, hint), "\n"), width, height)
}

// state is what stands in for the list while it loads, fails or is empty.
func (c *commits) state(s *side, width int) []string {
	switch {
	case s.loading:
		return []string{c.spinner.View() + " Leyendo los commits…"}
	case s.err != nil:
		return strings.Split(lipgloss.NewStyle().Width(width).Render(errStyle.Render("✗ "+s.err.Error())+"\n"+faintStyle.Render("Puedes agregar el hash a mano con h.")), "\n")
	case s.rows() == 0:
		return []string{faintStyle.Render("Sin commits en esta rama.")}
	}
	return nil
}

// hint is the line under the list: the hash being typed, a note, or what h does.
func (c *commits) hint(s *side) string {
	switch {
	case c.typing:
		return "Hash (" + strings.ToLower(s.label) + "): " + c.hash + accentStyle.Render("█") + faintStyle.Render("   enter agrega · esc cancela")
	case c.note != "":
		return warnStyle.Render(c.note)
	}
	return faintStyle.Render("Un commit de otra rama se agrega con h.")
}

// tabs are the repos, the current one with its branch; the other gives up its
// repo name when they do not fit.
func (c *commits) tabs(width int) string {
	build := func(others bool) string {
		var tabs []string
		for i, sd := range c.sides {
			label := sd.label
			if n := len(sd.hashes()); n > 0 {
				label += " (" + strconv.Itoa(n) + ")"
			}
			switch {
			case i == c.cur:
				tab := accentStyle.Bold(true).Render("▸ "+label) + "  " + sd.repoName()
				if sd.branch != "" {
					tab += faintStyle.Render(" · " + sd.branch)
				}
				tabs = append(tabs, tab)
			case others:
				tabs = append(tabs, faintStyle.Render("  "+label+"  "+sd.repoName()))
			default:
				tabs = append(tabs, faintStyle.Render("  "+label))
			}
		}
		return strings.Join(tabs, "    ")
	}
	if t := build(true); lipgloss.Width(t) <= width {
		return t
	}
	return build(false)
}

// withWho reports whether the table has room for the author and the date; the
// detail shows them when it does not.
func withWho(width int) bool { return width >= whoMinWidth }

// header names the columns of the table.
func (c *commits) header(width int) string {
	out := strings.Repeat(" ", 6) + padRight("HASH", hashWidth+1) + padRight("MENSAJE", messageWidth(width)+1)
	if withWho(width) {
		out += padRight("AUTOR", authorWidth+1) + padRight("FECHA", dateWidth+1)
	}
	return out + padLeft("ARCH.", filesWidth)
}

// messageWidth is what the table leaves to the message.
func messageWidth(width int) int {
	fixed := 6 + hashWidth + 1 + 1 + filesWidth
	if withWho(width) {
		fixed += authorWidth + 1 + dateWidth + 1
	}
	return max(width-fixed, 8)
}

// table shows the window of rows around the cursor that fits in rows lines.
func (c *commits) table(s *side, rows, width int) []string {
	msgWidth := messageWidth(width)
	var out []string
	start := window(s.cursor, s.rows(), rows)
	for i := start; i < min(start+rows, s.rows()); i++ {
		hash := s.hashAt(i)
		cursor := "  "
		if i == s.cursor {
			cursor = accentStyle.Render("▸ ")
		}
		message, author, date := faintStyle.Render("(a mano)"), "", ""
		if i < len(s.commits) {
			cm := s.commits[i]
			message, author, date = cm.Subject, cm.Author, faintStyle.Render(cm.Date.Format("2006-01-02"))
			if i == s.cursor {
				message = accentStyle.Bold(true).Render(message)
			}
		}
		row := cursor + box(s.picked[hash] || i >= len(s.commits)) + " " + cell(warnStyle.Render(hash), hashWidth) + " " + cell(message, msgWidth) + " "
		if withWho(width) {
			row += cell(author, authorWidth) + " " + cell(date, dateWidth) + " "
		}
		out = append(out, row+padLeft(fileCount(s, hash), filesWidth))
	}
	return out
}

// fileCount is how many files a commit changed, once they are read.
func fileCount(s *side, hash string) string {
	if fr := s.files[hash]; fr != nil && fr.done && fr.err == nil {
		return strconv.Itoa(len(fr.names))
	}
	return ""
}

// cell cuts s to width and pads it to it.
func cell(s string, width int) string {
	return padRight(lipgloss.NewStyle().MaxWidth(width).Render(s), width)
}

func padLeft(s string, n int) string { return strings.Repeat(" ", max(n-lipgloss.Width(s), 0)) + s }

func (c *commits) detailBox(width, height int) string {
	return panel(strings.Join(c.detail(c.sides[c.cur]), "\n"), width, height)
}

// detail is the commit under the cursor and the files it changed.
func (c *commits) detail(s *side) []string {
	hash := s.hashAt(s.cursor)
	if hash == "" {
		return nil
	}
	var lines []string
	if s.cursor < len(s.commits) {
		cm := s.commits[s.cursor]
		meta := slices.DeleteFunc([]string{cm.Author, cm.Date.Format("2006-01-02"), s.branch}, func(p string) bool { return p == "" })
		lines = append(lines, accentStyle.Render(cm.ShortHash)+" "+boldStyle.Render(cm.Subject), faintStyle.Render(strings.Join(meta, " · ")))
	} else {
		lines = append(lines, warnStyle.Render(hash)+faintStyle.Render(" (a mano)"))
	}
	fr := s.files[hash]
	switch {
	case fr == nil || !fr.done:
		return append(lines, "", faintStyle.Render("ARCHIVOS"), c.spinner.View()+" Leyendo los archivos…")
	case fr.err != nil:
		return append(lines, "", faintStyle.Render("ARCHIVOS"), errStyle.Render("✗ "+fr.err.Error()))
	case len(fr.names) == 0:
		return append(lines, "", faintStyle.Render("ARCHIVOS (0)"), faintStyle.Render("Sin archivos propios: ¿un commit de merge?"))
	}
	lines = append(lines, "", faintStyle.Render("ARCHIVOS ("+strconv.Itoa(len(fr.names))+")"))
	return append(lines, fr.names...)
}

// strip is what is marked in both repos and how many files that touches.
func (c *commits) strip(width int) string {
	commitsN, filesN := c.totals()
	head := faintStyle.Render("SELECCIONADOS") + "  " + boldStyle.Render(plural(commitsN, "commit", "commits")+" · "+plural(filesN, "archivo", "archivos"))
	if commitsN == 0 {
		return boxStyle.Width(width).Render(head + "   " + faintStyle.Render("ninguno: marca con espacio"))
	}
	var items, none []string
	for _, s := range c.sides {
		hashes := s.hashes()
		if len(hashes) == 0 {
			none = append(none, faintStyle.Render(strings.ToLower(s.label)+": ninguno"))
		}
		for _, h := range hashes {
			items = append(items, okStyle.Render("✓")+" "+warnStyle.Render(h)+" "+faintStyle.Render(lipgloss.NewStyle().MaxWidth(26).Render(s.subject(h))))
		}
	}
	if len(items) > stripItems {
		items = append(items[:stripItems], faintStyle.Render("+"+strconv.Itoa(len(items)-stripItems)+" más"))
	}
	return boxStyle.Width(width).Render(head + "   " + strings.Join(append(items, none...), "   "))
}

// narrow is the list alone, the files of the commit under the cursor and a
// count of what is marked, one under the other.
func (c *commits) narrow(width, height int) string {
	s := c.sides[c.cur]
	inner := width - 2
	lines := []string{boldStyle.Render(c.issue), c.tabs(inner), ""}
	lines = append(lines, c.state(s, inner)...)

	commitsN, filesN := c.totals()
	files := lipgloss.NewStyle().Width(inner).Render(faintStyle.Render("ARCHIVOS ") + c.cursorFiles(s))
	if fl := strings.Split(files, "\n"); len(fl) > 2 {
		files = strings.Join(append(fl[:1], faintStyle.Render("…")), "\n")
	}
	bottom := []string{"", files, faintStyle.Render("MARCADOS ") + plural(commitsN, "commit", "commits") + " · " + plural(filesN, "archivo", "archivos"),
		lipgloss.NewStyle().Width(inner).Render(c.hint(s))}

	lines = append(lines, faintStyle.Render(c.header(inner)))
	rows := max(height-len(lines)-lipgloss.Height(strings.Join(bottom, "\n")), 3)
	for _, r := range c.table(s, rows, inner) {
		lines = append(lines, lipgloss.NewStyle().MaxWidth(inner).Render(r))
	}
	return lipgloss.NewStyle().Padding(0, 1).Render(strings.Join(append(lines, bottom...), "\n"))
}

// cursorFiles is the files of the commit under the cursor, in one line.
func (c *commits) cursorFiles(s *side) string {
	fr := s.files[s.hashAt(s.cursor)]
	switch {
	case s.hashAt(s.cursor) == "":
		return faintStyle.Render("—")
	case fr == nil || !fr.done:
		return faintStyle.Render("leyendo…")
	case fr.err != nil:
		return errStyle.Render(fr.err.Error())
	case len(fr.names) == 0:
		return faintStyle.Render("sin archivos propios")
	}
	return strings.Join(fr.names, ", ")
}

func box(on bool) string {
	if on {
		return okStyle.Render("[x]")
	}
	return faintStyle.Render("[ ]")
}
