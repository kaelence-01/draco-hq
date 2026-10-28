package tui

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
)

const (
	minColumn     = 4
	maxWord       = 14 // the longest word a column holds whole, when there is room
	columnGap     = 2
	mdPunctuation = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
)

var (
	headingRe   = regexp.MustCompile(`^(#{1,6})\s+(.*)$`)
	directiveRe = regexp.MustCompile(`^<!--\s*draco:(tabla|recuadro)((?:\s+[a-z]+=\S+)*)\s*-->$`)
	separatorRe = regexp.MustCompile(`^:?-+:?$`)
	orderedRe   = regexp.MustCompile(`^\d+[.)]\s`)
	taskRe      = regexp.MustCompile(`^\[([ xX])\]\s`)
	brRe        = regexp.MustCompile(`(?i)<br\s*/?>`)
)

// renderMarkdown draws the Markdown draco generates for a terminal width
// columns wide: headings, tables with their column widths, lists, task boxes,
// links, bold and italic, and the draco comments (which are not shown). A
// line it does not know is shown as it is.
func renderMarkdown(md string, width int) string {
	r := &mdRenderer{width: max(width, 20)}
	r.render(strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n"))
	return strings.Join(r.blocks, "\n\n")
}

type mdRenderer struct {
	width   int
	blocks  []string
	weights []int // the column widths of the next table, from its draco comment
	boxed   bool  // the next block goes in a box
}

func (r *mdRenderer) render(lines []string) {
	for i := 0; i < len(lines); {
		line := strings.TrimSpace(lines[i])
		switch {
		case line == "":
			i++
		case directiveRe.MatchString(line):
			r.directive(line)
			i++
		case headingRe.MatchString(line):
			r.heading(line)
			i++
		case isTableStart(lines, i):
			i = r.table(lines, i)
		case listMarker(line) > 0:
			i = r.list(lines, i)
		case isUnknown(lines[i]):
			r.emit(lipgloss.NewStyle().Width(r.w()).Render(lines[i]))
			i++
		default:
			i = r.paragraph(lines, i)
		}
	}
}

// w is the width a block can use: a box takes four columns for its border and
// padding.
func (r *mdRenderer) w() int {
	if r.boxed {
		return r.width - 4
	}
	return r.width
}

// emit adds a block, in a box as wide as its content when a recuadro comment
// asked for it.
func (r *mdRenderer) emit(block string) {
	if r.boxed {
		lines := strings.Split(block, "\n")
		for i := range lines {
			lines[i] = strings.TrimRight(lines[i], " ")
		}
		block, r.boxed = boxStyle.Render(strings.Join(lines, "\n")), false
	}
	r.blocks = append(r.blocks, block)
}

func (r *mdRenderer) directive(line string) {
	m := directiveRe.FindStringSubmatch(line)
	if m[1] == "recuadro" {
		r.boxed = true
		return
	}
	r.weights = nil
	for _, kv := range strings.Fields(m[2]) {
		if v, ok := strings.CutPrefix(kv, "anchos="); ok {
			for _, n := range strings.Split(v, ",") {
				w, err := strconv.Atoi(n)
				if err != nil || w <= 0 {
					r.weights = nil
					return
				}
				r.weights = append(r.weights, w)
			}
		}
	}
}

func (r *mdRenderer) heading(line string) {
	m := headingRe.FindStringSubmatch(line)
	style := boldStyle
	if len(m[1]) == 2 {
		style = accentStyle.Bold(true)
	}
	r.emit(lipgloss.NewStyle().Width(r.w()).Render(style.Render(inline(m[2]))))
}

// isUnknown is a line this renderer does not draw: it is shown as it is.
func isUnknown(line string) bool {
	trim := strings.TrimSpace(line)
	return strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") || strings.HasPrefix(trim, ">") ||
		strings.HasPrefix(trim, "```") || strings.HasPrefix(trim, "~~~") || strings.HasPrefix(trim, "<") ||
		orderedRe.MatchString(trim) || isRule(trim) || strings.HasPrefix(trim, "|")
}

// isRule is a thematic break: three or more of the same -, * or _.
func isRule(s string) bool {
	s = strings.ReplaceAll(s, " ", "")
	return len(s) >= 3 && strings.Trim(s, s[:1]) == "" && strings.Contains("-*_", s[:1])
}

// listMarker is the length of the bullet that starts an item, 0 if the line
// is not one.
func listMarker(trim string) int {
	switch {
	case trim == "-" || trim == "*" || trim == "+":
		return 1
	case len(trim) > 1 && strings.Contains("-*+", trim[:1]) && trim[1] == ' ' && !isRule(trim):
		return 2
	}
	return 0
}

func (r *mdRenderer) list(lines []string, i int) int {
	var rows []string
	for ; i < len(lines); i++ {
		raw := lines[i]
		trim := strings.TrimSpace(raw)
		n := listMarker(trim)
		if n == 0 {
			break
		}
		indent := strings.Repeat(" ", (len(raw)-len(strings.TrimLeft(raw, " ")))/2*2)
		text, prefix := strings.TrimSpace(trim[n:]), faintStyle.Render("· ")
		if m := taskRe.FindStringSubmatch(text + " "); m != nil {
			text, prefix = strings.TrimSpace(text[3:]), box(m[1] != " ")+" "
		}
		rows = append(rows, hang(indent+prefix, inline(text), r.w()))
	}
	r.emit(strings.Join(rows, "\n"))
	return i
}

func (r *mdRenderer) paragraph(lines []string, i int) int {
	var words []string
	for ; i < len(lines); i++ {
		trim := strings.TrimSpace(lines[i])
		if trim == "" || (len(words) > 0 && r.startsBlock(lines, i)) {
			break
		}
		words = append(words, trim)
	}
	r.emit(lipgloss.NewStyle().Width(r.w()).Render(inline(strings.Join(words, " "))))
	return i
}

// startsBlock is a line that cannot continue a paragraph.
func (r *mdRenderer) startsBlock(lines []string, i int) bool {
	trim := strings.TrimSpace(lines[i])
	return directiveRe.MatchString(trim) || headingRe.MatchString(trim) || isTableStart(lines, i) || listMarker(trim) > 0 || isUnknown(lines[i])
}

// isTableStart is a row of cells followed by the line of dashes.
func isTableStart(lines []string, i int) bool {
	if i+1 >= len(lines) || !strings.HasPrefix(strings.TrimSpace(lines[i]), "|") {
		return false
	}
	sep := splitRow(lines[i+1])
	if !strings.HasPrefix(strings.TrimSpace(lines[i+1]), "|") || len(sep) == 0 {
		return false
	}
	for _, c := range sep {
		if !separatorRe.MatchString(c) {
			return false
		}
	}
	return true
}

// splitRow is the cells of a table row; an escaped pipe belongs to its cell.
func splitRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	if strings.HasSuffix(line, "|") && !strings.HasSuffix(line, `\|`) {
		line = strings.TrimSuffix(line, "|")
	}
	var cells []string
	var cur strings.Builder
	for i := 0; i < len(line); i++ {
		switch {
		case line[i] == '\\' && i+1 < len(line):
			cur.WriteString(line[i : i+2])
			i++
		case line[i] == '|':
			cells = append(cells, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteByte(line[i])
		}
	}
	return append(cells, strings.TrimSpace(cur.String()))
}

// mdCell is the content of a table cell: its lines, already styled. A cell
// that is only a link keeps its URL, to show its tail when the column is
// narrow.
type mdCell struct {
	lines []string
	url   string
}

var urlCellRe = regexp.MustCompile(`^<(https?://[^>\s]+)>$`)

func parseCell(text string, head bool) mdCell {
	if m := urlCellRe.FindStringSubmatch(text); m != nil && !head {
		return mdCell{lines: []string{accentStyle.Render(m[1])}, url: m[1]}
	}
	switch {
	case text == "":
		return mdCell{}
	case head:
		return mdCell{lines: []string{faintStyle.Bold(true).Render(inline(text))}}
	case text == "✔":
		return mdCell{lines: []string{okStyle.Render("✓")}}
	case text == "✖":
		return mdCell{lines: []string{errStyle.Render("✗")}}
	case text == "?":
		return mdCell{lines: []string{warnStyle.Render("?")}}
	case strings.HasPrefix(text, "**") && strings.HasSuffix(text, "**") && len(text) > 4 && !strings.Contains(text[2:len(text)-2], "**"):
		return mdCell{lines: []string{faintStyle.Render(inline(text[2 : len(text)-2]))}}
	}
	parts := brRe.Split(text, -1)
	bullets := true
	for _, p := range parts {
		bullets = bullets && strings.HasPrefix(strings.TrimSpace(p), "- ")
	}
	c := mdCell{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if bullets {
			p = faintStyle.Render("· ") + inline(strings.TrimPrefix(p, "- "))
		} else {
			p = inline(p)
		}
		c.lines = append(c.lines, p)
	}
	return c
}

func (c mdCell) width() int {
	w := 0
	for _, l := range c.lines {
		w = max(w, lipgloss.Width(l))
	}
	return w
}

// longest is the widest word of the cell.
func (c mdCell) longest() int {
	w := 0
	for _, l := range c.lines {
		for _, word := range strings.Fields(l) {
			w = max(w, lipgloss.Width(word))
		}
	}
	return w
}

// render wraps every line of the cell to width, the continuation of a bullet
// under its text.
func (c mdCell) render(width int) string {
	if c.url != "" {
		return accentStyle.Render(shortURL(c.url, width))
	}
	var out []string
	for _, l := range c.lines {
		if rest, ok := strings.CutPrefix(l, faintStyle.Render("· ")); ok {
			out = append(out, hang(faintStyle.Render("· "), rest, width))
			continue
		}
		out = append(out, lipgloss.NewStyle().Width(width).Render(l))
	}
	return strings.Join(out, "\n")
}

// shortURL is u when it fits in width columns, else its last path segments
// (the issue key of a task), else its end.
func shortURL(u string, width int) string {
	if len(u) <= width {
		return u
	}
	path := u
	if i := strings.Index(u, "://"); i >= 0 {
		path = u[i+3:]
	}
	segs := strings.Split(path, "/")
	for n := min(len(segs)-1, 2); n >= 1; n-- {
		if s := "…/" + strings.Join(segs[len(segs)-n:], "/"); utf8.RuneCountInString(s) <= width {
			return s
		}
	}
	runes := []rune(u)
	return "…" + string(runes[max(len(runes)-width+1, 0):])
}

func (r *mdRenderer) table(lines []string, i int) int {
	header := splitRow(lines[i])
	var rows [][]string
	for i += 2; i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|"); i++ {
		rows = append(rows, splitRow(lines[i]))
	}
	n := len(header)
	grid, natural, lower := [][]mdCell{}, make([]int, n), make([]int, n)
	emptyHeader := true
	for _, h := range header {
		emptyHeader = emptyHeader && h == ""
	}
	all := rows
	if !emptyHeader {
		all = append([][]string{header}, rows...)
	}
	for k, row := range all {
		cells := make([]mdCell, n)
		for c := range n {
			text := ""
			if c < len(row) {
				text = row[c]
			}
			cells[c] = parseCell(text, !emptyHeader && k == 0)
			natural[c] = max(natural[c], cells[c].width())
			lower[c] = max(lower[c], min(cells[c].longest(), maxWord))
		}
		grid = append(grid, cells)
	}
	widths := columnWidths(r.weights, natural, lower, r.w()-columnGap*(n-1))
	r.weights = nil

	var out []string
	for k, cells := range grid {
		parts := make([]string, 0, 2*n)
		for c, cell := range cells {
			if c > 0 {
				parts = append(parts, strings.Repeat(" ", columnGap))
			}
			parts = append(parts, lipgloss.NewStyle().Width(widths[c]).Render(cell.render(widths[c])))
		}
		out = append(out, lipgloss.JoinHorizontal(lipgloss.Top, parts...))
		if k == 0 && !emptyHeader {
			total := columnGap * (n - 1)
			for _, w := range widths {
				total += w
			}
			out = append(out, faintStyle.Render(strings.Repeat("─", total)))
		}
	}
	r.emit(strings.Join(out, "\n"))
	return i
}

// columnWidths gives the table its width: in the proportions of its draco
// comment, or each column as wide as its content, shrunk to fit. A column keeps
// at least its longest word (lower) while there is room, so labels are not
// cut in the middle.
func columnWidths(weights, natural, lower []int, avail int) []int {
	out := make([]int, len(natural))
	sum := sumOf(natural)
	switch {
	case len(weights) == len(natural):
		total := 0
		for _, w := range weights {
			total += max(w, 1)
		}
		for i, w := range weights {
			out[i] = avail * max(w, 1) / total
		}
	case sum <= avail:
		return slices.Clone(natural)
	default:
		for i, w := range natural {
			out[i] = avail * max(w, 1) / sum
		}
	}
	for i := range out {
		out[i] = max(out[i], lower[i], minColumn)
	}
	shrink(out, lower, avail)
	shrink(out, nil, avail)
	if extra := avail - sumOf(out); extra > 0 {
		out[slices.Index(out, slices.Max(out))] += extra
	}
	return out
}

// shrink takes width from the columns, the one with the most to spare first,
// until they fit in avail or none can go below its floor (minColumn without one).
func shrink(out, floor []int, avail int) {
	for sumOf(out) > avail {
		best, spare := -1, 0
		for i, w := range out {
			lowest := minColumn
			if floor != nil {
				lowest = max(floor[i], minColumn)
			}
			if w-lowest > spare {
				best, spare = i, w-lowest
			}
		}
		if best < 0 {
			return
		}
		out[best]--
	}
}

func sumOf(s []int) int {
	total := 0
	for _, v := range s {
		total += v
	}
	return total
}

// inline styles the bold, italic, code and links of a line and removes the
// backslashes of escaped characters.
func inline(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		rest := s[i:]
		switch {
		case s[i] == '\\' && i+1 < len(s) && strings.IndexByte(mdPunctuation, s[i+1]) >= 0:
			b.WriteByte(s[i+1])
			i += 2
		case strings.HasPrefix(rest, "**"):
			if j := strings.Index(rest[2:], "**"); j > 0 {
				b.WriteString(boldStyle.Render(inline(rest[2 : 2+j])))
				i += j + 4
				continue
			}
			b.WriteString("**")
			i += 2
		case s[i] == '_' && openUnderscore(s, i):
			if j := closeUnderscore(rest[1:]); j > 0 {
				b.WriteString(lipgloss.NewStyle().Italic(true).Render(inline(rest[1 : 1+j])))
				i += j + 2
				continue
			}
			b.WriteByte('_')
			i++
		case s[i] == '`':
			if j := strings.IndexByte(rest[1:], '`'); j > 0 {
				b.WriteString(accentStyle.Render(rest[1 : 1+j]))
				i += j + 2
				continue
			}
			b.WriteByte('`')
			i++
		case s[i] == '[':
			if text, n := link(rest); n > 0 {
				b.WriteString(accentStyle.Render(inline(text)))
				i += n
				continue
			}
			b.WriteByte('[')
			i++
		case s[i] == '<':
			if j := strings.IndexByte(rest, '>'); j > 1 && strings.Contains(rest[1:j], "://") && !strings.ContainsAny(rest[1:j], " \t") {
				b.WriteString(accentStyle.Render(rest[1:j]))
				i += j + 1
				continue
			}
			b.WriteByte('<')
			i++
		default:
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// link reads "[text](url)" at the start of s: the text and how many bytes the
// link takes; 0 when s does not start with one.
func link(s string) (text string, n int) {
	j := strings.Index(s, "](")
	if j < 1 {
		return "", 0
	}
	k := strings.IndexByte(s[j+2:], ')')
	if k < 0 {
		return "", 0
	}
	return s[1:j], j + 2 + k + 1
}

// openUnderscore is an _ that can start italics: not inside a word.
func openUnderscore(s string, i int) bool {
	if i > 0 {
		prev, _ := utf8.DecodeLastRuneInString(s[:i])
		if unicode.IsLetter(prev) || unicode.IsDigit(prev) {
			return false
		}
	}
	next, _ := utf8.DecodeRuneInString(s[i+1:])
	return next != utf8.RuneError && !unicode.IsSpace(next)
}

// closeUnderscore is where the italics that start before s end, or -1.
func closeUnderscore(s string) int {
	for j := 0; j < len(s); j++ {
		if s[j] == '\\' {
			j++
			continue
		}
		if s[j] != '_' {
			continue
		}
		next, _ := utf8.DecodeRuneInString(s[j+1:])
		if j > 0 && !unicode.IsSpace(rune(s[j-1])) && (j+1 == len(s) || !(unicode.IsLetter(next) || unicode.IsDigit(next))) {
			return j
		}
	}
	return -1
}
