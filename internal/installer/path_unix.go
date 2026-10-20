//go:build !windows

package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// addToPath appends an export line to the rc file of the user's shell, once.
func addToPath(dir string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	rc := filepath.Join(home, ".bashrc")
	if strings.Contains(os.Getenv("SHELL"), "zsh") {
		rc = filepath.Join(home, ".zshrc")
	}
	line := fmt.Sprintf("export PATH=\"$PATH:%s\"", dir)
	if b, err := os.ReadFile(rc); err == nil && strings.Contains(string(b), line) {
		return rc, nil
	}
	f, err := os.OpenFile(rc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	if _, err = fmt.Fprintf(f, "\n# draco\n%s\n", line); err != nil {
		f.Close()
		return "", err
	}
	return rc, f.Close()
}
