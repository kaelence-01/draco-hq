package atlassian

import (
	"strings"
	"testing"
	"time"
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

func TestParseDevTaskKey(t *testing.T) {
	cases := map[string]string{
		"Revisión de Tarea - APP-1257":           "APP-1257",
		"Revisión de Tarea — APP-1257":           "APP-1257", // guión largo
		"Revisión de Tarea - APP-1257 - ajuste":  "APP-1257", // sufijo
		"revisión de tarea - app-1257":           "APP-1257", // minúsculas
		"Revisión de Tarea - ECU-42":             "ECU-42",
		"QA APP-9 validación":                    "APP-9",
		"Revisión de Tarea sin clave":            "",
		"":                                       "",
		"Revisión de Tarea - APP-1257 vs APP-99": "APP-1257", // gana el primero
	}
	for summary, want := range cases {
		if got := parseDevTaskKey(summary); got != want {
			t.Errorf("parseDevTaskKey(%q) = %q, esperaba %q", summary, got, want)
		}
	}
}

func TestBuildReviewMapIndexesByDevTaskKey(t *testing.T) {
	qaTasks := []QAIssue{
		{Key: "APP-2160", Summary: "Revisión de Tarea - APP-1257"},
		{Key: "APP-2161", Summary: "Revisión de Tarea - APP-1300"},
		{Key: "APP-2162", Summary: "Tarea suelta sin clave de dev"},
	}

	m := BuildReviewMap(qaTasks)

	if got := m["APP-1257"].Key; got != "APP-2160" {
		t.Errorf("APP-1257 → %q, esperaba APP-2160", got)
	}
	if got := m["APP-1300"].Key; got != "APP-2161" {
		t.Errorf("APP-1300 → %q, esperaba APP-2161", got)
	}
	// "Tarea suelta..." contains no key, so it must not create an entry.
	if len(m) != 2 {
		t.Errorf("esperaba 2 entradas, obtuve %d: %v", len(m), m)
	}
}

func TestBusinessDaysAgoSkipsWeekends(t *testing.T) {
	// Monday 2026-09-21 minus 10 business days is Monday 2026-09-07.
	from := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	got := businessDaysAgo(10, from)

	want := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("businessDaysAgo(10, %s) = %s, esperaba %s",
			from.Format("2006-01-02"), got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}

func TestKanbanWindow(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	since, period := KanbanWindow(now)
	if got := since.Format("2006-01-02"); got != "2026-09-07" {
		t.Errorf("since = %s, esperaba 2026-09-07", got)
	}
	if period != "07/09 al 21/09/2026" {
		t.Errorf("period = %q, esperaba %q", period, "07/09 al 21/09/2026")
	}
}

func TestBusinessDaysAgoNeverLandsOnWeekend(t *testing.T) {
	for day := 19; day <= 25; day++ {
		from := time.Date(2026, 9, day, 12, 0, 0, 0, time.UTC)
		for _, n := range []int{1, 5, 10, 22} {
			got := businessDaysAgo(n, from)
			if wd := got.Weekday(); wd == time.Saturday || wd == time.Sunday {
				t.Errorf("businessDaysAgo(%d, %s) cayó en %s",
					n, from.Format("2006-01-02"), wd)
			}
		}
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

func TestDevTasksOnlyDropsReviewTasks(t *testing.T) {
	got := devTasksOnly([]QAIssue{
		{Key: "OPS-1", Summary: "Agregar filtro"},
		{Key: "OPS-2", Summary: "Revisión de Tarea - OPS-1"},
		{Key: "OPS-3", Summary: "revision de tarea - OPS-9"},
		{Key: "OPS-4", Summary: "QA regresión"},
	})
	if len(got) != 1 || got[0].Key != "OPS-1" {
		t.Errorf("devTasksOnly = %+v, esperaba solo OPS-1", got)
	}
}
