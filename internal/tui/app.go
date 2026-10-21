package tui

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// screen is one page of the visual mode; the app draws the frame around it.
type screen interface {
	title() string
	update(msg tea.Msg) (screen, tea.Cmd)
	view(width, height int) string
	keys() []key.Binding
}

// Optional parts of a screen; see docs/arquitectura/tui.md §5.
type (
	// keyCapturer takes a key away from the global ones (?, esc) while the
	// screen needs it, for example while the user types. ctrl+c is never
	// captured.
	keyCapturer interface{ captures(k tea.KeyPressMsg) bool }
	// initer starts the screen when it is shown: opened, or back on top.
	initer interface{ init() tea.Cmd }
	// closer releases what the screen started (a step's context) when it is
	// closed.
	closer interface{ close() }
)

type openMsg struct{ s screen }
type backMsg struct{}
type backToMsg struct{ s screen }
type homeMsg struct{}

// open stacks s over the current screen.
func open(s screen) tea.Cmd { return func() tea.Msg { return openMsg{s} } }

// back closes the current screen.
func back() tea.Msg { return backMsg{} }

// backTo closes the screens above s.
func backTo(s screen) tea.Cmd { return func() tea.Msg { return backToMsg{s} } }

// toHome closes every screen down to the home one.
func toHome() tea.Msg { return homeMsg{} }

var (
	quitKey = key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "salir"))
	helpKey = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "ayuda"))
	backKey = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "volver"))
)

// app is the root model: a stack of screens inside a common frame.
type app struct {
	st            *state
	stack         []screen
	width, height int
	help          help.Model
}

func newApp(d Deps) *app {
	st := loadState(d)
	return &app{st: st, stack: []screen{newHome(st)}, help: help.New()}
}

func (m *app) Init() tea.Cmd { return nil }

func (m *app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.help.SetWidth(msg.Width - 1) // the footer starts with a space
		return m, nil
	case tea.KeyPressMsg:
		if key.Matches(msg, quitKey) {
			return m, tea.Quit
		}
		if !m.captured(msg) {
			switch {
			case key.Matches(msg, helpKey):
				m.help.ShowAll = !m.help.ShowAll
				return m, nil
			case key.Matches(msg, backKey) && len(m.stack) > 1:
				return m, back
			}
		}
	case openMsg:
		m.stack = append(m.stack, msg.s)
		if i, ok := msg.s.(initer); ok {
			return m, i.init()
		}
		return m, nil
	case backMsg:
		if len(m.stack) > 1 {
			m.pop()
			if i, ok := m.top().(initer); ok {
				return m, i.init()
			}
		}
		return m, nil
	case backToMsg:
		for len(m.stack) > 1 && m.top() != msg.s {
			m.pop()
		}
		if i, ok := m.top().(initer); ok {
			return m, i.init()
		}
		return m, nil
	case homeMsg:
		for len(m.stack) > 1 {
			m.pop()
		}
		return m, nil
	}
	top, cmd := m.top().update(msg)
	m.stack[len(m.stack)-1] = top
	return m, cmd
}

func (m *app) top() screen { return m.stack[len(m.stack)-1] }

// captured reports whether the screen on top takes k from the frame.
func (m *app) captured(k tea.KeyPressMsg) bool {
	c, ok := m.top().(keyCapturer)
	return ok && c.captures(k)
}

func (m *app) pop() {
	if c, ok := m.top().(closer); ok {
		c.close()
	}
	m.stack = m.stack[:len(m.stack)-1]
}

func (m *app) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.WindowTitle = "draco"
	return v
}

func (m *app) render() string {
	if m.width == 0 {
		return ""
	}
	header, footer := m.header(), m.footer()
	h := max(m.height-lipgloss.Height(header)-lipgloss.Height(footer), 0)
	body := lipgloss.NewStyle().Width(m.width).Height(h).MaxHeight(h).Render(m.top().view(m.width, h))
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m *app) header() string {
	left := boldStyle.Render("draco") + " " + faintStyle.Render(m.st.deps.Version)
	var parts []string
	if m.st.profile != "" {
		parts = append(parts, "perfil: "+accentStyle.Render(m.st.profile)+faintStyle.Render(" ("+m.st.tracker+" → "+m.st.docs+")"))
	}
	if m.st.project != "" {
		parts = append(parts, "proyecto: "+accentStyle.Render(m.st.project))
	}
	right := strings.Join(parts, "  ")
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		right, gap = "", m.width-lipgloss.Width(left)-2
	}
	line := " " + left + strings.Repeat(" ", max(gap, 1)) + right + " "
	out := lipgloss.NewStyle().MaxWidth(m.width).Render(line) + "\n" + m.rule()
	if len(m.stack) > 1 {
		out += "\n" + lipgloss.NewStyle().MaxWidth(m.width).Render(" "+m.crumbs())
	}
	return out
}

// crumbs is the path of open screens, the current one highlighted.
func (m *app) crumbs() string {
	var parts []string
	for i, s := range m.stack {
		if i == len(m.stack)-1 {
			parts = append(parts, boldStyle.Render(s.title()))
		} else {
			parts = append(parts, faintStyle.Render(s.title()))
		}
	}
	return strings.Join(parts, faintStyle.Render(" › "))
}

func (m *app) footer() string {
	bindings := m.top().keys()
	if len(m.stack) > 1 && !m.captured(tea.KeyPressMsg{Code: tea.KeyEscape}) {
		bindings = append(bindings, backKey)
	}
	if !m.captured(tea.KeyPressMsg{Code: '?', Text: "?"}) {
		bindings = append(bindings, helpKey)
	}
	bindings = append(bindings, quitKey)
	return m.rule() + "\n " + m.help.View(keyMap(bindings))
}

func (m *app) rule() string { return ruleStyle.Render(strings.Repeat("─", m.width)) }

// keyMap adapts a screen's bindings to the help bubble.
type keyMap []key.Binding

func (k keyMap) ShortHelp() []key.Binding  { return k }
func (k keyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k} }
