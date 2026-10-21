package tui

import (
	"reflect"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/qa"
)

func (e *settingsEnv) toQA(t *testing.T) {
	t.Helper()
	send(e.m, keyDown)
	send(e.m, keyDown)
	send(e.m, keyEnter)
	if out := view(e.m); !strings.Contains(out, "Configuración › QA") || !strings.Contains(out, "Email de la cuenta de QA") {
		t.Fatalf("enter en QA abre su formulario:\n%s", out)
	}
}

func TestQAConfigFromScratchWorksForTheReport(t *testing.T) {
	e := newSettingsApp(t, savedConfig())
	if out := view(e.m); !strings.Contains(out, "⚠ sin configurar: solo hace falta para el consolidado QA") {
		t.Fatalf("sin qa_jira, la sección lo dice:\n%s", out)
	}
	e.toQA(t)
	e.fill("qa@example.test", "Ana", "Luis", "Sofía",
		"OPS", "10001, En QA", "", "",
		"customfield_10001", "customfield_10002", "customfield_10003",
		"", "")

	c := e.file.cfg
	if c.QAEmail != "qa@example.test" || c.QAReport == nil || c.QAReport.PMO != "Luis" || c.QAJira == nil {
		t.Fatalf("se guardan qa_email, qa_report y qa_jira: %q %+v %+v", c.QAEmail, c.QAReport, c.QAJira)
	}
	if !reflect.DeepEqual(c.QAJira.Statuses, []string{"10001", "En QA"}) || c.QAJira.ReviewPrefixes != nil || c.QAJira.TestingStatus != "" {
		t.Errorf("las listas se separan por comas y lo vacío queda vacío: %+v", c.QAJira)
	}
	target, err := qa.Resolve(c, qa.Request{Sprint: 12, Module: "Pagos"}, time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("qa.Resolve acepta la config creada desde la TUI: %v", err)
	}
	if target.SprintName != "Pagos_Sprint 12" || target.Report.LiderTecnico != "Ana" || len(target.Warnings) != 0 {
		t.Errorf("el consolidado usa lo guardado: %+v", target)
	}
	out := view(e.m)
	if len(e.m.stack) != 2 || !strings.Contains(out, "✓ QA guardado") || !strings.Contains(out, "✓ proyecto OPS; publica qa@example.test") {
		t.Errorf("vuelve a Configuración con la sección completa:\n%s", out)
	}
	if c.AtlassianToken != "token-del-archivo" {
		t.Errorf("guardar QA no toca el token del archivo: %q", c.AtlassianToken)
	}
}

func TestQAConfigShowsEveryProblemAndSavesNothing(t *testing.T) {
	e := newSettingsApp(t, savedConfig())
	e.toQA(t)
	e.fill("qa@example.test", "", "", "",
		"OPS", "", "", "Sprint",
		"campo", "", "",
		"", "")
	out := view(e.m)
	if e.file.saves != 0 || len(e.m.stack) != 3 {
		t.Fatalf("con problemas no se guarda y el formulario sigue abierto: %d escrituras\n%s", e.file.saves, out)
	}
	for _, want := range []string{"No se guardó", "qa_jira.statuses", "qa_jira.sprint_name: debe incluir {sprint}", "qa_jira.fields.coding_errors", "qa_jira.fields.dev_status"} {
		if !strings.Contains(out, want) {
			t.Errorf("falta %q: todos los problemas a la vez:\n%s", want, out)
		}
	}
	for range 4 {
		send(e.m, keyEnter)
	}
	if out := view(e.m); !strings.Contains(out, "OPS") || !strings.Contains(out, "Sprint") {
		t.Errorf("el formulario conserva lo escrito:\n%s", out)
	}
	send(e.m, keyEsc)
	if e.file.saves != 0 || len(e.m.stack) != 2 {
		t.Errorf("esc cierra sin guardar: %d escrituras", e.file.saves)
	}
}

func TestQAConfigWithoutJiraSavesOnlyTheEmail(t *testing.T) {
	cfg := savedConfig()
	cfg.QAReport = &config.QAReportConfig{QA: "Sofía"}
	e := newSettingsApp(t, cfg)
	e.toQA(t)
	e.fill("qa@example.test", "", "")
	for range len([]rune("Sofía")) {
		send(e.m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	}
	for range 10 {
		send(e.m, keyEnter)
	}
	c := e.file.cfg
	if c.QAEmail != "qa@example.test" || c.QAJira != nil || c.QAReport != nil {
		t.Errorf("sin datos de Jira ni nombres, solo se guarda qa_email: %q %+v %+v", c.QAEmail, c.QAJira, c.QAReport)
	}
}

func TestQAConfigStartsFromTheFile(t *testing.T) {
	cfg := savedConfig()
	cfg.QAEmail = "qa@example.test"
	cfg.QAJira = &config.QAJiraConfig{Project: "OPS", Statuses: []string{"10001", "En QA"}, ReviewPrefixes: []string{"Validación"},
		Fields: config.QAJiraFields{CodingErrors: "customfield_10001", DevReturns: "customfield_10002", DevStatus: "customfield_10003"}}
	e := newSettingsApp(t, cfg)
	e.toQA(t)
	for range 4 {
		send(e.m, keyEnter)
	}
	if out := view(e.m); !strings.Contains(out, "10001, En QA") {
		t.Fatalf("las listas se muestran separadas por comas:\n%s", out)
	}
	for range 9 {
		send(e.m, keyEnter)
	}
	if got := e.file.cfg.QAJira; got == nil || !reflect.DeepEqual(got, cfg.QAJira) {
		t.Errorf("guardar sin cambios conserva qa_jira: %+v", got)
	}
}
