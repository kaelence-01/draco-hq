package backlog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// repo runs read-only git commands in a local repository, stopping them when
// ctx, the scan's, is canceled.
type repo struct {
	ctx     context.Context
	dir     string
	gitPath string
}

// openRepo resolves git to an absolute path (P-005) and checks that dir is a work tree.
func openRepo(ctx context.Context, dir string) (*repo, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git no encontrado en PATH: %w", err)
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("ruta inválida %s: %w", dir, err)
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("la ruta %s no existe o no es una carpeta", abs)
	}

	r := &repo{ctx: ctx, dir: abs, gitPath: gitPath}
	if _, err := r.git("rev-parse", "--is-inside-work-tree"); err != nil {
		return nil, fmt.Errorf("%s no es un repositorio git", abs)
	}
	return r, nil
}

func (r *repo) git(args ...string) ([]byte, error) {
	cmd := exec.CommandContext(r.ctx, r.gitPath, args...)
	cmd.Dir = r.dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && stderr.Len() > 0 {
			return out, fmt.Errorf("git %s: %s", gitSubcommand(args), strings.TrimSpace(stderr.String()))
		}
		return out, fmt.Errorf("git %s: %w", gitSubcommand(args), err)
	}
	return out, nil
}

// trackedFiles lists tracked files as slash-separated repo-relative paths.
func (r *repo) trackedFiles() ([]string, error) {
	out, err := r.git("ls-files", "-z")
	if err != nil {
		return nil, err
	}
	var files []string
	for _, f := range strings.Split(string(out), "\x00") {
		if f != "" {
			files = append(files, f)
		}
	}
	return files, nil
}

func (r *repo) readFile(rel string) ([]byte, error) {
	return os.ReadFile(filepath.Join(r.dir, filepath.FromSlash(rel)))
}

// gitSubcommand names the subcommand in args, skipping leading "-c key=value".
func gitSubcommand(args []string) string {
	for i := 0; i < len(args); i++ {
		if args[i] == "-c" {
			i++
			continue
		}
		return args[i]
	}
	return ""
}
