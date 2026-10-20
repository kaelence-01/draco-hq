package atlassian

import "strings"

// Hand-edited sections are carried over in ADF, never through Markdown, so
// task states and formatting survive an update.

// preservedSection returns the content of the named section of a published
// page, and whether the page has that section.
func preservedSection(adf map[string]any, name string) ([]any, bool) {
	if nodes, ok := extractSection(adf, name); ok {
		return nodes, true
	}
	if normalizeHeading(name) == "observación" {
		return legacyObservation(adf)
	}
	return nil, false
}

// legacyObservation reads the QA observation of a report published before it
// became a section: the second cell of the data row of the summary table.
func legacyObservation(adf map[string]any) ([]any, bool) {
	nodes, ok := extractSection(adf, "Resumen")
	if !ok {
		return nil, false
	}
	for _, raw := range nodes {
		table, ok := raw.(map[string]any)
		if !ok || table["type"] != "table" {
			continue
		}
		rows, _ := table["content"].([]any)
		if len(rows) < 2 {
			return nil, false
		}
		row, _ := rows[1].(map[string]any)
		cells, _ := row["content"].([]any)
		if len(cells) < 2 {
			return nil, false
		}
		cell, _ := cells[1].(map[string]any)
		content, _ := cell["content"].([]any)
		return copyNodes(content), true
	}
	return nil, false
}

// extractSection returns the top-level nodes under the heading, up to the next
// heading of the same or higher rank (a deeper one belongs to the section), and
// whether the heading exists.
func extractSection(adf map[string]any, heading string) ([]any, bool) {
	content, _ := adf["content"].([]any)
	start, end := sectionBounds(content, heading)
	if start < 0 {
		return nil, false
	}
	return copyNodes(content[start:end]), true
}

// replaceSection puts nodes under the heading in place of what is there; it
// reports false when adf has no such heading.
func replaceSection(adf map[string]any, heading string, nodes []any) bool {
	content, _ := adf["content"].([]any)
	start, end := sectionBounds(content, heading)
	if start < 0 {
		return false
	}
	out := make([]any, 0, len(content)-(end-start)+len(nodes))
	out = append(out, content[:start]...)
	out = append(out, nodes...)
	out = append(out, content[end:]...)
	adf["content"] = out
	return true
}

// sectionBounds returns the range of nodes after the heading, or -1 when the
// heading is absent.
func sectionBounds(content []any, heading string) (start, end int) {
	want := normalizeHeading(heading)
	start, level := -1, 0
	for i, raw := range content {
		node, ok := raw.(map[string]any)
		if !ok || node["type"] != "heading" {
			continue
		}
		if start < 0 {
			if normalizeHeading(nodeText(node)) == want {
				start, level = i+1, headingLevel(node)
			}
			continue
		}
		if headingLevel(node) <= level {
			return start, i
		}
	}
	if start < 0 {
		return -1, -1
	}
	return start, len(content)
}

func copyNodes(nodes []any) []any {
	if len(nodes) == 0 {
		return nil
	}
	out := make([]any, len(nodes))
	copy(out, nodes)
	return out
}

// normalizeHeading ignores case, surrounding space and a trailing colon, which
// draco itself emits ("A considerar:").
func normalizeHeading(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.TrimSpace(strings.TrimSuffix(s, ":"))
}

func nodeText(node map[string]any) string {
	if t, ok := node["text"].(string); ok {
		return t
	}
	var sb strings.Builder
	if content, ok := node["content"].([]any); ok {
		for _, raw := range content {
			if child, ok := raw.(map[string]any); ok {
				sb.WriteString(nodeText(child))
			}
		}
	}
	return sb.String()
}

// headingLevel reads attrs.level as float64 (after JSON) or int (built in memory).
func headingLevel(node map[string]any) int {
	attrs, ok := node["attrs"].(map[string]any)
	if !ok {
		return 1
	}
	switch v := attrs["level"].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 1
}
