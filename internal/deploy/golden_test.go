package deploy

import (
	"testing"

	"github.com/kaelence-01/draco-hq/internal/golden"
)

func TestGoldenDraftCreate(t *testing.T) {
	s := readySession(newAPI())
	s.Target.VCSHost, s.Target.VCSOrg = "https://bitbucket.org", "mi-org"
	s.Target.Checklist = []string{"Respaldar base de datos", "Desplegar backend"}

	d := s.DraftCreate()
	golden.Assert(t, "draft-create", map[string]any{"title": d.Title, "adf": d.ADF})
}
