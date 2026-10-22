package atlassian

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// MarkdownToADF converts the Markdown draco publishes (GFM: tables, task
// lists, strikethrough) to an ADF document. Conventions that recover what
// Markdown cannot express (docs/arquitectura/proveedores.md, §5):
//   - a link whose text is its URL, or an autolink, becomes an inline card;
//   - "✔" and "✖" become Confluence emoji;
//   - a table header row with every cell empty is dropped;
//   - a table cell whose whole content is bold becomes a header cell;
//   - "<br>" inside a table cell is a line break, and a cell whose every line
//     starts with "- " is a bullet list;
//   - "<!-- draco:tabla anchos=63,697 [ancho=1800] [layout=default] -->" sets
//     the column widths of the next table;
//   - "<!-- draco:recuadro -->" puts the next block in a one-cell table.
//
// Any other raw HTML is kept as literal text, never interpreted.
func MarkdownToADF(md string) map[string]any {
	source := []byte(md)
	parser := goldmark.New(goldmark.WithExtensions(extension.Table, extension.TaskList, extension.Strikethrough)).Parser()
	doc := parser.Parse(text.NewReader(source))
	c := &mdConverter{source: source}
	return map[string]any{"type": "doc", "version": 1, "content": c.blocks(doc)}
}

type mdConverter struct {
	source    []byte
	taskLists int
}

func (c *mdConverter) blocks(parent gast.Node) []any {
	out := []any{}
	var pending func(map[string]any) map[string]any
	for n := parent.FirstChild(); n != nil; n = n.NextSibling() {
		if d, ok := c.directive(n); ok {
			pending = d
			continue
		}
		b := c.block(n)
		if b == nil {
			continue
		}
		if pending != nil {
			b, pending = pending(b), nil
		}
		out = append(out, b)
	}
	return out
}

// tableWidth is the table width gtt published, which the column widths scale to.
const tableWidth = 1800

var directiveRe = regexp.MustCompile(`^<!--\s*draco:(tabla|recuadro)((?:\s+[a-z]+=\S+)*)\s*-->$`)

// directive reads a draco comment and returns what it does to the next block.
func (c *mdConverter) directive(n gast.Node) (func(map[string]any) map[string]any, bool) {
	html, ok := n.(*gast.HTMLBlock)
	if !ok {
		return nil, false
	}
	m := directiveRe.FindStringSubmatch(strings.TrimSpace(string(html.Lines().Value(c.source))))
	if m == nil {
		return nil, false
	}
	params := map[string]string{"ancho": strconv.Itoa(tableWidth), "layout": "default"}
	for _, kv := range strings.Fields(m[2]) {
		k, v, _ := strings.Cut(kv, "=")
		params[k] = v
	}
	width, err := strconv.Atoi(params["ancho"])
	if err != nil {
		return nil, false
	}
	attrs := map[string]any{"layout": params["layout"], "width": width}
	if m[1] == "recuadro" {
		return func(b map[string]any) map[string]any {
			cell := map[string]any{"type": "tableCell", "attrs": map[string]any{}, "content": []any{b}}
			row := map[string]any{"type": "tableRow", "content": []any{cell}}
			return map[string]any{"type": "table", "attrs": attrs, "content": []any{row}}
		}, true
	}
	var widths []int
	for _, s := range strings.Split(params["anchos"], ",") {
		w, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return nil, false
		}
		widths = append(widths, w)
	}
	return func(b map[string]any) map[string]any {
		if b["type"] != "table" {
			return b
		}
		b["attrs"] = attrs
		for _, r := range b["content"].([]any) {
			for i, cell := range r.(map[string]any)["content"].([]any) {
				if i < len(widths) {
					cell.(map[string]any)["attrs"] = map[string]any{"colwidth": []int{widths[i]}}
				}
			}
		}
		return b
	}, true
}

func (c *mdConverter) block(n gast.Node) map[string]any {
	switch n := n.(type) {
	case *gast.Heading:
		return withContent(map[string]any{"type": "heading", "attrs": map[string]any{"level": n.Level}}, c.inlines(n, nil))
	case *gast.Paragraph, *gast.TextBlock:
		return withContent(map[string]any{"type": "paragraph"}, c.inlines(n, nil))
	case *gast.List:
		if isTaskList(n) {
			return c.taskList(n)
		}
		list := map[string]any{"type": "bulletList"}
		if n.IsOrdered() {
			list = map[string]any{"type": "orderedList", "attrs": map[string]any{"order": n.Start}}
		}
		items := []any{}
		for li := n.FirstChild(); li != nil; li = li.NextSibling() {
			content := c.blocks(li)
			if len(content) == 0 {
				content = []any{map[string]any{"type": "paragraph"}}
			}
			items = append(items, map[string]any{"type": "listItem", "content": content})
		}
		list["content"] = items
		return list
	case *east.Table:
		return c.table(n)
	case *gast.FencedCodeBlock:
		node := map[string]any{"type": "codeBlock"}
		if lang := string(n.Language(c.source)); lang != "" {
			node["attrs"] = map[string]any{"language": lang}
		}
		return withContent(node, textOnly(string(n.Lines().Value(c.source))))
	case *gast.CodeBlock:
		return withContent(map[string]any{"type": "codeBlock"}, textOnly(string(n.Lines().Value(c.source))))
	case *gast.Blockquote:
		return map[string]any{"type": "blockquote", "content": c.blocks(n)}
	case *gast.ThematicBreak:
		return map[string]any{"type": "rule"}
	case *gast.HTMLBlock:
		return withContent(map[string]any{"type": "paragraph"}, textOnly(strings.TrimRight(string(n.Lines().Value(c.source)), "\n")))
	}
	return nil
}

func isTaskList(l *gast.List) bool {
	for li := l.FirstChild(); li != nil; li = li.NextSibling() {
		if checkBox(li) == nil {
			return false
		}
	}
	return l.FirstChild() != nil
}

func checkBox(li gast.Node) *east.TaskCheckBox {
	if tb := li.FirstChild(); tb != nil {
		if cb, ok := tb.FirstChild().(*east.TaskCheckBox); ok {
			return cb
		}
	}
	return nil
}

func (c *mdConverter) taskList(l *gast.List) map[string]any {
	c.taskLists++
	items := []any{}
	i := 0
	for li := l.FirstChild(); li != nil; li = li.NextSibling() {
		i++
		state := "TODO"
		if checkBox(li).IsChecked {
			state = "DONE"
		}
		item := map[string]any{
			"type":  "taskItem",
			"attrs": map[string]any{"state": state, "localId": fmt.Sprintf("task-%d-%d", c.taskLists, i)},
		}
		items = append(items, withContent(item, c.inlines(li.FirstChild(), nil)))
	}
	return map[string]any{
		"type":    "taskList",
		"attrs":   map[string]any{"localId": fmt.Sprintf("tasklist-%d", c.taskLists)},
		"content": items,
	}
}

func (c *mdConverter) table(t *east.Table) map[string]any {
	rows := []any{}
	for r := t.FirstChild(); r != nil; r = r.NextSibling() {
		_, header := r.(*east.TableHeader)
		if header && c.emptyRow(r) {
			continue
		}
		cells := []any{}
		for cell := r.FirstChild(); cell != nil; cell = cell.NextSibling() {
			kind := "tableCell"
			strong := wholeStrong(cell)
			if header || strong != nil {
				kind = "tableHeader"
			}
			src := cell
			if strong != nil {
				src = strong
			}
			inline := c.inlines(src, nil)
			content := []any{withContent(map[string]any{"type": "paragraph"}, inline)}
			if list, ok := bulletLines(inline); ok {
				content = []any{list}
			}
			cells = append(cells, map[string]any{"type": kind, "content": content})
		}
		rows = append(rows, map[string]any{"type": "tableRow", "content": cells})
	}
	return map[string]any{
		"type":    "table",
		"attrs":   map[string]any{"layout": "full-width"},
		"content": rows,
	}
}

func (c *mdConverter) emptyRow(r gast.Node) bool {
	for cell := r.FirstChild(); cell != nil; cell = cell.NextSibling() {
		if len(c.inlines(cell, nil)) > 0 {
			return false
		}
	}
	return true
}

// bulletLines turns cell content whose every line (split at hardBreak) starts
// with "- " into a bullet list.
func bulletLines(inline []any) (map[string]any, bool) {
	if len(inline) == 0 {
		return nil, false
	}
	var items, line []any
	flush := func() bool {
		first, ok := line[0].(map[string]any)
		text, _ := first["text"].(string)
		if !ok || first["type"] != "text" || first["marks"] != nil || !strings.HasPrefix(text, "- ") {
			return false
		}
		rest := line[1:]
		if t := strings.TrimPrefix(text, "- "); t != "" {
			rest = append([]any{textNode(t, nil)}, rest...)
		}
		items = append(items, map[string]any{"type": "listItem", "content": []any{withContent(map[string]any{"type": "paragraph"}, rest)}})
		line = nil
		return true
	}
	for _, n := range inline {
		if m, ok := n.(map[string]any); ok && m["type"] == "hardBreak" {
			if len(line) == 0 || !flush() {
				return nil, false
			}
			continue
		}
		line = append(line, n)
	}
	if len(line) == 0 || !flush() {
		return nil, false
	}
	return map[string]any{"type": "bulletList", "content": items}, true
}

// wholeStrong returns the bold node when it is the cell's only content.
func wholeStrong(cell gast.Node) gast.Node {
	if cell.ChildCount() != 1 {
		return nil
	}
	if e, ok := cell.FirstChild().(*gast.Emphasis); ok && e.Level == 2 {
		return e
	}
	return nil
}

var brTag = regexp.MustCompile(`(?i)^<br\s*/?>$`)

// inlines converts the inline children of n, carrying the marks of its
// ancestors.
func (c *mdConverter) inlines(n gast.Node, marks []any) []any {
	out := []any{}
	for ch := n.FirstChild(); ch != nil; ch = ch.NextSibling() {
		switch ch := ch.(type) {
		case *gast.Text:
			out = append(out, textNodes(unescape(ch.Value(c.source)), marks)...)
			if ch.HardLineBreak() {
				out = append(out, map[string]any{"type": "hardBreak"})
			} else if ch.SoftLineBreak() {
				out = append(out, textNodes(" ", marks)...)
			}
		case *gast.String:
			out = append(out, textNodes(string(ch.Value), marks)...)
		case *gast.Emphasis:
			mark := "em"
			if ch.Level == 2 {
				mark = "strong"
			}
			out = append(out, c.inlines(ch, withMark(marks, map[string]any{"type": mark}))...)
		case *east.Strikethrough:
			out = append(out, c.inlines(ch, withMark(marks, map[string]any{"type": "strike"}))...)
		case *gast.CodeSpan:
			var sb strings.Builder
			for t := ch.FirstChild(); t != nil; t = t.NextSibling() {
				if tt, ok := t.(*gast.Text); ok {
					sb.Write(tt.Value(c.source))
				}
			}
			out = append(out, textNodes(sb.String(), withMark(marks, map[string]any{"type": "code"}))...)
		case *gast.AutoLink:
			out = append(out, inlineCardNode(string(ch.URL(c.source))))
		case *gast.Link:
			href := unescape(ch.Destination)
			label := c.plain(ch)
			if label == href {
				out = append(out, inlineCardNode(href))
				continue
			}
			out = append(out, c.inlines(ch, withMark(marks, map[string]any{"type": "link", "attrs": map[string]any{"href": href}}))...)
		case *gast.RawHTML:
			raw := string(ch.Segments.Value(c.source))
			if brTag.MatchString(strings.TrimSpace(raw)) {
				out = append(out, map[string]any{"type": "hardBreak"})
			} else {
				out = append(out, textNodes(raw, marks)...)
			}
		case *gast.Image:
			out = append(out, textNodes(c.plain(ch), marks)...)
		case *east.TaskCheckBox:
		default:
			out = append(out, c.inlines(ch, marks)...)
		}
	}
	return out
}

// plain returns the text of n without marks.
func (c *mdConverter) plain(n gast.Node) string {
	var sb strings.Builder
	for _, node := range c.inlines(n, nil) {
		if m, ok := node.(map[string]any); ok {
			if t, ok := m["text"].(string); ok {
				sb.WriteString(t)
			}
		}
	}
	return sb.String()
}

func unescape(b []byte) string {
	return string(util.ResolveEntityNames(util.ResolveNumericReferences(util.UnescapePunctuations(b))))
}

var emoji = map[rune]map[string]any{
	'✔': {"shortName": ":check_mark:", "id": "atlassian-check_mark", "text": "✔"},
	'✖': {"shortName": ":cross_mark:", "id": "atlassian-cross_mark", "text": "✖"},
}

// textNodes splits s into text nodes and the emoji the conventions recognize.
func textNodes(s string, marks []any) []any {
	out := []any{}
	var buf strings.Builder
	flush := func() {
		if buf.Len() > 0 {
			out = append(out, textNode(buf.String(), marks))
			buf.Reset()
		}
	}
	for _, r := range s {
		if attrs, ok := emoji[r]; ok {
			flush()
			out = append(out, map[string]any{"type": "emoji", "attrs": attrs})
			continue
		}
		buf.WriteRune(r)
	}
	flush()
	return out
}

func textNode(s string, marks []any) map[string]any {
	node := map[string]any{"type": "text", "text": s}
	if len(marks) > 0 {
		node["marks"] = marks
	}
	return node
}

func textOnly(s string) []any {
	if s == "" {
		return nil
	}
	return []any{map[string]any{"type": "text", "text": s}}
}

func inlineCardNode(url string) map[string]any {
	return map[string]any{"type": "inlineCard", "attrs": map[string]any{"url": url}}
}

func withMark(marks []any, m map[string]any) []any {
	out := make([]any, 0, len(marks)+1)
	return append(append(out, marks...), m)
}

// withContent sets content only when there is some: ADF rejects empty text
// nodes, and an empty paragraph is valid without it.
func withContent(node map[string]any, content []any) map[string]any {
	if len(content) > 0 {
		node["content"] = content
	}
	return node
}
