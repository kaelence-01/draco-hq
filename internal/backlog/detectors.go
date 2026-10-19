package backlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// maxEvidence caps the evidence lines kept per finding.
const maxEvidence = 5

// maxHotspotSubjects caps the commit subjects quoted as hotspot evidence.
const maxHotspotSubjects = 3

// fileChurn accumulates the commits that touched one file in the period.
type fileChurn struct {
	commits  int
	fixes    int      // commits whose subject looks like a fix (FixKeywords)
	subjects []string // most recent first, as git log lists them
}

// hotspotLogArgs starts each commit with \x1e so subjects and file lists cannot be confused.
func hotspotLogArgs(since string) []string {
	return []string{"-c", "core.quotePath=false", "log", "--since=" + since, "--no-merges", "--format=%x1e%s", "--name-only"}
}

// parseChurn counts, per file, the commits and fixes in hotspotLogArgs output.
func parseChurn(log string, o Options) map[string]*fileChurn {
	churn := map[string]*fileChurn{}

	for _, record := range strings.Split(log, "\x1e") {
		lines := strings.Split(strings.TrimSpace(record), "\n")
		subject := strings.TrimSpace(lines[0])
		if subject == "" {
			continue
		}
		isFix := o.isFixCommit(subject)
		seen := map[string]bool{}

		for _, f := range lines[1:] {
			f = strings.TrimSpace(f)
			if f == "" || seen[f] {
				continue
			}
			seen[f] = true

			fc := churn[f]
			if fc == nil {
				fc = &fileChurn{}
				churn[f] = fc
			}
			fc.commits++
			if isFix {
				fc.fixes++
			}
			if len(fc.subjects) < maxHotspotSubjects && !slices.Contains(fc.subjects, subject) {
				fc.subjects = append(fc.subjects, subject)
			}
		}
	}
	return churn
}

// hotspotFindings reports source files changed in at least MinCommits commits.
func hotspotFindings(churn map[string]*fileChurn, tracked map[string]bool, lines map[string]int, o Options) []Finding {
	var out []Finding
	for file, fc := range churn {
		if fc.commits < o.MinCommits || !tracked[file] || !o.isSource(file) {
			continue
		}

		size := lines[file]
		sev, pts := SeverityMedium, 2
		title := fmt.Sprintf("Cubrir con tests el flujo más modificado de %s (%d commits en el período)", path.Base(file), fc.commits)
		if size > o.MaxLines {
			sev, pts = SeverityHigh, 3
			title = fmt.Sprintf("Extraer una responsabilidad de %s y cubrirla con tests (%d commits en el período, %d líneas)", path.Base(file), fc.commits, size)
		}

		var evidence []string
		if fc.fixes > 0 {
			evidence = append(evidence, fmt.Sprintf("%d de %d commits parecen correcciones", fc.fixes, fc.commits))
		}
		for _, s := range fc.subjects {
			evidence = append(evidence, "Commit: "+truncate(s, 110))
		}

		out = append(out, Finding{
			ID:       findingID(KindHotspot, file),
			Kind:     KindHotspot,
			Severity: sev,
			Points:   pts,
			Title:    title,
			Files:    []string{file},
			Evidence: evidence,
			Score:    fc.commits,
		})
	}
	return out
}

// markerPattern matches upper-case TODO/FIXME/HACK only, so the Spanish "todo"
// does not count. XXX is left out: it is common as a placeholder in data.
const markerPattern = `(^|[^A-Za-z0-9_])(TODO|FIXME|HACK)([^A-Za-z0-9_]|$)`

var markerRe = regexp.MustCompile(markerPattern)

// markerGrepArgs uses --null so paths containing ':' parse correctly.
func markerGrepArgs() []string {
	return []string{"grep", "-n", "-I", "--null", "-E", "-e", markerPattern}
}

type markerHit struct {
	line   int
	text   string
	urgent bool // FIXME or HACK: a known defect or shortcut, not a plan
}

// parseMarkers groups markerGrepArgs output by file. Older git separates the
// line number with ':' instead of NUL ("path\x00lineno:text").
func parseMarkers(out string) map[string][]markerHit {
	hits := map[string][]markerHit{}

	for _, l := range strings.Split(out, "\n") {
		file, rest, ok := strings.Cut(l, "\x00")
		if !ok {
			continue
		}
		num, text, ok := strings.Cut(rest, "\x00")
		if !ok {
			if num, text, ok = strings.Cut(rest, ":"); !ok {
				continue
			}
		}
		n, err := strconv.Atoi(num)
		if err != nil {
			continue
		}

		m := markerRe.FindStringSubmatch(text)
		hits[file] = append(hits[file], markerHit{
			line:   n,
			text:   strings.TrimSpace(text),
			urgent: m != nil && m[2] != "TODO",
		})
	}
	return hits
}

// markerFindings reports one finding per source file with pending markers.
func markerFindings(hits map[string][]markerHit, o Options) []Finding {
	var out []Finding
	for file, hs := range hits {
		if !o.isSource(file) {
			continue
		}

		sev, pts := SeverityLow, 1
		var evidence []string
		for _, h := range hs {
			if h.urgent {
				sev = SeverityMedium
			}
			if len(evidence) < maxEvidence {
				evidence = append(evidence, fmt.Sprintf("L%d: %s", h.line, truncate(h.text, 120)))
			}
		}
		if len(hs) > 3 {
			pts = 2
		}

		out = append(out, Finding{
			ID:       findingID(KindMarker, file),
			Kind:     KindMarker,
			Severity: sev,
			Points:   pts,
			Title:    fmt.Sprintf("Resolver %d marcador(es) pendiente(s) en %s", len(hs), path.Base(file)),
			Files:    []string{file},
			Evidence: evidence,
			Score:    len(hs),
		})
	}
	return out
}

// countLines counts the last line even without a trailing newline.
func countLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	n := bytes.Count(data, []byte{'\n'})
	if data[len(data)-1] != '\n' {
		n++
	}
	return n
}

// largeFileFinding reports a source file over MaxLines, or nil. It is meant for
// files that are not hotspots, so it ranks low unless the file is huge.
func largeFileFinding(file string, lines int, o Options) *Finding {
	if lines <= o.MaxLines {
		return nil
	}

	sev := SeverityLow
	if lines >= 3*o.MaxLines {
		sev = SeverityMedium
	}
	return &Finding{
		ID:       findingID(KindLargeFile, file),
		Kind:     KindLargeFile,
		Severity: sev,
		Points:   3,
		Title:    fmt.Sprintf("Extraer un módulo de %s (%d líneas, umbral %d)", path.Base(file), lines, o.MaxLines),
		Files:    []string{file},
		Score:    lines,
	}
}

var identifierRe = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)

// collectIdentifiers adds every identifier in src to set.
func collectIdentifiers(src []byte, set map[string]bool) {
	for _, id := range identifierRe.FindAll(src, -1) {
		set[string(id)] = true
	}
}

// className returns the class a file should declare: the base name, or its
// parts joined in PascalCase (pago-detalle.service.ts → PagoDetalleService).
func className(file string) string {
	base := path.Base(file)
	name := strings.TrimSuffix(base, path.Ext(base))
	if !strings.ContainsAny(name, ".-_") {
		return name
	}

	var b strings.Builder
	for _, part := range strings.FieldsFunc(name, func(r rune) bool { return r == '.' || r == '-' || r == '_' }) {
		b.WriteString(strings.ToUpper(part[:1]))
		b.WriteString(part[1:])
	}
	return b.String()
}

// untestedFindings reports ClassGlobs files whose class never appears in the tests.
func untestedFindings(tracked []string, testIdentifiers map[string]bool, o Options) []Finding {
	var out []Finding
	for _, file := range tracked {
		if !matchAny(o.ClassGlobs, file) || !o.isSource(file) {
			continue
		}
		name := className(file)
		if testIdentifiers[name] {
			continue
		}
		out = append(out, Finding{
			ID:       findingID(KindUntested, file),
			Kind:     KindUntested,
			Severity: SeverityLow,
			Points:   2,
			Title:    fmt.Sprintf("Agregar tests para %s", name),
			Files:    []string{file},
		})
	}
	return out
}

// composerAdvisory is the part of `composer audit --format=json` we use.
type composerAdvisory struct {
	Title            string `json:"title"`
	CVE              string `json:"cve"`
	AffectedVersions string `json:"affectedVersions"`
}

// parseComposerAudit returns advisories per package; with none, composer prints [], not {}.
func parseComposerAudit(out []byte) (map[string][]composerAdvisory, error) {
	var raw struct {
		Advisories json.RawMessage `json:"advisories"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("salida de composer audit no reconocida: %w", err)
	}

	trimmed := bytes.TrimSpace(raw.Advisories)
	if len(trimmed) == 0 || trimmed[0] == '[' || string(trimmed) == "null" {
		return map[string][]composerAdvisory{}, nil
	}

	advisories := map[string][]composerAdvisory{}
	if err := json.Unmarshal(trimmed, &advisories); err != nil {
		return nil, fmt.Errorf("advisories de composer audit no reconocidos: %w", err)
	}
	return advisories, nil
}

// dependencyFindings reports one finding per vulnerable package.
func dependencyFindings(advisories map[string][]composerAdvisory, lockFile string) []Finding {
	var out []Finding
	for pkg, advs := range advisories {
		if len(advs) == 0 {
			continue
		}

		var evidence []string
		for _, a := range advs {
			if len(evidence) == maxEvidence {
				break
			}
			id := a.CVE
			if id == "" {
				id = "sin CVE"
			}
			evidence = append(evidence, fmt.Sprintf("%s: %s (afecta %s)", id, a.Title, a.AffectedVersions))
		}

		out = append(out, Finding{
			ID:       findingID(KindDependency, pkg),
			Kind:     KindDependency,
			Severity: SeverityHigh,
			Points:   min(len(advs), 3),
			Title:    fmt.Sprintf("Actualizar %s: %d vulnerabilidad(es) reportada(s)", pkg, len(advs)),
			Files:    []string{lockFile},
			Evidence: evidence,
			Score:    len(advs),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

// truncate shortens s to at most n runes, marking the cut with "…".
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
