package tui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/profile"
	"github.com/kaelence-01/draco-hq/internal/provider"
	"github.com/kaelence-01/draco-hq/internal/qa"
)

// qaForm asks for the sprint and module; without a sprint, Kanban mode.
type qaForm struct {
	st     *state
	sprint string
	module string
	form   *huh.Form
	err    string
}

func newQAForm(st *state) *qaForm {
	f := &qaForm{st: st}
	f.form = f.build()
	return f
}

func (f *qaForm) build() *huh.Form {
	return newForm(
		huh.NewInput().Title("Sprint").Placeholder("12 (vacío: modo Kanban, últimos 10 días hábiles)").Value(&f.sprint).Validate(validSprint),
		huh.NewInput().Title("Módulo").Placeholder("Pagos (solo en modo Sprint)").Value(&f.module),
	)
}

// validSprint accepts a positive number, or nothing for Kanban mode.
func validSprint(s string) error {
	if s = strings.TrimSpace(s); s == "" {
		return nil
	}
	if n, err := strconv.Atoi(s); err != nil || n < 1 {
		return errors.New("el sprint debe ser un número positivo")
	}
	return nil
}

func (f *qaForm) title() string       { return "Consolidado QA" }
func (f *qaForm) keys() []key.Binding { return []key.Binding{nextFieldKey, prevFieldKey} }

func (f *qaForm) captures(k tea.KeyPressMsg) bool { return !key.Matches(k, backKey) }

func (f *qaForm) init() tea.Cmd {
	if f.form.State != huh.StateNormal {
		f.form = f.build()
	}
	return f.form.Init()
}

func (f *qaForm) update(msg tea.Msg) (screen, tea.Cmd) {
	m, cmd := f.form.Update(msg)
	f.form = m.(*huh.Form)
	if f.form.State != huh.StateCompleted {
		return f, cmd
	}
	next, err := f.next()
	if err != nil {
		f.err = err.Error()
		f.form = f.build()
		return f, f.form.Init()
	}
	f.err = ""
	return f, open(next)
}

func (f *qaForm) next() (screen, error) {
	if f.st.cfg == nil {
		return nil, errors.New(f.st.notice)
	}
	sprint, _ := strconv.Atoi(strings.TrimSpace(f.sprint))
	bundle, err := f.st.deps.OpenProfile(f.st.cfg, profile.Options{})
	if err != nil {
		return nil, explainProfile(err)
	}
	tracker, err := bundle.QA()
	if errors.Is(err, profile.ErrNoQA) {
		return nil, fmt.Errorf("%w: usa un perfil con tracker jira", err)
	}
	if err != nil {
		return nil, fmt.Errorf("%w\nCompleta lo que falta en Configuración › QA", err)
	}
	target, err := qa.Resolve(f.st.cfg, qa.Request{Sprint: sprint, Module: f.module, Template: bundle.Profile.TemplateDir("qa")}, f.st.deps.Now())
	if err != nil {
		return nil, err
	}
	return newQAProgress(f.st, bundle, qa.NewSession(tracker, bundle.Docs, target)), nil
}

func (f *qaForm) view(width, height int) string { return formView(f.form, f.err, width) }

type qaStep int

const (
	qaAccount qaStep = iota
	qaCollect
	qaEvaluate
	qaFind
	qaDone
)

type qaStepMsg struct {
	owner    *qaProgress
	step     qaStep
	task     int // the task evaluated, in qaEvaluate
	warnings []string
	err      error
}

// qaProgress checks the account, collects the tasks and evaluates them one by
// one, adding each row as it arrives. It keeps its own copy of the rows: the
// step running writes the Session from its goroutine.
type qaProgress struct {
	st       *state
	bundle   *profile.Bundle
	session  *qa.Session
	step     qaStep
	err      error
	account  string
	rows     []provider.QAIssue // collected; the first done are evaluated
	own      int                // tasks of the QA account
	done     int                // rows evaluated
	warnings []string
	notes    map[int][]string // the warnings of each task, to show with it
	existing *provider.DocRef
	cursor   int
	started  bool
	spinner  spinner.Model
	ctx      context.Context
	cancel   context.CancelFunc
}

func newQAProgress(st *state, bundle *profile.Bundle, s *qa.Session) *qaProgress {
	p := &qaProgress{st: st, bundle: bundle, session: s, notes: map[int][]string{}, spinner: spinner.New(spinner.WithSpinner(spinner.MiniDot))}
	p.ctx, p.cancel = context.WithCancel(context.Background())
	return p
}

func (p *qaProgress) title() string { return "Evaluar" }
func (p *qaProgress) close()        { p.cancel() }

func (p *qaProgress) keys() []key.Binding {
	switch {
	case p.step != qaDone:
		return nil
	case len(p.rows) > 0:
		return []key.Binding{moveKey, previewKey}
	}
	return []key.Binding{previewKey}
}

func (p *qaProgress) moreKeys() []key.Binding {
	if p.step != qaDone || len(p.rows) == 0 {
		return nil
	}
	return []key.Binding{upKey, downKey}
}

func (p *qaProgress) init() tea.Cmd {
	if p.started {
		return nil
	}
	p.started = true
	return tea.Batch(p.spinner.Tick, p.run(qaAccount, 0))
}

func (p *qaProgress) run(st qaStep, task int) tea.Cmd {
	p.step = st
	s, ctx := p.session, p.ctx
	return func() tea.Msg {
		msg := qaStepMsg{owner: p, step: st, task: task}
		switch st {
		case qaAccount:
			msg.err = s.CheckAccount(ctx)
		case qaCollect:
			msg.err = s.Collect(ctx)
		case qaEvaluate:
			msg.warnings = s.EvaluateTask(ctx, task)
			msg.err = ctx.Err()
		case qaFind:
			msg.err = s.FindExisting(ctx)
		}
		return msg
	}
}

func (p *qaProgress) update(msg tea.Msg) (screen, tea.Cmd) {
	switch msg := msg.(type) {
	case qaStepMsg:
		if msg.owner != p || p.ctx.Err() != nil {
			return p, nil
		}
		return p, p.finished(msg)
	case spinner.TickMsg:
		if p.step == qaDone || p.err != nil {
			return p, nil
		}
		var cmd tea.Cmd
		p.spinner, cmd = p.spinner.Update(msg)
		return p, cmd
	case tea.KeyPressMsg:
		if p.step != qaDone {
			return p, nil
		}
		switch {
		case key.Matches(msg, upKey):
			p.cursor = max(p.cursor-1, 0)
		case key.Matches(msg, downKey):
			p.cursor = min(p.cursor+1, max(len(p.rows)-1, 0))
		case key.Matches(msg, previewKey):
			return p, open(newPreview(&qaRun{bundle: p.bundle, session: p.session, command: p.command()}))
		}
	}
	return p, nil
}

// finished takes in what a step found and starts the next one.
func (p *qaProgress) finished(msg qaStepMsg) tea.Cmd {
	s := p.session
	if msg.err != nil {
		p.err = msg.err
		if msg.step == qaAccount && errors.Is(msg.err, qa.ErrNotQAAccount) {
			p.err = notQAAccount(msg.err, s.Account, s.Target.QAEmail)
		}
		return nil
	}
	switch msg.step {
	case qaAccount:
		p.account = s.Account.Email
		return p.run(qaCollect, 0)
	case qaCollect:
		p.rows, p.own = slices.Clone(s.Tasks), len(s.QATasks)
		if len(p.rows) == 0 {
			return p.run(qaFind, 0)
		}
		return p.run(qaEvaluate, 0)
	case qaEvaluate:
		p.rows[msg.task] = s.Tasks[msg.task]
		p.done = msg.task + 1
		p.cursor = msg.task
		p.warnings = append(p.warnings, msg.warnings...)
		if len(msg.warnings) > 0 {
			p.notes[msg.task] = msg.warnings
		}
		if p.done < len(p.rows) {
			return p.run(qaEvaluate, p.done)
		}
		return p.run(qaFind, 0)
	case qaFind:
		p.existing = s.Existing
		p.step = qaDone
	}
	return nil
}

// notQAAccount says which account answered and where to change qa_email.
func notQAAccount(err error, me *provider.User, configured string) error {
	if configured == "" {
		configured = "(sin configurar)"
	}
	where := "la configuración"
	if path, perr := config.ConfigPath(); perr == nil {
		where = path
	}
	email := ""
	if me != nil {
		email = me.Email
	}
	return fmt.Errorf("%w.\nCuenta autenticada: %s\nqa_email del config: %s\nSi te corresponde generar el consolidado, ajusta qa_email en Configuración › QA (%s)", err, email, configured, where)
}

// command is the same report in the command mode.
func (p *qaProgress) command() string {
	t := p.session.Target
	if t.Kanban {
		return "draco qa   (Enter en el sprint: modo Kanban)"
	}
	return "draco qa -s " + strconv.Itoa(t.Sprint) + " -m " + shellQuote(t.Module)
}

// mode is what the report covers: a sprint of a module, or the Kanban window.
func (p *qaProgress) mode() string {
	t := p.session.Target
	if t.Kanban {
		return "Kanban · " + t.Period
	}
	return "Sprint " + strconv.Itoa(t.Sprint) + " · " + t.Module
}

func (p *qaProgress) view(width, height int) string {
	if width >= panelMinWidth && p.step >= qaEvaluate && len(p.rows) > 0 {
		if out, ok := p.wide(width, height); ok {
			return out
		}
	}
	return p.narrow(width, height)
}

// narrow is the list of tasks with what the user needs under it, one block
// after the other.
func (p *qaProgress) narrow(width, height int) string {
	t := p.session.Target
	wrap := lipgloss.NewStyle().Width(width - 2)
	head := []string{boldStyle.Render(p.mode()) + faintStyle.Render("  "+t.Title)}
	for _, w := range t.Warnings {
		head = append(head, wrap.Render(warnStyle.Render("⚠ "+w)))
	}
	head = append(head, "")

	if p.step == qaAccount {
		head = append(head, p.status("Verificando la cuenta de QA"))
		return pad(lipgloss.JoinVertical(lipgloss.Left, head...))
	}
	account := okStyle.Render("✓ ") + "Cuenta " + p.account
	if p.step == qaCollect {
		head = append(head, account, p.status("Buscando tareas"))
		return pad(lipgloss.JoinVertical(lipgloss.Left, head...))
	}
	own := faintStyle.Render(" · " + plural(p.own, "tarea QA propia", "tareas QA propias"))
	if len(p.rows) == 0 {
		head = append(head, account, wrap.Render(warnStyle.Render("⚠ No hay tareas en revisión para este período")+own))
	} else {
		head = append(head, wrap.Render(account+" · "+plural(len(p.rows), "tarea", "tareas")+" para revisión"+own))
		if p.step >= qaEvaluate {
			head = append(head, wrap.Render(p.counters("  ")))
		}
	}

	var foot []string
	for _, w := range p.warnings {
		foot = append(foot, wrap.Render(warnStyle.Render("⚠ "+w)))
	}
	switch {
	case p.step == qaEvaluate:
		foot = append(foot, p.status("Evaluando "+p.rows[p.done].Key+" ("+strconv.Itoa(p.done+1)+" de "+strconv.Itoa(len(p.rows))+")"))
	case p.step == qaFind:
		foot = append(foot, p.status("Buscando el consolidado publicado"))
	case p.step == qaDone:
		foot = append(foot, p.detail(wrap)...)
		foot = append(foot, "", p.outcome(wrap), faintStyle.Render("enter abre la vista previa; nada se publica hasta que lo confirmes."))
	}

	room := max(height-lipgloss.Height(strings.Join(head, "\n"))-lipgloss.Height(strings.Join(foot, "\n"))-4, 3)
	body := append(head, "")
	if len(p.rows) > 0 {
		body = append(body, p.table(room, width-2)...)
	}
	return pad(lipgloss.JoinVertical(lipgloss.Left, append(append(body, ""), foot...)...))
}

// status is the line of the step running, or its error.
func (p *qaProgress) status(running string) string {
	if p.err != nil {
		return errStyle.Render("✗ ") + running + "\n  " + errStyle.Render(strings.ReplaceAll(p.err.Error(), "\n", "\n  "))
	}
	return p.spinner.View() + " " + running + "…"
}

// table is one row per task, with the marks of the ones evaluated.
func (p *qaProgress) table(height, width int) []string {
	row := lipgloss.NewStyle().MaxWidth(width)
	keyWidth := len("Tarea")
	for _, r := range p.rows {
		keyWidth = max(keyWidth, lipgloss.Width(r.Key))
	}
	out := []string{row.Render(faintStyle.Render("  " + padRight("Tarea", keyWidth) + "  Err Dev Doc PR   Resumen"))}
	rows := max(height-1, 1)
	start := window(p.cursor, len(p.rows), rows)
	for i := start; i < min(start+rows, len(p.rows)); i++ {
		r := p.rows[i]
		mark := "  "
		if i == p.cursor && p.step == qaDone {
			mark = accentStyle.Render("▸ ")
		}
		marks := faintStyle.Render(" ·   ·   ·   · ")
		if i < p.done {
			marks = " " + check(!r.HasCodingErrors) + "   " + check(!r.HasDevReturns) + "   " + docMark(r) + "   " + check(r.PRMerged) + " "
		}
		out = append(out, row.Render(mark+padRight(r.Key, keyWidth)+"  "+marks+"  "+r.Summary))
	}
	return out
}

func padRight(s string, n int) string { return s + strings.Repeat(" ", max(n-lipgloss.Width(s), 0)) }

func check(ok bool) string {
	if ok {
		return okStyle.Render("✓")
	}
	return errStyle.Render("✗")
}

// docMark is the deploy document check; ? when the lookup failed.
func docMark(r provider.QAIssue) string {
	if r.DeployDocUnknown {
		return warnStyle.Render("?")
	}
	return check(r.HasDeployDoc)
}

// detail is the review task and observations of the task under the cursor.
func (p *qaProgress) detail(wrap lipgloss.Style) []string {
	if len(p.rows) == 0 {
		return nil
	}
	r := p.rows[min(p.cursor, len(p.rows)-1)]
	line := boldStyle.Render(r.Key)
	if r.ReviewTaskKey != "" {
		line += faintStyle.Render(" · revisión ") + r.ReviewTaskKey
	}
	out := []string{wrap.Render(line)}
	if r.Observations != "" {
		out = append(out, wrap.Render(faintStyle.Render("Observaciones: ")+r.Observations))
	}
	return out
}

// counters is how the tasks stand: the evaluated ones classified by
// qa.Tally, the rest pending.
func (p *qaProgress) counters(sep string) string {
	c := qa.Tally(p.rows, p.done)
	return strings.Join([]string{
		boldStyle.Render(plural(c.Total, "tarea", "tareas")),
		okStyle.Render("✓ " + plural(c.Complete, "completa", "completas")),
		errStyle.Render("✗ " + strconv.Itoa(c.Failing) + " con fallas"),
		warnStyle.Render("? " + strconv.Itoa(c.Unverified) + " sin verificar"),
		faintStyle.Render("· " + strconv.Itoa(c.Pending()) + " por evaluar"),
	}, sep)
}

// wide is the counters and the table of tasks across the screen and, under
// them, the task under the cursor beside the report. It reports false when
// the height is too short for both.
func (p *qaProgress) wide(width, height int) (string, bool) {
	t := p.session.Target
	inner := width - 2
	steps := slices.Clone(qaSteps)
	if p.step == qaEvaluate {
		steps[1] = "Evaluar " + strconv.Itoa(p.done) + "/" + strconv.Itoa(len(p.rows))
	}
	wrap := lipgloss.NewStyle().Width(inner)
	head := []string{titleLine(boldStyle.Render(p.mode()), steps, 1, inner)}
	for _, w := range t.Warnings {
		head = append(head, wrap.Render(warnStyle.Render("⚠ "+w)))
	}
	own := faintStyle.Render(" · " + plural(p.own, "tarea QA propia", "tareas QA propias"))
	head = append(head, wrap.Render(okStyle.Render("✓ ")+"Cuenta "+p.account+" · "+plural(len(p.rows), "tarea", "tareas")+" para revisión"+own), "")
	headBlock := strings.Join(head, "\n")

	listWidth, sideWidth := columns(inner, 2, 6, 4)
	detail, side := p.detailText(listWidth-4), p.consolidated(sideWidth-4)
	natural := max(lipgloss.Height(detail), lipgloss.Height(side)) + 2
	room := height - lipgloss.Height(headBlock) - 4 // the table's border, counters and header
	rows := min(len(p.rows), max(room-min(natural, 11), 3))
	bottom := room - rows
	if bottom < 6 {
		return "", false
	}
	cols := lipgloss.JoinHorizontal(lipgloss.Top, panel(detail, listWidth, bottom), "  ", panel(side, sideWidth, bottom))
	return lipgloss.NewStyle().PaddingLeft(1).Render(lipgloss.JoinVertical(lipgloss.Left, headBlock, p.wideTable(rows, inner), cols)), true
}

// wideTable is the counters and one row per task, with its marks and its
// state, in a box as wide as the screen.
func (p *qaProgress) wideTable(rows, width int) string {
	const marksWidth, stateWidth = 20, 14
	keyWidth := len("TAREA")
	for _, r := range p.rows {
		keyWidth = max(keyWidth, lipgloss.Width(r.Key))
	}
	summaryWidth := max(width-4-(2+keyWidth+2+marksWidth+2+2+stateWidth), 10)
	line := func(prefix, key, marks, summary, state string) string {
		return prefix + padRight(key, keyWidth) + "  " + marks + "  " + cell(summary, summaryWidth) + "  " + state
	}
	lines := []string{p.counters("      "), faintStyle.Render(line("  ", "TAREA", padRight("ERR", 5)+padRight("DEV", 5)+padRight("DOC", 5)+padRight("PR", 5), "RESUMEN", "ESTADO"))}
	start := window(p.cursor, len(p.rows), rows)
	for i := start; i < min(start+rows, len(p.rows)); i++ {
		r := p.rows[i]
		cursor, key, summary := "  ", accentStyle.Render(r.Key), r.Summary
		if i == p.cursor {
			cursor, key, summary = accentStyle.Render("▸ "), accentStyle.Bold(true).Render(r.Key), boldStyle.Render(r.Summary)
		}
		marks := faintStyle.Render(padRight(" ·", 5) + padRight(" ·", 5) + padRight(" ·", 5) + padRight(" ·", 5))
		if i < p.done {
			marks = padRight(" "+check(!r.HasCodingErrors), 5) + padRight(" "+check(!r.HasDevReturns), 5) + padRight(" "+docMark(r), 5) + padRight(" "+check(r.PRMerged), 5)
		}
		lines = append(lines, line(cursor, key, marks, summary, p.stateCell(i)))
	}
	return boxStyle.Width(width).Render(strings.Join(lines, "\n"))
}

// stateCell is where a task stands: what its checks say, or that it is being
// evaluated or waits its turn.
func (p *qaProgress) stateCell(i int) string {
	switch {
	case i < p.done:
		switch qa.StatusOf(p.rows[i]) {
		case qa.Complete:
			return okStyle.Render("completa")
		case qa.Failing:
			return errStyle.Render("con fallas")
		}
		return warnStyle.Render("sin verificar")
	case i == p.done && p.step == qaEvaluate:
		return accentStyle.Render(p.spinner.View() + " evaluando…")
	}
	return faintStyle.Render("por evaluar")
}

// detailText is the task under the cursor and what each of its checks found.
// The tracker answers yes or no, so there are no counts of errors or returns.
func (p *qaProgress) detailText(width int) string {
	i := min(p.cursor, len(p.rows)-1)
	r := p.rows[i]
	lines := []string{accentStyle.Render(r.Key) + " " + boldStyle.Render(r.Summary)}
	if r.ReviewTaskKey != "" {
		lines = append(lines, faintStyle.Render("Revisión QA "+r.ReviewTaskKey))
	}
	if i >= p.done {
		return strings.Join(append(lines, "", faintStyle.Render("Todavía sin evaluar.")), "\n")
	}
	doc := "sin documento de despliegue"
	switch {
	case r.DeployDocUnknown:
		doc = warnStyle.Render("no se pudo verificar")
	case r.HasDeployDoc:
		doc = "enlazado"
	}
	lines = append(lines, "",
		fact(check(!r.HasCodingErrors), "Errores de código", pick(r.HasCodingErrors, "con errores registrados", "ninguno"), width),
		fact(check(!r.HasDevReturns), "Devoluciones", pick(r.HasDevReturns, "con devoluciones a desarrollo", "ninguna"), width),
		fact(docMark(r), "Documento", doc, width),
		fact(check(r.PRMerged), "Pull request", pick(r.PRMerged, "aprobado", "sin aprobar"), width), "")
	if r.Observations != "" {
		lines = append(lines, hang(faintStyle.Render("Observaciones: "), r.Observations, width))
	} else {
		lines = append(lines, faintStyle.Render("Observaciones: ninguna"))
	}
	for _, w := range p.notes[i] {
		lines = append(lines, warnStyle.Render("⚠ "+w))
	}
	return strings.Join(lines, "\n")
}

func pick(yes bool, a, b string) string {
	if yes {
		return a
	}
	return b
}

// fact is one check of a task: its mark, its name and what it found.
func fact(mark, label, value string, width int) string {
	return hang(mark+" "+padRight(label, 19), faintStyle.Render(value), width)
}

// consolidated is the report these tasks make and what happens to it next.
func (p *qaProgress) consolidated(width int) string {
	lines := []string{faintStyle.Render("CONSOLIDADO"), p.session.Target.Title, ""}
	wrap := lipgloss.NewStyle().Width(width)
	switch p.step {
	case qaEvaluate:
		lines = append(lines, p.status("Evaluando "+p.rows[p.done].Key+" ("+strconv.Itoa(p.done+1)+" de "+strconv.Itoa(len(p.rows))+")"))
	case qaFind:
		lines = append(lines, p.status("Buscando el consolidado publicado"))
	case qaDone:
		lines = append(lines, p.outcome(wrap), "", faintStyle.Render("enter abre la vista previa; nada se publica hasta que lo confirmes."))
	}
	return strings.Join(lines, "\n")
}

// outcome says what publishing would do.
func (p *qaProgress) outcome(wrap lipgloss.Style) string {
	if p.existing != nil {
		return wrap.Render(okStyle.Render("✓ ") + "Ya existe «" + p.existing.Title + "»: se actualizará conservando lo escrito a mano.")
	}
	return wrap.Render(okStyle.Render("✓ ") + "No hay un consolidado publicado: se creará «" + p.session.Target.Title + "».")
}
