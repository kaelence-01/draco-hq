package qa

import (
	"testing"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

func ok() provider.QAIssue { return provider.QAIssue{PRMerged: true, HasDeployDoc: true} }

func TestStatusOf(t *testing.T) {
	cases := []struct {
		name string
		edit func(*provider.QAIssue)
		want Status
	}{
		{"las cuatro comprobaciones pasan", func(*provider.QAIssue) {}, Complete},
		{"errores de código", func(i *provider.QAIssue) { i.HasCodingErrors = true }, Failing},
		{"devoluciones a desarrollo", func(i *provider.QAIssue) { i.HasDevReturns = true }, Failing},
		{"pull request sin aprobar", func(i *provider.QAIssue) { i.PRMerged = false }, Failing},
		{"sin documento de despliegue", func(i *provider.QAIssue) { i.HasDeployDoc = false }, Failing},
		{"la consulta del documento falló: sin verificar, no con fallas", func(i *provider.QAIssue) { i.HasDeployDoc, i.DeployDocUnknown = false, true }, Unverified},
		{"una falla real no se esconde detrás de una consulta fallida", func(i *provider.QAIssue) {
			i.HasDeployDoc, i.DeployDocUnknown, i.HasCodingErrors = false, true, true
		}, Failing},
		{"consulta fallida y el pull request sin aprobar: con fallas", func(i *provider.QAIssue) {
			i.HasDeployDoc, i.DeployDocUnknown, i.PRMerged = false, true, false
		}, Failing},
	}
	for _, c := range cases {
		task := ok()
		c.edit(&task)
		if got := StatusOf(task); got != c.want {
			t.Errorf("%s: se esperaba %d y se obtuvo %d", c.name, c.want, got)
		}
	}
}

func TestTally(t *testing.T) {
	failing := provider.QAIssue{HasCodingErrors: true}
	unknown := provider.QAIssue{PRMerged: true, DeployDocUnknown: true}
	tasks := []provider.QAIssue{ok(), failing, unknown, ok(), {}}

	cases := []struct {
		name      string
		evaluated int
		want      Counts
	}{
		{"ninguna evaluada", 0, Counts{Total: 5}},
		{"las que se evaluaron", 3, Counts{Total: 5, Complete: 1, Failing: 1, Unverified: 1}},
		{"todas", 5, Counts{Total: 5, Complete: 2, Failing: 2, Unverified: 1}},
		{"de más no cuenta", 9, Counts{Total: 5, Complete: 2, Failing: 2, Unverified: 1}},
		{"de menos tampoco", -1, Counts{Total: 5}},
	}
	for _, c := range cases {
		got := Tally(tasks, c.evaluated)
		if got != c.want {
			t.Errorf("%s: se esperaba %+v y se obtuvo %+v", c.name, c.want, got)
		}
		if got.Evaluated()+got.Pending() != got.Total {
			t.Errorf("%s: evaluadas y por evaluar suman el total: %+v", c.name, got)
		}
	}
	if got := Tally(nil, 0); got != (Counts{}) {
		t.Errorf("sin tareas, todo en cero: %+v", got)
	}
}

func TestPendingTasksAreNotFailures(t *testing.T) {
	pending := provider.QAIssue{Key: "OPS-9"}
	if got := Tally([]provider.QAIssue{ok(), pending}, 1); got.Failing != 0 || got.Pending() != 1 {
		t.Errorf("una tarea sin evaluar no tiene documento comprobado y no es una falla: %+v", got)
	}
}
