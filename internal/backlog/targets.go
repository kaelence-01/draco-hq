package backlog

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/config"
)

var (
	// ErrNoRepoPath means the project has no path for the repo asked for.
	ErrNoRepoPath = errors.New("el proyecto no tiene ruta para ese repo")

	// ErrNoModules means a module filter was asked for a repo whose config
	// has no modules.
	ErrNoModules = errors.New("el filtro de módulos necesita modules en la config del repo")
)

// Target is one repo to scan.
type Target struct {
	Repo string // "backend" or "frontend"
	Dir  string
}

// Targets resolves what to scan: path, that folder; repo, that repo of the
// project; neither, every repo the project has a path for. Without a project
// (or one with no paths), the current folder.
func Targets(proj *config.ProjectConfig, repo, path string) ([]Target, error) {
	if repo != "" && repo != "backend" && repo != "frontend" {
		return nil, fmt.Errorf("el repo debe ser backend o frontend, no %q", repo)
	}
	label := repo
	if label == "" {
		label = "backend"
	}
	if path != "" {
		return []Target{{Repo: label, Dir: config.ExpandHome(path)}}, nil
	}

	var targets []Target
	if proj != nil {
		if repo != "frontend" && proj.BackendPath != "" {
			targets = append(targets, Target{Repo: "backend", Dir: config.ExpandHome(proj.BackendPath)})
		}
		if repo != "backend" && proj.FrontendPath != "" {
			targets = append(targets, Target{Repo: "frontend", Dir: config.ExpandHome(proj.FrontendPath)})
		}
	}
	if len(targets) > 0 {
		return targets, nil
	}
	if proj != nil && repo != "" {
		return nil, fmt.Errorf("%w (%s_path)", ErrNoRepoPath, repo)
	}
	return []Target{{Repo: label, Dir: "."}}, nil
}

var sinceShorthand = regexp.MustCompile(`^(\d+)([dwm])$`)

// normalizeSince accepts 90d, 12w and 3m on top of anything `git log --since`
// understands.
func normalizeSince(v string) string {
	m := sinceShorthand.FindStringSubmatch(v)
	if m == nil {
		return v
	}
	unit := map[string]string{"d": "days", "w": "weeks", "m": "months"}[m[2]]
	return m[1] + " " + unit + " ago"
}

// OptionsFor layers since over the repo's config; what stays empty takes the
// defaults of the scan.
func OptionsFor(rc *config.BacklogRepoConfig, since string) Options {
	var o Options
	if rc != nil {
		o = Options{
			Since:       rc.Since,
			Extensions:  rc.Extensions,
			Exclude:     rc.Exclude,
			MaxLines:    rc.MaxLines,
			MinCommits:  rc.MinCommits,
			FixKeywords: rc.FixKeywords,
			ClassGlobs:  rc.ClassGlobs,
			TestsDir:    rc.TestsDir,
			TestGlobs:   rc.TestGlobs,
			Modules:     rc.Modules,
		}
	}
	if since != "" {
		o.Since = since
	}
	o.Since = normalizeSince(o.Since)
	return o
}

// SplitModules reads a comma-separated list of modules.
func SplitModules(v string) []string {
	var out []string
	for _, m := range strings.Split(v, ",") {
		if m = strings.TrimSpace(m); m != "" {
			out = append(out, m)
		}
	}
	return out
}

// ApplyModules keeps the findings of modules, if any; ErrNoModules when the
// repo has no modules configured.
func ApplyModules(rep *Report, modules []string) error {
	if len(modules) == 0 {
		return nil
	}
	if len(rep.Modules) == 0 {
		return ErrNoModules
	}
	rep.FilterModules(modules)
	return nil
}
