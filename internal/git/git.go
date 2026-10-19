package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CommitFiles holds the files changed in a commit, grouped by repo.
type CommitFiles struct {
	RepoName string
	AppName  string
	Files    []string
}

// GetChangedFiles lists the files changed by a commit. An empty workDir is the
// current directory.
func GetChangedFiles(commitHash, workDir string) ([]string, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git no encontrado en PATH: %w", err)
	}
	cmd := exec.Command(gitPath, "show", "--name-only", "--format=", commitHash)
	if workDir != "" {
		cmd.Dir = filepath.Clean(workDir)
	}
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("%s", explainGitError(string(ee.Stderr), commitHash, workDir))
		}
		return nil, fmt.Errorf("error al leer el commit %s: %w", commitHash, err)
	}

	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		files = append(files, line)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("el commit %s no tiene archivos o no existe", commitHash)
	}

	return files, nil
}

// GetChangedFilesMulti returns the deduplicated union, in order of first appearance.
func GetChangedFilesMulti(hashes []string, workDir string) ([]string, error) {
	seen := make(map[string]bool)
	var all []string
	for _, h := range hashes {
		files, err := GetChangedFiles(h, workDir)
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
				"          3. El hash es incorrecto. Verifícalo en Bitbucket",
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

// GroupByDirectory groups file names by their parent directory.
func GroupByDirectory(files []string) map[string][]string {
	groups := make(map[string][]string)
	for _, f := range files {
		parts := strings.Split(f, "/")
		var dir string
		if len(parts) == 1 {
			dir = "."
		} else {
			dir = strings.Join(parts[:len(parts)-1], "/")
		}
		groups[dir] = append(groups[dir], parts[len(parts)-1])
	}
	return groups
}
