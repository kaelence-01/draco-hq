// Package git is a local git clone as a provider.ChangeSource.
package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kaelence-01/draco-hq/internal/provider"
)

// Local reads commits from the clone at the given workDir.
type Local struct{}

var _ provider.ChangeSource = Local{}

// ChangedFiles returns the deduplicated union of the commits' files, in order
// of first appearance.
func (Local) ChangedFiles(ctx context.Context, workDir string, hashes []string) ([]string, error) {
	seen := make(map[string]bool)
	var all []string
	for _, h := range hashes {
		files, err := changedFiles(ctx, h, workDir)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			if !seen[f] {
				seen[f] = true
				all = append(all, f)
			}
		}
	}
	return all, nil
}

// logSep separates the fields of one commit in the git log output.
const logSep = "\x1f"

// Commits lists the commits of the current branch, or of every branch.
func (Local) Commits(ctx context.Context, workDir string, q provider.CommitQuery) ([]provider.Commit, error) {
	out, err := run(ctx, workDir, logArgs(q)...)
	if err != nil {
		return nil, fmt.Errorf("error al leer el historial: %w", err)
	}
	return parseLog(out), nil
}

// logArgs is the git log of q. Grep goes inside its own argument, as fixed
// text: it is never read as an option or a regular expression.
func logArgs(q provider.CommitQuery) []string {
	format := strings.Join([]string{"%H", "%h", "%s", "%an", "%ae", "%aI"}, logSep)
	args := []string{"log", fmt.Sprintf("-n%d", q.Limit), "--format=" + format}
	if q.Skip > 0 {
		args = append(args, fmt.Sprintf("--skip=%d", q.Skip))
	}
	if q.AllBranches {
		args = append(args, "--branches", "--remotes")
	}
	if q.Grep != "" {
		args = append(args, "--regexp-ignore-case", "--fixed-strings", "--grep="+q.Grep)
	}
	return args
}

// Identity returns the user.email of the clone; empty when it is not set.
func (Local) Identity(ctx context.Context, workDir string) (string, error) {
	out, err := run(ctx, workDir, "config", "user.email")
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 1 {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("no se pudo leer user.email: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// CurrentBranch returns the branch checked out in the clone.
func (Local) CurrentBranch(ctx context.Context, workDir string) (string, error) {
	out, err := run(ctx, workDir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("no se pudo leer la rama actual: %w", err)
	}
	return strings.TrimSpace(out), nil
}

func parseLog(out string) []provider.Commit {
	var commits []provider.Commit
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(line, logSep)
		if len(f) != 6 {
			continue
		}
		date, _ := time.Parse(time.RFC3339, f[5])
		commits = append(commits, provider.Commit{Hash: f[0], ShortHash: f[1], Subject: f[2], Author: f[3], Email: f[4], Date: date})
	}
	return commits
}

func changedFiles(ctx context.Context, commitHash, workDir string) ([]string, error) {
	out, err := run(ctx, workDir, "show", "--name-only", "--format=", commitHash)
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("%s", explainGitError(string(ee.Stderr), commitHash, workDir))
		}
		return nil, fmt.Errorf("error al leer el commit %s: %w", commitHash, err)
	}

	var files []string
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			files = append(files, line)
		}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("el commit %s no tiene archivos o no existe", commitHash)
	}
	return files, nil
}

// run executes git by its absolute path (P-005) in workDir; empty is the
// current directory.
func run(ctx context.Context, workDir string, args ...string) (string, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return "", fmt.Errorf("git no encontrado en PATH: %w", err)
	}
	cmd := exec.CommandContext(ctx, gitPath, args...)
	if workDir != "" {
		cmd.Dir = filepath.Clean(workDir)
	}
	out, err := cmd.Output()
	return string(out), err
}

// explainGitError turns git's stderr into an actionable message: unknown or
// ambiguous hash, or not a git repository.
func explainGitError(stderr, commitHash, workDir string) string {
	s := strings.ToLower(stderr)
	loc := workDir
	if loc == "" {
		if wd, err := os.Getwd(); err == nil {
			loc = wd
		} else {
			loc = "."
		}
	}

	switch {
	case strings.Contains(s, "bad object"),
		strings.Contains(s, "unknown revision"),
		strings.Contains(s, "bad revision"):
		return fmt.Sprintf(
			"el commit %s no existe en el repo local (%s).\n"+
				"        Posibles causas:\n"+
				"          1. Tu repo local está desactualizado. Ejecuta:\n"+
				"               cd %q && git fetch --all\n"+
				"          2. El hash pertenece al otro repo (¿backend en lugar de frontend, o viceversa?)\n"+
				"          3. El hash es incorrecto. Verifícalo en tu VCS",
			commitHash, loc, loc)

	case strings.Contains(s, "not a git repository"):
		return fmt.Sprintf(
			"el directorio %s no es un repositorio Git.\n"+
				"        Soluciones:\n"+
				"          • Ejecuta draco dentro del clon local del repo correspondiente, o\n"+
				"          • Configura las rutas backend_path / frontend_path con: draco init",
			loc)

	case strings.Contains(s, "ambiguous argument"):
		return fmt.Sprintf(
			"el hash %s es ambiguo o demasiado corto. Usa al menos 7-8 caracteres.",
			commitHash)
	}

	return fmt.Sprintf("error al leer el commit %s: %s", commitHash, strings.TrimSpace(stderr))
}
