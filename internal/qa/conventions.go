package qa

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

// devTasksOnly drops the QA review tasks ("Revisión de Tarea - …", "QA …").
func devTasksOnly(all []provider.QAIssue) []provider.QAIssue {
	dev := make([]provider.QAIssue, 0, len(all))
	for _, t := range all {
		lower := strings.ToLower(t.Summary)
		if !strings.HasPrefix(lower, "revisión de tarea") &&
			!strings.HasPrefix(lower, "revision de tarea") &&
			!strings.HasPrefix(lower, "qa") {
			dev = append(dev, t)
		}
	}
	return dev
}

// buildReviewMap maps each dev task key to its QA review task
// ("Revisión de Tarea - OPS-7").
func buildReviewMap(qaTasks []provider.QAIssue) map[string]provider.QAIssue {
	m := make(map[string]provider.QAIssue, len(qaTasks))
	for _, t := range qaTasks {
		if key := parseDevTaskKey(t.Summary); key != "" {
			m[key] = t
		}
	}
	return m
}

var issueKeyRe = regexp.MustCompile(`[A-Z][A-Z0-9]*-\d+`)

// parseDevTaskKey returns the first issue key in a review task summary: the
// prefix never has one, and a trailing note may mention another key.
func parseDevTaskKey(summary string) string {
	return issueKeyRe.FindString(strings.ToUpper(summary))
}

// businessDaysAgo returns the date n weekdays before from.
func businessDaysAgo(n int, from time.Time) time.Time {
	d := from
	for count := 0; count < n; {
		d = d.AddDate(0, 0, -1)
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday {
			count++
		}
	}
	return d
}

// kanbanWindow returns the start date (10 business days before now) and a
// display label like "08/05 al 22/05/2026".
func kanbanWindow(now time.Time) (since time.Time, period string) {
	since = businessDaysAgo(10, now)
	period = fmt.Sprintf("%s al %s", since.Format("02/01"), now.Format("02/01/2006"))
	return
}
