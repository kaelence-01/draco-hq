package qa

import (
	"errors"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/atlassian"
	"github.com/kaelence-01/draco-hq/internal/golden"
)

func TestGoldenDraftCreateSprint(t *testing.T) {
	api := &fakeAPI{
		review: []atlassian.QAIssue{
			{Key: "OPS-7", URL: "https://example.test/browse/OPS-7", PRMerged: true},
			{Key: "OPS-8", URL: "https://example.test/browse/OPS-8", HasCodingErrors: true},
			{Key: "OPS-9", URL: "https://example.test/browse/OPS-9"},
		},
		assigned: []atlassian.QAIssue{
			{Key: "OPS-20", URL: "https://example.test/browse/OPS-20", Summary: "Revisión de Tarea - OPS-7"},
		},
		docLinks: map[string]bool{"OPS-7": true},
		docErr:   map[string]error{"OPS-9": errors.New("403")},
		novedad:  map[string]string{"OPS-8": "falta validar el monto"},
	}
	s := NewSession(api, sprintTarget(t))
	if err := s.Collect(); err != nil {
		t.Fatal(err)
	}
	for i := range s.Tasks {
		s.EvaluateTask(i)
	}

	d := s.DraftCreate()
	golden.Assert(t, "draft-create-sprint", map[string]any{"title": d.Title, "adf": d.ADF})
}
