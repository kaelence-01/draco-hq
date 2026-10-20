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
	golden.AssertText(t, "draft-create.golden.md", "# "+d.Title+"\n\n"+d.Markdown)
}
