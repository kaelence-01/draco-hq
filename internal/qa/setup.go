package qa

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/provider"
)

// ErrNoStatuses means the tracker returned no statuses for the project.
var ErrNoStatuses = errors.New("el proyecto no tiene estados")

// Setup reads from the tracker what the QA settings (qa_jira) are chosen
// from, so the user picks names instead of typing ids.
type Setup struct {
	src provider.QASetup

	Project  string
	Statuses []provider.Status
	Fields   []provider.Field
}

// NewSetup starts a setup over the tracker.
func NewSetup(src provider.QASetup) *Setup { return &Setup{src: src} }

// Read loads the statuses of the Jira project and the instance's fields.
func (s *Setup) Read(ctx context.Context, project string) error {
	project = strings.TrimSpace(project)
	if project == "" {
		return errors.New("falta la clave del proyecto de Jira (ej: OPS)")
	}
	statuses, err := s.src.ProjectStatuses(ctx, project)
	if err != nil {
		return err
	}
	if len(statuses) == 0 {
		return fmt.Errorf("%w '%s': revisa la clave", ErrNoStatuses, project)
	}
	fields, err := s.src.QAFields(ctx)
	if err != nil {
		return err
	}
	s.Project, s.Statuses, s.Fields = project, statuses, fields
	return nil
}

// FieldsOf returns the fields of one kind.
func (s *Setup) FieldsOf(kind provider.FieldKind) []provider.Field {
	return slices.DeleteFunc(slices.Clone(s.Fields), func(f provider.Field) bool { return f.Kind != kind })
}

// Choice is what the user picks: status and field ids, and the name of the
// status a task enters QA with.
type Choice struct {
	Testing      string
	Statuses     []string
	CodingErrors string
	DevReturns   string
	DevStatus    string
}

// Propose is the Choice the setup starts from: what current (nil: none)
// already has for the project and still exists in Jira, else a suggestion —
// the status named as the default testing one, and it plus the "done" ones
// as reached QA. Nothing is suggested for the counters: their names are the
// team's.
func (s *Setup) Propose(current *config.QAJiraConfig) Choice {
	var c Choice
	var have *config.QAJiraConfig
	if current != nil {
		have, _ = current.Lookup(s.Project)
		c.CodingErrors = s.field(current.Fields.CodingErrors, provider.FieldNumber)
		c.DevReturns = s.field(current.Fields.DevReturns, provider.FieldNumber)
		c.DevStatus = s.field(current.Fields.DevStatus, provider.FieldPRStatus)
	}
	if c.DevStatus == "" {
		if prs := s.FieldsOf(provider.FieldPRStatus); len(prs) > 0 {
			c.DevStatus = prs[0].ID
		}
	}
	if have != nil {
		c.Testing = s.statusName(have.Testing())
		for _, v := range have.Statuses {
			if i := s.status(v); i >= 0 {
				c.Statuses = append(c.Statuses, s.Statuses[i].ID)
			}
		}
	}
	if c.Testing == "" {
		c.Testing = s.statusName(config.DefaultTestingStatus)
	}
	if len(c.Statuses) == 0 {
		for _, st := range s.Statuses {
			if st.Done || (c.Testing != "" && st.Name == c.Testing) {
				c.Statuses = append(c.Statuses, st.ID)
			}
		}
	}
	return c
}

// Apply writes the choice into q (nil: a new qa_jira) for the setup's
// project, keeping everything else it has, and validates the result.
func (s *Setup) Apply(q *config.QAJiraConfig, c Choice) (*config.QAJiraConfig, error) {
	out := &config.QAJiraConfig{}
	if q != nil {
		cp := *q
		cp.Projects = maps.Clone(q.Projects)
		out = &cp
	}
	if len(c.Statuses) == 0 {
		return nil, errors.New("elige al menos un estado: los de una tarea que ya llegó a QA")
	}
	out.SetProject(s.Project, c.Statuses, c.Testing)
	out.Fields = config.QAJiraFields{CodingErrors: c.CodingErrors, DevReturns: c.DevReturns, DevStatus: c.DevStatus}
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return out, nil
}

// status finds a status by id or, without case, by name; -1 when missing.
func (s *Setup) status(v string) int {
	return slices.IndexFunc(s.Statuses, func(st provider.Status) bool { return st.ID == v || strings.EqualFold(st.Name, v) })
}

func (s *Setup) statusName(v string) string {
	if i := s.status(v); i >= 0 {
		return s.Statuses[i].Name
	}
	return ""
}

func (s *Setup) field(id string, kind provider.FieldKind) string {
	if slices.ContainsFunc(s.Fields, func(f provider.Field) bool { return f.ID == id && f.Kind == kind }) {
		return id
	}
	return ""
}
