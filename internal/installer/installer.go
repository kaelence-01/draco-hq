// Package installer copies a downloaded draco binary to a folder of the
// user's PATH on its first run.
package installer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Dir is where draco installs itself.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			base = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(base, "Programs", "draco"), nil
	}
	return filepath.Join(home, ".local", "bin"), nil
}

// Executable is the running binary with symlinks resolved.
func Executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// IsInstalled reports whether exe needs no installation: it already runs from
// dir, or it is a `go run` / `go test` build.
func IsInstalled(exe, dir string) bool {
	slashed := filepath.ToSlash(exe)
	if strings.Contains(slashed, "/go-build") {
		return true
	}
	rel, err := filepath.Rel(dir, filepath.Dir(exe))
	return err == nil && rel == "."
}

// Result is what Install changed.
type Result struct {
	Binary string // installed binary
	// PathFile is the shell file that got the PATH line; empty when nothing
	// was written (the folder was already in PATH, or Windows' registry).
	PathFile string
	// PathChanged means a new terminal is needed to find draco.
	PathChanged bool
}

// Install copies exe to dir as draco (draco.exe) and adds dir to the user's
// PATH when it is not there yet.
func Install(exe, dir string) (*Result, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("no se pudo crear %s: %w", dir, err)
	}
	name := "draco"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	r := &Result{Binary: filepath.Join(dir, name)}
	if err := copyFile(exe, r.Binary); err != nil {
		return nil, fmt.Errorf("no se pudo copiar el ejecutable: %w", err)
	}
	if inPath(os.Getenv("PATH"), dir) {
		return r, nil
	}
	var err error
	if r.PathFile, err = addToPath(dir); err != nil {
		return r, fmt.Errorf("draco quedó en %s pero no se pudo agregar al PATH: %w", dir, err)
	}
	r.PathChanged = true
	return r, nil
}

// copyFile writes next to dst and renames, so an installed draco that is
// running is replaced whole.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".draco-install-*")
	if err != nil {
		return err
	}
	_, err = io.Copy(tmp, in)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(tmp.Name(), 0o755)
	}
	if err == nil {
		err = os.Rename(tmp.Name(), dst)
	}
	if err != nil {
		os.Remove(tmp.Name())
	}
	return err
}

// inPath reports whether dir is one of the entries of a PATH value. Entries
// are compared whole: a substring of the value would match a sibling folder
// like draco-old.
func inPath(pathValue, dir string) bool {
	want := canonEntry(dir)
	if want == "" {
		return false
	}
	for _, e := range filepath.SplitList(pathValue) {
		if canonEntry(e) == want {
			return true
		}
	}
	return false
}

func canonEntry(e string) string {
	e = strings.Trim(strings.TrimSpace(e), `"`)
	if e == "" {
		return ""
	}
	e = strings.TrimRight(filepath.Clean(e), `\/`)
	if runtime.GOOS == "windows" {
		e = strings.ToLower(e)
	}
	return e
}
