package tui

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// formMaxWidth is the widest a form gets, however wide the terminal.
const formMaxWidth = 64

// fieldRuleMark is what huh writes between two fields; formBody turns it into
// a dotted line as wide as the form.
const fieldRuleMark = "┄┄┄"

var fieldRuleLine = regexp.MustCompile(`(?m)^` + fieldRuleMark + ` *$`)

// withFieldRule puts the dotted line under each field but the last, in place
// of the blank line huh leaves between them.
func withFieldRule(base func(bool) *huh.Styles) huh.Theme {
	return huh.ThemeFunc(func(dark bool) *huh.Styles {
		t := base(dark)
		t.FieldSeparator = lipgloss.NewStyle().SetString("\n" + fieldRuleMark + "\n")
		return t
	})
}

func newForm(fields ...huh.Field) *huh.Form { return formOf(huh.NewGroup(fields...)) }

func formOf(groups ...*huh.Group) *huh.Form {
	return huh.NewForm(groups...).WithShowHelp(false).WithTheme(formTheme).WithWidth(formMaxWidth)
}

// page is one page of a paged form, with the name the pager shows.
type page struct {
	name   string
	fields []huh.Field
	hide   func() bool
}

func newPage(name string, fields ...huh.Field) page { return page{name: name, fields: fields} }

// hiddenWhen skips the page while hide is true, as huh's WithHideFunc does.
func (p page) hiddenWhen(hide func() bool) page {
	p.hide = hide
	return p
}

// pager knows the pages of a form, to show which one has the focus.
type pager []page

// newPagedForm is a form of several pages, for what does not fit in 24 lines.
func newPagedForm(pages ...page) (*huh.Form, pager) {
	groups := make([]*huh.Group, len(pages))
	for i, p := range pages {
		groups[i] = huh.NewGroup(p.fields...)
		if p.hide != nil {
			groups[i] = groups[i].WithHideFunc(p.hide)
		}
	}
	return formOf(groups...), pager(pages)
}

// line is "●○○  Página 1 de 3 · name" for the page of the focused field;
// empty with fewer than two pages to show.
func (p pager) line(f *huh.Form) string {
	focused := f.GetFocusedField()
	var shown []page
	current := -1
	for _, pg := range p {
		if pg.hide != nil && pg.hide() {
			continue
		}
		if slices.Contains(pg.fields, focused) {
			current = len(shown)
		}
		shown = append(shown, pg)
	}
	if len(shown) < 2 || current < 0 {
		return ""
	}
	var dots strings.Builder
	for i := range shown {
		if i == current {
			dots.WriteString(accentStyle.Render("●"))
		} else {
			dots.WriteString(faintStyle.Render("○"))
		}
	}
	return dots.String() + "  " + faintStyle.Render(fmt.Sprintf("Página %d de %d ·", current+1, len(shown))) + " " + boldStyle.Render(shown[current].name)
}

// formView is a form with the error of its last submission above it, in
// height lines.
func formView(f *huh.Form, err string, width, height int) string {
	return lipgloss.NewStyle().Padding(1, 2).MaxWidth(width).Render(formBody(f, err, width, height-2))
}

// formView is a paged form under its pager.
func (p pager) formView(f *huh.Form, err string, width, height int) string {
	line := p.line(f)
	if line == "" {
		return formView(f, err, width, height)
	}
	body := lipgloss.JoinVertical(lipgloss.Left, line, "", formBody(f, err, width, height-4))
	return lipgloss.NewStyle().Padding(1, 2).MaxWidth(width).Render(body)
}

// formBody draws the form in height lines, scrolling to keep the focused
// field in sight; 0 draws it whole.
func formBody(f *huh.Form, err string, width, height int) string {
	top, topLines := "", 0
	if err != "" {
		top = lipgloss.NewStyle().Width(max(width-4, 10)).Render(errStyle.Render("✗ " + err))
		topLines = lipgloss.Height(top) + 1
	}
	if height > 0 {
		f.WithHeight(max(height-topLines, minFormHeight))
		f.Update(relayoutMsg{}) // huh scrolls to the focused field only on Update
	}
	rule := "  " + faintStyle.Render(strings.Repeat("┄", max(min(formMaxWidth, width-4)-2, 4)))
	body := fieldRuleLine.ReplaceAllLiteralString(f.View(), rule)
	if top != "" {
		body = lipgloss.JoinVertical(lipgloss.Left, top, "", body)
	}
	return body
}

// minFormHeight keeps one field in sight in a very short terminal.
const minFormHeight = 5

// relayoutMsg is no event: it makes huh lay the form out again.
type relayoutMsg struct{}
