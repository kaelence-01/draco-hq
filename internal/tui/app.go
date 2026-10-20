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
	update(msg tea.Msg) (screen, tea.Cmd)
	view(width, height int) string
	keys() []key.Binding
}

type openMsg struct{ s screen }
type backMsg struct{}

// open stacks s over the current screen.
func open(s screen) tea.Cmd { return func() tea.Msg { return openMsg{s} } }

// back closes the current screen.
func back() tea.Msg { return backMsg{} }

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
		m.help.SetWidth(msg.Width)
		return m, nil
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, quitKey):
			return m, tea.Quit
		case key.Matches(msg, helpKey):
			m.help.ShowAll = !m.help.ShowAll
			return m, nil
		case key.Matches(msg, backKey) && len(m.stack) > 1:
			return m, back
		}
	case openMsg:
		m.stack = append(m.stack, msg.s)
		return m, nil
	case backMsg:
		if len(m.stack) > 1 {
			m.stack = m.stack[:len(m.stack)-1]
		}
		return m, nil
	}
	top, cmd := m.top().update(msg)
	m.stack[len(m.stack)-1] = top
	return m, cmd
}

func (m *app) top() screen { return m.stack[len(m.stack)-1] }

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
	return lipgloss.NewStyle().MaxWidth(m.width).Render(line) + "\n" + m.rule()
}

func (m *app) footer() string {
	bindings := m.top().keys()
	if len(m.stack) > 1 {
		bindings = append(bindings, backKey)
	}
	bindings = append(bindings, helpKey, quitKey)
	return m.rule() + "\n " + m.help.View(keyMap(bindings))
}

func (m *app) rule() string { return ruleStyle.Render(strings.Repeat("─", m.width)) }

// keyMap adapts a screen's bindings to the help bubble.
type keyMap []key.Binding

func (k keyMap) ShortHelp() []key.Binding  { return k }
func (k keyMap) FullHelp() [][]key.Binding { return [][]key.Binding{k} }
