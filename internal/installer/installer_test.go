package installer

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestIsInstalled(t *testing.T) {
	dir := filepath.Join("/home", "yo", ".local", "bin")
	cases := map[string]bool{
		filepath.Join(dir, "draco"):                            true,
		filepath.Join("/home", "yo", "Descargas", "draco"):     false,
		filepath.Join(dir, "sub", "draco"):                     false,
		"/tmp/go-build123/b001/exe/draco":                      true,
		filepath.Join("/home", "yo", ".local", "bin-old", "x"): false,
	}
	for exe, want := range cases {
		if got := IsInstalled(exe, dir); got != want {
			t.Errorf("IsInstalled(%q) = %v", exe, got)
		}
	}
}

func TestInPathComparesWholeEntries(t *testing.T) {
	sep := string(os.PathListSeparator)
	value := strings.Join([]string{"/usr/bin", "/home/yo/.local/bin-old", "/home/yo/.local/bin/"}, sep)
	if !inPath(value, "/home/yo/.local/bin") {
		t.Error("la barra final no importa")
	}
	if inPath("/usr/bin"+sep+"/home/yo/.local/bin-old", "/home/yo/.local/bin") || inPath(value, "") {
		t.Error("una carpeta hermana no cuenta")
	}
}

func TestInstallCopiesAndAddsToPathOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("en Windows el PATH va al registro")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("PATH", "/usr/bin")
	src := filepath.Join(t.TempDir(), "draco-linux-amd64")
	if err := os.WriteFile(src, []byte("binario"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, ".local", "bin")

	r, err := Install(src, dir)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(r.Binary)
	info, _ := os.Stat(r.Binary)
	if string(b) != "binario" || filepath.Base(r.Binary) != "draco" || info.Mode().Perm()&0o100 == 0 {
		t.Errorf("binario instalado: %q %v %v", b, r.Binary, info.Mode())
	}
	if r.PathFile != filepath.Join(home, ".zshrc") || !r.PathChanged {
		t.Errorf("PATH: %+v", r)
	}
	if _, err := Install(src, dir); err != nil {
		t.Fatal(err)
	}
	rc, _ := os.ReadFile(filepath.Join(home, ".zshrc"))
	if strings.Count(string(rc), dir) != 1 {
		t.Errorf("la línea del PATH se escribe una sola vez:\n%s", rc)
	}

	t.Setenv("PATH", "/usr/bin:"+dir)
	if r, _ := Install(src, filepath.Join(dir)); r.PathChanged || r.PathFile != "" {
		t.Errorf("si ya está en el PATH no se toca nada: %+v", r)
	}
}
