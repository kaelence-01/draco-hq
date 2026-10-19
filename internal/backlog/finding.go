// Package backlog turns technical debt in a local git repository into task
// candidates. It only reads git history, tracked files and dependency audits.
package backlog

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
)

// Kind identifies the detector that produced a finding.
type Kind string

const (
	KindHotspot    Kind = "hotspot"
	KindMarker     Kind = "marcador"
	KindLargeFile  Kind = "archivo-grande"
	KindUntested   Kind = "sin-test"
	KindDependency Kind = "dependencia"
)

// Severity orders findings by how urgent the underlying debt is.
type Severity string

const (
	SeverityHigh   Severity = "alta"
	SeverityMedium Severity = "media"
	SeverityLow    Severity = "baja"
)

// rank weighs the severity: higher is more urgent.
func (s Severity) rank() int {
	switch s {
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	default:
		return 1
	}
}

// Finding is one task candidate.
type Finding struct {
	// ID is stable across runs for the same kind and subject.
	ID       string   `json:"id"`
	Kind     Kind     `json:"tipo"`
	Severity Severity `json:"severidad"`
	// Points is a rough size suggestion.
	Points int    `json:"puntos"`
	Title  string `json:"titulo"`
	// Module comes from Options.Modules; empty when no pattern matches.
	Module   string   `json:"modulo,omitempty"`
	Files    []string `json:"archivos"`
	Evidence []string `json:"evidencia,omitempty"`
	// Score is the detector's raw measure, used to break ties.
	Score int `json:"metrica"`
}

// findingID hashes the kind and the subject (a file path or a package).
func findingID(kind Kind, subject string) string {
	sum := sha1.Sum([]byte(string(kind) + "\x00" + subject))
	return hex.EncodeToString(sum[:])[:10]
}

// Rank sorts findings most urgent first, in a deterministic order.
func Rank(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a.Severity.rank() != b.Severity.rank() {
			return a.Severity.rank() > b.Severity.rank()
		}
		if a.Points != b.Points {
			return a.Points > b.Points
		}
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		return firstFile(a) < firstFile(b)
	})
}

func firstFile(f Finding) string {
	if len(f.Files) == 0 {
		return ""
	}
	return f.Files[0]
}
