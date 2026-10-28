package qa

import "github.com/kaelence-01/draco-hq/internal/provider"

// Status is where an evaluated task stands in the report.
type Status int

const (
	Complete   Status = iota // the four checks pass
	Failing                  // at least one check fails
	Unverified               // no check fails, but the document link could not be looked up
)

// StatusOf classifies an evaluated task. A failed lookup is never a failed
// check: it is evidence that is missing, and a real failure is never hidden
// behind it.
func StatusOf(t provider.QAIssue) Status {
	switch {
	case t.HasCodingErrors || t.HasDevReturns || !t.PRMerged || (!t.DeployDocUnknown && !t.HasDeployDoc):
		return Failing
	case t.DeployDocUnknown:
		return Unverified
	}
	return Complete
}

// Counts is how the tasks of a report stand while they are evaluated.
type Counts struct {
	Total      int
	Complete   int
	Failing    int
	Unverified int
}

// Evaluated is the tasks classified so far.
func (c Counts) Evaluated() int { return c.Complete + c.Failing + c.Unverified }

// Pending is the tasks still to evaluate.
func (c Counts) Pending() int { return c.Total - c.Evaluated() }

// Tally counts the tasks, of which only the first evaluated have been
// evaluated: the others have no document check yet and are not classified.
func Tally(tasks []provider.QAIssue, evaluated int) Counts {
	c := Counts{Total: len(tasks)}
	for _, t := range tasks[:max(min(evaluated, len(tasks)), 0)] {
		switch StatusOf(t) {
		case Complete:
			c.Complete++
		case Failing:
			c.Failing++
		default:
			c.Unverified++
		}
	}
	return c
}
