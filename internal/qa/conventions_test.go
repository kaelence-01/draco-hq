package qa

import (
	"testing"
	"time"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

func TestParseDevTaskKey(t *testing.T) {
	cases := map[string]string{
		"Revisión de Tarea - OPS-1257":           "OPS-1257",
		"Revisión de Tarea — OPS-1257":           "OPS-1257",
		"Revisión de Tarea - OPS-1257 - ajuste":  "OPS-1257",
		"revisión de tarea - ops-1257":           "OPS-1257",
		"Revisión de Tarea - PAG-42":             "PAG-42",
		"QA OPS-9 validación":                    "OPS-9",
		"Revisión de Tarea sin clave":            "",
		"":                                       "",
		"Revisión de Tarea - OPS-1257 vs OPS-99": "OPS-1257",
	}
	for summary, want := range cases {
		if got := parseDevTaskKey(summary); got != want {
			t.Errorf("parseDevTaskKey(%q) = %q, esperaba %q", summary, got, want)
		}
	}
}

func TestBuildReviewMapIndexesByDevTaskKey(t *testing.T) {
	m := buildReviewMap([]provider.QAIssue{
		{Key: "OPS-2160", Summary: "Revisión de Tarea - OPS-1257"},
		{Key: "OPS-2161", Summary: "Revisión de Tarea - OPS-1300"},
		{Key: "OPS-2162", Summary: "Tarea suelta sin clave de dev"},
	})
	if got := m["OPS-1257"].Key; got != "OPS-2160" {
		t.Errorf("OPS-1257 → %q, esperaba OPS-2160", got)
	}
	if got := m["OPS-1300"].Key; got != "OPS-2161" {
		t.Errorf("OPS-1300 → %q, esperaba OPS-2161", got)
	}
	if len(m) != 2 {
		t.Errorf("esperaba 2 entradas, obtuve %d: %v", len(m), m)
	}
}

func TestDevTasksOnlyDropsReviewTasks(t *testing.T) {
	got := devTasksOnly([]provider.QAIssue{
		{Key: "OPS-1", Summary: "Agregar filtro"},
		{Key: "OPS-2", Summary: "Revisión de Tarea - OPS-1"},
		{Key: "OPS-3", Summary: "revision de tarea - OPS-9"},
		{Key: "OPS-4", Summary: "QA regresión"},
	}, config.DefaultReviewPrefixes)
	if len(got) != 1 || got[0].Key != "OPS-1" {
		t.Errorf("devTasksOnly = %+v, esperaba solo OPS-1", got)
	}
}

func TestKanbanWindow(t *testing.T) {
	since, period := kanbanWindow(time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC))
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
			if wd := businessDaysAgo(n, from).Weekday(); wd == time.Saturday || wd == time.Sunday {
				t.Errorf("businessDaysAgo(%d, %s) cayó en %s", n, from.Format("2006-01-02"), wd)
			}
		}
	}
}
