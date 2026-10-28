package config

import "testing"

func completeQAJira() *QAJiraConfig {
	return &QAJiraConfig{
		Project:  "OPS",
		Statuses: []string{"10001"},
		Fields:   QAJiraFields{CodingErrors: "customfield_1", DevReturns: "customfield_2", DevStatus: "customfield_3"},
	}
}

func TestQAState(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want QAState
	}{
		{"nada configurado", Config{}, QAUnset},
		{"solo el email", Config{QAEmail: "qa@example.test"}, QAUnset},
		{"qa_jira vacío", Config{QAJira: &QAJiraConfig{}}, QAIncomplete},
		{"qa_jira a medias", Config{QAJira: &QAJiraConfig{Project: "OPS"}, QAEmail: "qa@example.test"}, QAIncomplete},
		{"sin email de la cuenta", Config{QAJira: completeQAJira()}, QANoAccount},
		{"lista", Config{QAJira: completeQAJira(), QAEmail: "qa@example.test"}, QAReady},
	}
	for _, c := range cases {
		if got := c.cfg.QAState(); got != c.want {
			t.Errorf("%s: se esperaba %d y se obtuvo %d", c.name, c.want, got)
		}
	}
}

func TestHasQAReport(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		want bool
	}{
		{"sin sección", Config{}, false},
		{"sección vacía", Config{QAReport: &QAReportConfig{}}, false},
		{"con un nombre", Config{QAReport: &QAReportConfig{PMO: "Ana"}}, true},
	}
	for _, c := range cases {
		if got := c.cfg.HasQAReport(); got != c.want {
			t.Errorf("%s: se esperaba %v y se obtuvo %v", c.name, c.want, got)
		}
	}
}
