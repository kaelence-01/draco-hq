package atlassian

import (
	"strings"
	"testing"
)

func TestQuoteLiteralEscapesQuotesAndBackslashes(t *testing.T) {
	cases := map[string]string{
		`FAC`:            `"FAC"`,
		`Pagos "nuevo"`:  `"Pagos \"nuevo\""`,
		`ruta\interna`:   `"ruta\\interna"`,
		`" OR project=X`: `"\" OR project=X"`,
		``:               `""`,
	}
	for in, want := range cases {
		if got := quoteLiteral(in); got != want {
			t.Errorf("quoteLiteral(%q) = %s, esperaba %s", in, got, want)
		}
	}
}

func TestQuoteLiteralFoldsControlCharacters(t *testing.T) {
	// Neither JQL nor CQL accepts a raw newline inside a literal.
	if got := quoteLiteral("FAC\nOR project=X"); got != `"FAC OR project=X"` {
		t.Errorf("no plegó el salto de línea: %s", got)
	}
}

func TestAdfExtractTextWalksNestedNodes(t *testing.T) {
	node := map[string]any{
		"type": "doc",
		"content": []any{
			map[string]any{
				"type": "paragraph",
				"content": []any{
					map[string]any{"type": "text", "text": "Novedad::"},
					map[string]any{"type": "text", "text": "falta validar el campo"},
				},
			},
		},
	}
	if got := adfExtractText(node); got != "Novedad::falta validar el campo" {
		t.Errorf("adfExtractText = %q", got)
	}
}

var testSchema = QASchema{
	Project:       "OPS",
	Statuses:      []string{"10001", "En Revisión"},
	TestingStatus: "Testing",
}

func TestQAStatusListQuotesNamesButNotIDs(t *testing.T) {
	if got, want := testSchema.statusList(), `10001, "En Revisión"`; got != want {
		t.Errorf("statusList() = %s, esperaba %s", got, want)
	}
}

func TestQAJQLQuotesEveryInterpolatedValue(t *testing.T) {
	cases := map[string]struct{ got, want string }{
		"sprint review": {
			sprintReviewJQL(testSchema, "Pagos_Sprint 7", "Pagos"),
			`project = "OPS" AND sprint = "Pagos_Sprint 7" AND status in (10001, "En Revisión") AND component = "Pagos" ORDER BY key ASC`,
		},
		"sprint assignee": {
			sprintAssigneeJQL(testSchema, "Pagos_Sprint 7", "currentUser()"),
			`project = "OPS" AND sprint = "Pagos_Sprint 7" AND assignee = currentUser() ORDER BY key ASC`,
		},
		"kanban review": {
			kanbanReviewJQL(testSchema, "2026-10-05"),
			`project = "OPS" AND status in (10001, "En Revisión") AND status CHANGED TO "Testing" AFTER "2026-10-05" ORDER BY key ASC`,
		},
		"kanban assignee": {
			kanbanAssigneeJQL(testSchema, "2026-10-05", `"abc123"`),
			`project = "OPS" AND assignee = "abc123" AND updated >= "2026-10-05" ORDER BY key ASC`,
		},
	}
	for name, c := range cases {
		if c.got != c.want {
			t.Errorf("%s:\n  obtuve    %s\n  esperaba  %s", name, c.got, c.want)
		}
	}
}

func TestQAJQLCannotBeBrokenByAModuleName(t *testing.T) {
	jql := sprintReviewJQL(testSchema, "S1", `Pagos" OR project = X`)
	if want := `component = "Pagos\" OR project = X"`; !strings.Contains(jql, want) {
		t.Errorf("el módulo no quedó escapado: %s", jql)
	}
}
