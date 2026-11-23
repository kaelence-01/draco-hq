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

// How many commits each read lists: a page of the list, and those that
// mention the issue in any branch.
const (
	commitsPage  = 100
	suggestedMax = 50
)

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
	findKey = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "buscar"))
	mineKey = key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "solo los míos"))
	allKey  = key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "todas las ramas"))
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
	label       string
	repo        deploy.Repo
	loading     bool
	paging      bool
	full        bool // the last page was full: there may be older commits
	branch      string
	email       string
	identityErr error
	commits     []provider.Commit // the pages read, of the branch or of every branch
	suggested   []provider.Commit // those that mention the issue, in any branch
	err         error
	cursor      int
	picked      map[string]bool
	known       map[string]provider.Commit // every commit read, by short hash
	order       []string                   // the short hashes of known, as they arrived
	manual      []string                   // hashes typed by hand, in order
	given       []string                   // hashes from the command bar, marked when the side loads
	files       map[string]*fileRead       // by hash, read as the cursor reaches each commit
}

// row is one line of the list: a commit, or a hash typed by hand.
type row struct {
	commit    *provider.Commit
	hash      string
	suggested bool
}

func (s *side) remember(list []provider.Commit) {
	for _, c := range list {
		if _, ok := s.known[c.ShortHash]; !ok {
			s.known[c.ShortHash] = c
			s.order = append(s.order, c.ShortHash)
		}
	}
}

// find is the commit read whose hash starts with h, or that h starts with.
func (s *side) find(h string) (string, bool) {
	i := slices.IndexFunc(s.order, func(c string) bool {
		return strings.HasPrefix(c, h) || strings.HasPrefix(h, c) || strings.HasPrefix(s.known[c].Hash, h)
	})
	if i < 0 {
		return "", false
	}
	return s.order[i], true
}

// mark marks the given hashes: one already read is picked; any other is added
// as typed by hand.
func (s *side) mark() {
	for _, h := range s.given {
		if c, ok := s.find(h); ok {
			s.picked[c] = true
		} else if !slices.Contains(s.manual, h) {
			s.manual = append(s.manual, h)
		}
	}
	s.given = nil
}

// hashes are the marked commits, newest first (the one the document links),
// then the ones typed by hand.
func (s *side) hashes() []string {
	var marked []provider.Commit
	for _, h := range s.order {
		if s.picked[h] {
			marked = append(marked, s.known[h])
		}
	}
	slices.SortStableFunc(marked, func(a, b provider.Commit) int { return b.Date.Compare(a.Date) })
	var out []string
	for _, c := range marked {
		out = append(out, c.ShortHash)
	}
	return append(out, s.manual...)
}

// subject is the message of a marked commit; one typed by hand has none.
func (s *side) subject(hash string) string {
	if c, ok := s.known[hash]; ok {
		return c.Subject
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
	owner       *commits
	side        int
	branch      string
	email       string
	identityErr error
	commits     []provider.Commit
	suggested   []provider.Commit
	err         error
}

type pageLoaded struct {
	owner   *commits
	side    int
	all     bool
	skip    int
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
	filter  filter
	mine    bool
	all     bool
	fromBar bool
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
	return &side{label: label, repo: r, picked: map[string]bool{}, known: map[string]provider.Commit{}, files: map[string]*fileRead{}}
}

// preset takes the hashes of the command bar (-b, -f); each side marks its
// own when it loads, and nothing is marked for the user.
func (c *commits) preset(back, front []string) {
	for _, g := range []struct {
		label, flag string
		hashes      []string
	}{{"Backend", "-b", back}, {"Frontend", "-f", front}} {
		if len(g.hashes) == 0 {
			continue
		}
		c.fromBar = true
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

func (c *commits) moreKeys() []key.Binding {
	return []key.Binding{findKey, mineKey, allKey, upKey, downKey}
}

func (c *commits) captures(k tea.KeyPressMsg) bool { return c.filter.captures(k) }

// paste goes to the search; from a line copied from git log --oneline, only
// its hash.
func (c *commits) paste(s string) tea.Cmd {
	if !c.filter.typing {
		return nil
	}
	if f := strings.Fields(s); len(f) > 0 && c.filter.text == "" && deploy.LooksLikeHash(f[0]) {
		c.filter.text = f[0]
	} else {
		c.filter.paste(s)
	}
	c.top()
	return c.needCursor()
}

func (c *commits) close() { c.cancel() }

func (c *commits) init() tea.Cmd {
	if c.started {
		return nil
	}
	c.started = true
	cmds := []tea.Cmd{c.spinner.Tick}
	for i, s := range c.sides {
		s.loading = true
		cmds = append(cmds, c.load(i))
	}
	return tea.Batch(cmds...)
}

// load reads a side for the first time: its branch, the identity of the
// clone, the first page and the commits that mention the issue.
func (c *commits) load(i int) tea.Cmd {
	changes, ctx, dir, issue := c.bundle.Changes, c.ctx, c.sides[i].repo.WorkDir, c.issue
	return func() tea.Msg {
		msg := commitsLoaded{owner: c, side: i}
		if msg.branch, msg.err = changes.CurrentBranch(ctx, dir); msg.err != nil {
			return msg
		}
		msg.email, msg.identityErr = changes.Identity(ctx, dir)
		if msg.commits, msg.err = changes.Commits(ctx, dir, provider.CommitQuery{Limit: commitsPage}); msg.err != nil {
			return msg
		}
		found, err := changes.Commits(ctx, dir, provider.CommitQuery{Limit: suggestedMax, AllBranches: true, Grep: issue})
		msg.suggested = slices.DeleteFunc(found, func(cm provider.Commit) bool { return !deploy.MentionsIssue(cm.Subject, issue) })
		msg.err = err
		return msg
	}
}

// loadPage reads the next page of a side: of its branch, or of every branch.
func (c *commits) loadPage(i int) tea.Cmd {
	s := c.sides[i]
	s.paging = true
	changes, ctx, dir := c.bundle.Changes, c.ctx, s.repo.WorkDir
	q := provider.CommitQuery{Limit: commitsPage, Skip: len(s.commits), AllBranches: c.all}
	return func() tea.Msg {
		list, err := changes.Commits(ctx, dir, q)
		return pageLoaded{owner: c, side: i, all: q.AllBranches, skip: q.Skip, commits: list, err: err}
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
	return c.need(c.cur, c.cursorHash(c.sides[c.cur]))
}

// needMarked reads the files of every marked commit of a side.
func (c *commits) needMarked(i int) tea.Cmd {
	var cmds []tea.Cmd
	for _, h := range c.sides[i].hashes() {
		cmds = append(cmds, c.need(i, h))
	}
	return tea.Batch(cmds...)
}

// rows are what the list of a side shows: the commits that mention the issue,
// then the others read, both as the search and m leave them; and the hashes
// typed by hand.
func (c *commits) rows(s *side) []row {
	var out []row
	seen := map[string]bool{}
	add := func(list []provider.Commit, suggested bool) {
		for i := range list {
			cm := &list[i]
			if seen[cm.ShortHash] {
				continue
			}
			seen[cm.ShortHash] = true
			if (!c.mine || deploy.Mine(*cm, s.email)) && deploy.CommitMatches(*cm, c.filter.text) {
				out = append(out, row{commit: cm, hash: cm.ShortHash, suggested: suggested})
			}
		}
	}
	add(s.suggested, true)
	add(s.commits, false)
	for _, h := range s.manual {
		out = append(out, row{hash: h})
	}
	return out
}

// narrowed reports whether the search or m leave out commits.
func (c *commits) narrowed() bool { return c.mine || strings.TrimSpace(c.filter.text) != "" }

// cursorHash is the hash under the cursor; empty when the list is.
func (c *commits) cursorHash(s *side) string {
	if rows := c.rows(s); s.cursor < len(rows) {
		return rows[s.cursor].hash
	}
	return ""
}

// top puts the cursor of every side on its first row, after the list changed.
func (c *commits) top() {
	for _, s := range c.sides {
		s.cursor = 0
	}
}

func (c *commits) update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case commitsLoaded:
		if msg.owner != c || c.ctx.Err() != nil {
			return c, nil
		}
		return c, c.loaded(msg)
	case pageLoaded:
		if msg.owner != c || c.ctx.Err() != nil {
			return c, nil
		}
		s := c.sides[msg.side]
		if msg.all != c.all || msg.skip != len(s.commits) {
			return c, nil
		}
		s.paging = false
		if msg.err != nil {
			c.note = "No se pudieron leer más commits: " + msg.err.Error()
			return c, nil
		}
		s.commits = append(s.commits, msg.commits...)
		s.full = len(msg.commits) == commitsPage
		s.remember(msg.commits)
		return c, c.needCursor()
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
		return c, c.press(msg)
	}
	return c, nil
}

// loaded keeps what the first read of a side gave and marks, unless the
// command bar brought hashes, the user's commits that mention the issue.
func (c *commits) loaded(msg commitsLoaded) tea.Cmd {
	s := c.sides[msg.side]
	s.loading, s.branch, s.err = false, msg.branch, msg.err
	s.email, s.identityErr = msg.email, msg.identityErr
	if msg.err != nil {
		s.mark()
		return nil
	}
	s.suggested, s.commits, s.full = msg.suggested, msg.commits, len(msg.commits) == commitsPage
	s.remember(s.suggested)
	s.remember(s.commits)
	s.mark()
	if !c.fromBar {
		for _, cm := range s.suggested {
			if deploy.Mine(cm, s.email) {
				s.picked[cm.ShortHash] = true
			}
		}
	}
	return tea.Batch(c.need(msg.side, c.cursorHash(s)), c.needMarked(msg.side))
}

func (c *commits) press(k tea.KeyPressMsg) tea.Cmd {
	s := c.sides[c.cur]
	c.note = ""
	if c.filter.typing && k.Code == tea.KeyEnter {
		return c.enterSearch(s)
	}
	before := c.filter.text
	if c.filter.handle(k) {
		if c.filter.text != before {
			c.top()
		}
		return c.needCursor()
	}
	rows := c.rows(s)
	switch {
	case key.Matches(k, upKey):
		s.cursor = max(s.cursor-1, 0)
		return c.needCursor()
	case key.Matches(k, downKey):
		if s.cursor >= len(rows)-1 && s.full && !s.paging && !s.loading {
			return c.loadPage(c.cur)
		}
		s.cursor = min(s.cursor+1, max(len(rows)-1, 0))
		return c.needCursor()
	case key.Matches(k, sideKey) && len(c.sides) > 1:
		c.cur = 1 - c.cur
		return c.needCursor()
	case key.Matches(k, markKey):
		c.toggle(s, rows)
	case key.Matches(k, mineKey):
		return c.toggleMine(s)
	case key.Matches(k, allKey):
		return c.toggleAll()
	case key.Matches(k, goKey):
		back, front := c.hashes()
		if len(back) == 0 && len(front) == 0 {
			c.note = "Marca al menos un commit con espacio, o busca con /"
			return nil
		}
		return open(newProgress(c.st, c.bundle, c.project, c.issue, back, front))
	}
	return nil
}

// enterSearch ends the search; when what was typed is a hash, it marks that
// commit, or adds it as typed by hand when it was not read.
func (c *commits) enterSearch(s *side) tea.Cmd {
	c.filter.typing = false
	h := strings.TrimSpace(c.filter.text)
	if !deploy.LooksLikeHash(h) {
		return c.needCursor()
	}
	c.filter.text = ""
	c.top()
	if found, ok := s.find(h); ok {
		s.picked[found] = true
		c.note = "✓ " + found + " marcado"
		s.cursor = max(slices.IndexFunc(c.rows(s), func(r row) bool { return r.hash == found }), 0)
		return tea.Batch(c.needCursor(), c.need(c.cur, found))
	}
	if !slices.Contains(s.manual, h) {
		s.manual = append(s.manual, h)
	}
	s.cursor = len(c.rows(s)) - 1
	return c.need(c.cur, h)
}

func (c *commits) toggle(s *side, rows []row) {
	if s.cursor >= len(rows) {
		return
	}
	if r := rows[s.cursor]; r.commit != nil {
		s.picked[r.hash] = !s.picked[r.hash]
		return
	}
	s.manual = slices.DeleteFunc(s.manual, func(h string) bool { return h == rows[s.cursor].hash })
	s.cursor = min(s.cursor, max(len(rows)-2, 0))
}

// toggleMine shows only the commits of the user, when the clone says who that is.
func (c *commits) toggleMine(s *side) tea.Cmd {
	if !c.mine && s.email == "" && !s.loading {
		if s.identityErr != nil {
			c.note = "No se pudo leer tu user.email: " + s.identityErr.Error()
		} else {
			c.note = "Sin user.email en el repo: git config user.email"
		}
		return nil
	}
	c.mine = !c.mine
	c.top()
	return c.needCursor()
}

// toggleAll switches every side between its branch and every branch, from
// the first page again; what is marked stays.
func (c *commits) toggleAll() tea.Cmd {
	if slices.ContainsFunc(c.sides, func(s *side) bool { return s.loading }) {
		c.note = "Espera a que terminen de leerse los commits"
		return nil
	}
	c.all = !c.all
	var cmds []tea.Cmd
	for i, s := range c.sides {
		if s.err != nil {
			continue
		}
		s.commits, s.full, s.cursor = nil, false, 0
		cmds = append(cmds, c.loadPage(i))
	}
	return tea.Batch(cmds...)
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
	rows := c.rows(s)
	lines := []string{c.tabs(inner), faintStyle.Render(c.header(inner))}
	lines = append(lines, c.state(s, rows, inner)...)
	hint := lipgloss.NewStyle().Width(inner).Render(c.hint(s, rows))
	n := max(height-2-len(lines)-lipgloss.Height(hint), 1)
	lines = append(lines, c.table(s, rows, n, inner)...)
	for len(lines) < height-2-lipgloss.Height(hint) {
		lines = append(lines, "")
	}
	return stretch(strings.Join(append(lines, hint), "\n"), width, height)
}

// state is what stands in for the list while it loads, fails or is empty.
func (c *commits) state(s *side, rows []row, width int) []string {
	wrap := func(text string) []string { return strings.Split(lipgloss.NewStyle().Width(width).Render(text), "\n") }
	switch {
	case s.loading, s.paging && len(rows) == 0:
		return []string{c.spinner.View() + " Leyendo los commits…"}
	case s.err != nil:
		return wrap(errStyle.Render("✗ "+s.err.Error()) + "\n" + faintStyle.Render("Puedes pegar el hash con / y agregarlo con enter."))
	case len(rows) == 0 && c.narrowed():
		more := ""
		if s.full {
			more = " ↓ lee " + strconv.Itoa(commitsPage) + " más."
		}
		return wrap(faintStyle.Render("Ningún commit coincide." + more))
	case len(rows) == 0:
		return []string{faintStyle.Render("Sin commits en esta rama.")}
	}
	return nil
}

// hint is the line under the list: the search being typed, a note, or the
// state of the search, m and a.
func (c *commits) hint(s *side, rows []row) string {
	switch {
	case c.filter.typing:
		return "/ " + c.filter.text + accentStyle.Render("█") + faintStyle.Render("   enter: un hash se marca o se agrega · esc limpia")
	case c.note != "":
		if strings.HasPrefix(c.note, "✓") {
			return okStyle.Render(c.note)
		}
		return warnStyle.Render(c.note)
	}
	find := accentStyle.Render("/ ") + faintStyle.Render("buscar")
	if c.filter.text != "" {
		find = "filtro: " + c.filter.text + faintStyle.Render(" (esc lo quita)")
	}
	mine := faintStyle.Render("m míos")
	if c.mine {
		mine = accentStyle.Render("m: solo míos")
	}
	scope := faintStyle.Render("a todas las ramas")
	if c.all {
		scope = accentStyle.Render("a: todas las ramas")
	}
	return strings.Join([]string{find, mine, scope, faintStyle.Render(c.count(s, rows))}, faintStyle.Render(" · "))
}

// count is how many commits the list shows of those it has read.
func (c *commits) count(s *side, rows []row) string {
	seen := map[string]bool{}
	for _, cm := range append(slices.Clip(s.suggested), s.commits...) {
		seen[cm.ShortHash] = true
	}
	read := len(seen)
	out := strconv.Itoa(read) + " leídos"
	if c.narrowed() {
		out = strconv.Itoa(len(rows)-len(s.manual)) + " de " + strconv.Itoa(read)
	}
	if s.paging {
		out += ", leyendo más…"
	}
	return out
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
				if c.all {
					tab += faintStyle.Render(" · todas las ramas")
				} else if sd.branch != "" {
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

// table shows the window of rows around the cursor that fits in n lines; a
// commit that mentions the issue carries a star.
func (c *commits) table(s *side, rows []row, n, width int) []string {
	msgWidth := messageWidth(width)
	var out []string
	start := window(s.cursor, len(rows), n)
	for i := start; i < min(start+n, len(rows)); i++ {
		r := rows[i]
		cursor := "  "
		if i == s.cursor {
			cursor = accentStyle.Render("▸ ")
		}
		star := " "
		if r.suggested {
			star = accentStyle.Render("★")
		}
		message, author, date := faintStyle.Render("(a mano)"), "", ""
		if r.commit != nil {
			message, author, date = r.commit.Subject, r.commit.Author, faintStyle.Render(r.commit.Date.Format("2006-01-02"))
			if i == s.cursor {
				message = accentStyle.Bold(true).Render(message)
			}
		}
		line := cursor + box(s.picked[r.hash] || r.commit == nil) + star + cell(warnStyle.Render(r.hash), hashWidth) + " " + cell(message, msgWidth) + " "
		if withWho(width) {
			line += cell(author, authorWidth) + " " + cell(date, dateWidth) + " "
		}
		out = append(out, line+padLeft(fileCount(s, r.hash), filesWidth))
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
	rows := c.rows(s)
	if s.cursor >= len(rows) {
		return nil
	}
	r := rows[s.cursor]
	var lines []string
	if cm := r.commit; cm != nil {
		meta := slices.DeleteFunc([]string{cm.Author, cm.Date.Format("2006-01-02"), s.branch}, func(p string) bool { return p == "" })
		if c.all || r.suggested {
			meta = slices.DeleteFunc(meta, func(p string) bool { return p == s.branch })
		}
		lines = append(lines, accentStyle.Render(cm.ShortHash)+" "+boldStyle.Render(cm.Subject), faintStyle.Render(strings.Join(meta, " · ")))
		if r.suggested {
			lines = append(lines, accentStyle.Render("★")+faintStyle.Render(" menciona "+c.issue))
		}
	} else {
		lines = append(lines, warnStyle.Render(r.hash)+faintStyle.Render(" (a mano)"))
	}
	fr := s.files[r.hash]
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
	rows := c.rows(s)
	lines := []string{boldStyle.Render(c.issue), c.tabs(inner), ""}
	lines = append(lines, c.state(s, rows, inner)...)

	commitsN, filesN := c.totals()
	files := lipgloss.NewStyle().Width(inner).Render(faintStyle.Render("ARCHIVOS ") + c.cursorFiles(s))
	if fl := strings.Split(files, "\n"); len(fl) > 2 {
		files = strings.Join(append(fl[:1], faintStyle.Render("…")), "\n")
	}
	bottom := []string{"", files, faintStyle.Render("MARCADOS ") + plural(commitsN, "commit", "commits") + " · " + plural(filesN, "archivo", "archivos"),
		lipgloss.NewStyle().Width(inner).Render(c.hint(s, rows))}

	lines = append(lines, faintStyle.Render(c.header(inner)))
	n := max(height-len(lines)-lipgloss.Height(strings.Join(bottom, "\n")), 3)
	for _, r := range c.table(s, rows, n, inner) {
		lines = append(lines, lipgloss.NewStyle().MaxWidth(inner).Render(r))
	}
	return lipgloss.NewStyle().Padding(0, 1).Render(strings.Join(append(lines, bottom...), "\n"))
}

// cursorFiles is the files of the commit under the cursor, in one line.
func (c *commits) cursorFiles(s *side) string {
	hash := c.cursorHash(s)
	fr := s.files[hash]
	switch {
	case hash == "":
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
