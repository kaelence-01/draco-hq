package document

import (
	"testing"
	"time"

	"github.com/kaelence-01/draco-hq/internal/golden"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

type rendered struct {
	Title string         `json:"title"`
	ADF   map[string]any `json:"adf"`
}

func TestGoldenDeploy(t *testing.T) {
	cases := map[string]DeployDoc{
		"deploy-completo": {
			IssueKey:       "OPS-7",
			IssueSummary:   "Filtro de pagos: rango [fechas]",
			IssueURL:       "https://example.test/browse/OPS-7",
			VCSHost:        "https://bitbucket.org",
			VCSOrg:         "mi-org",
			BackendRepo:    "pagos-api",
			BackendCommit:  "a1b2c3d",
			BackendFiles:   map[string][]string{"app/Http/Pagos": {"FiltroController.php", "Rango.php"}, ".": {"composer.json"}},
			FrontendRepo:   "pagos-web",
			FrontendCommit: "e4f5a6b",
			FrontendFiles:  map[string][]string{"src/pagos": {"filtro.component.ts"}},
			Checklist:      []string{"Respaldar base de datos", "  ", "Desplegar backend"},
		},
		"deploy-github-checklist-default": {
			IssueKey:       "OPS-8",
			IssueSummary:   "Ajuste de facturación",
			IssueURL:       "https://example.test/browse/OPS-8",
			VCSHost:        "https://github.com",
			VCSOrg:         "mi-org",
			FrontendRepo:   "facturacion-web",
			FrontendCommit: "0f1e2d3",
			FrontendFiles:  map[string][]string{"src": {"main.ts"}},
		},
		"deploy-sin-vcs-preservado": {
			IssueKey:      "OPS-9",
			IssueSummary:  "Corrección de redondeo",
			IssueURL:      "https://example.test/browse/OPS-9",
			BackendRepo:   "pagos-api",
			BackendCommit: "9a8b7c6",
			BackendFiles:  map[string][]string{"app": {"Redondeo.php"}},
			PreservedConsider: []any{
				paragraph(textNode("Coordinar con el equipo de soporte")),
				considerTable([]string{"Paso agregado a mano"}),
			},
		},
	}
	for name, doc := range cases {
		golden.Assert(t, name, rendered{Title: BuildTitle(doc.IssueKey, doc.IssueSummary), ADF: Build(doc)})
	}
}

func TestGoldenQA(t *testing.T) {
	review := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	tasks := []provider.QAIssue{
		{Key: "OPS-7", URL: "https://example.test/browse/OPS-7", HasDeployDoc: true, PRMerged: true,
			ReviewTaskKey: "OPS-20", ReviewTaskURL: "https://example.test/browse/OPS-20"},
		{Key: "OPS-8", URL: "https://example.test/browse/OPS-8", HasCodingErrors: true, HasDevReturns: true,
			Observations: "falta validar el monto"},
		{Key: "OPS-9", URL: "https://example.test/browse/OPS-9", DeployDocUnknown: true, PRMerged: true},
	}

	golden.Assert(t, "qa-sprint", rendered{
		Title: BuildQATitle("Pagos", 12),
		ADF: BuildQA(QADoc{
			Sprint: 12, Module: "Pagos", Tasks: tasks, Date: review,
			LiderTecnico: "Líder de ejemplo", PMO: "PMO de ejemplo", QA: "QA de ejemplo",
		}),
	})

	golden.Assert(t, "qa-kanban-preservado", rendered{
		Title: BuildQAKanbanTitle("07/09 al 21/09/2026"),
		ADF: BuildQA(QADoc{
			Period: "07/09 al 21/09/2026", Tasks: tasks[:1], Date: review,
			Preserved: QAPreserved{
				Observation:    []any{paragraph(textNode("Sin bloqueos"))},
				Considerations: []any{paragraph(textNode("Ventana de despliegue el viernes"))},
			},
		}),
	})
}
