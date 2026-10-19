package document

import "strings"

// ConsiderHeading opens the deploy document section edited by hand; it must
// survive a regeneration.
const ConsiderHeading = "A considerar"

// ExtractSection returns the top-level nodes under heading, up to the next
// heading of the same or higher rank, copied as-is (never merged) so task
// states survive. Nil when the heading is absent or empty.
func ExtractSection(adf map[string]any, heading string) []any {
	content, ok := adf["content"].([]any)
	if !ok {
		return nil
	}

	want := normalizeHeading(heading)
	start := -1
	level := 0

	for i, raw := range content {
		node, ok := raw.(map[string]any)
		if !ok || node["type"] != "heading" {
			continue
		}

		if start == -1 {
			if normalizeHeading(nodeText(node)) == want {
				start = i + 1
				level = headingLevel(node)
			}
			continue
		}

		if headingLevel(node) <= level {
			return copyNodes(content[start:i])
		}
	}

	if start == -1 || start >= len(content) {
		return nil
	}
	return copyNodes(content[start:])
}

// copyNodes is shallow: the nodes are never mutated.
func copyNodes(nodes []any) []any {
	if len(nodes) == 0 {
		return nil
	}
	out := make([]any, len(nodes))
	copy(out, nodes)
	return out
}

// normalizeHeading ignores case, surrounding space and a trailing colon.
func normalizeHeading(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimSuffix(s, ":")
	return strings.TrimSpace(s)
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

// headingLevel reads attrs.level: float64 after JSON, int when built in memory.
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
