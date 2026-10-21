package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kaelence-01/draco-hq/internal/build"
	"github.com/kaelence-01/draco-hq/internal/updater"
)

// releaseEnv publishes latest with binary for this platform on a fake GitHub
// (sums overrides checksums.txt when not empty), runs draco as version from a
// temporary executable and returns its path.
func releaseEnv(t *testing.T, version, latest string, binary []byte, sums string) string {
	t.Helper()
	configHome(t)
	asset := updater.AssetName(runtime.GOOS, runtime.GOARCH)
	if sums == "" {
		h := sha256.Sum256(binary)
		sums = hex.EncodeToString(h[:]) + "  " + asset + "\n"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/"+updater.Repo+"/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"tag_name":"` + latest + `"}`))
	})
	base := "/" + updater.Repo + "/releases/download/" + latest + "/"
	mux.HandleFunc(base+"checksums.txt", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(sums)) })
	mux.HandleFunc(base+asset, func(w http.ResponseWriter, _ *http.Request) { w.Write(binary) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	exe := filepath.Join(t.TempDir(), "draco")
	if err := os.WriteFile(exe, []byte("versión instalada"), 0o755); err != nil {
		t.Fatal(err)
	}
	prevUpd, prevExe, prevDir, prevVer := newUpdater, executable, installDir, build.Version
	t.Cleanup(func() { newUpdater, executable, installDir, build.Version = prevUpd, prevExe, prevDir, prevVer })
	newUpdater = func() *updater.Client { return &updater.Client{API: srv.URL, Download: srv.URL, HTTP: srv.Client()} }
	executable = func() (string, error) { return exe, nil }
	installDir = func() (string, error) { return filepath.Dir(exe), nil }
	build.Version = version
	return exe
}

func content(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestUpdateReplacesTheBinary(t *testing.T) {
	exe := releaseEnv(t, "v0.1.0", "v0.2.0", []byte("binario v0.2.0"), "")

	code, out, _ := run("update", "--check")
	if code != exitOK || !strings.Contains(out, "Nueva versión: v0.2.0") || content(t, exe) != "versión instalada" {
		t.Fatalf("--check solo consulta: %d\n%s", code, out)
	}
	code, out, errOut := run("update")
	if code != exitOK || !strings.Contains(out, "Actualizado a v0.2.0") || content(t, exe) != "binario v0.2.0" {
		t.Fatalf("%d\n%s\n%s", code, out, errOut)
	}
}

func TestUpdateRefusesATamperedBinary(t *testing.T) {
	exe := releaseEnv(t, "v0.1.0", "v0.2.0", []byte("binario alterado"), strings.Repeat("0", 64)+"  "+updater.AssetName(runtime.GOOS, runtime.GOARCH)+"\n")
	if code, _, errOut := run("update"); code != exitError || !strings.Contains(errOut, "no coincide") || content(t, exe) != "versión instalada" {
		t.Errorf("un binario que no coincide con checksums.txt no se instala: %d %s", code, errOut)
	}
}

func TestUpdateUpToDateAndDev(t *testing.T) {
	exe := releaseEnv(t, "v0.2.0", "v0.2.0", nil, "")
	if code, out, _ := run("update"); code != exitOK || !strings.Contains(out, "Ya tienes la versión más reciente") {
		t.Errorf("al día: %d\n%s", code, out)
	}
	build.Version = "dev"
	if code, out, _ := run("update"); code != exitOK || !strings.Contains(out, "versión de desarrollo") || content(t, exe) != "versión instalada" {
		t.Errorf("una versión de desarrollo nunca se reemplaza: %d\n%s", code, out)
	}
}

// startWith runs Start as at a terminal.
func startWith(input string, args ...string) (int, string) {
	var o, e bytes.Buffer
	a := New(strings.NewReader(input), &o, &e)
	a.tty = true
	code := a.Start(args)
	return code, o.String() + e.String()
}

func TestStartOffersToInstallADownloadedBinary(t *testing.T) {
	exe := releaseEnv(t, "v0.1.0", "v0.1.0", nil, "")
	dir := filepath.Join(t.TempDir(), "bin")
	installDir = func() (string, error) { return dir, nil }
	t.Setenv("PATH", dir)
	t.Setenv("DRACO_NO_UPDATE_CHECK", "1")

	code, out := startWith("n\n", "version")
	if code != exitOK || !strings.Contains(out, "No se instaló") || !strings.Contains(out, "draco v0.1.0") {
		t.Errorf("si no se instala, el comando corre igual: %d\n%s", code, out)
	}
	code, out = startWith("\n\n")
	if code != exitOK || !strings.Contains(out, "Instalado en") || !strings.Contains(out, "draco init") || !strings.Contains(out, "Presiona Enter") {
		t.Fatalf("doble clic: instala y espera Enter: %d\n%s", code, out)
	}
	if content(t, filepath.Join(dir, filepath.Base(installedName()))) != content(t, exe) {
		t.Error("el binario instalado es el que corría")
	}
}

func installedName() string {
	if runtime.GOOS == "windows" {
		return "draco.exe"
	}
	return "draco"
}

func TestStartShowsTheNewVersionNotice(t *testing.T) {
	releaseEnv(t, "v0.1.0", "v0.2.0", nil, "")
	code, out := startWith("", "project", "list")
	if code != exitOK || !strings.Contains(out, "Nueva versión disponible: v0.2.0 (tienes v0.1.0)") {
		t.Errorf("aviso tras el comando: %d\n%s", code, out)
	}
	if _, out := startWith("", "version"); strings.Contains(out, "Nueva versión") {
		t.Errorf("version no muestra el aviso:\n%s", out)
	}
	t.Setenv("DRACO_NO_UPDATE_CHECK", "1")
	if _, out := startWith("", "project", "list"); strings.Contains(out, "Nueva versión") {
		t.Errorf("DRACO_NO_UPDATE_CHECK lo apaga:\n%s", out)
	}
	if code, out, _ := run("project", "list"); code != exitOK || strings.Contains(out, "Nueva versión") {
		t.Error("fuera de una terminal nunca hay aviso")
	}
}

func TestVisualModeLearnsTheNewVersion(t *testing.T) {
	releaseEnv(t, "v0.1.0", "v0.2.0", nil, "")
	if got := latestVersion(context.Background()); got != "v0.2.0" {
		t.Fatalf("la TUI recibe la versión nueva: %q", got)
	}
	silent := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("con la caché al día no se consulta GitHub") }))
	t.Cleanup(silent.Close)
	newUpdater = func() *updater.Client { return &updater.Client{API: silent.URL, HTTP: silent.Client()} }
	if got := latestVersion(context.Background()); got != "v0.2.0" {
		t.Errorf("la versión sale de la caché: %q", got)
	}
	t.Setenv("DRACO_NO_UPDATE_CHECK", "1")
	if got := latestVersion(context.Background()); got != "" {
		t.Errorf("DRACO_NO_UPDATE_CHECK lo apaga: %q", got)
	}
	t.Setenv("DRACO_NO_UPDATE_CHECK", "")
	build.Version = "dev"
	if got := latestVersion(context.Background()); got != "" {
		t.Errorf("una versión de desarrollo no avisa: %q", got)
	}
}

func TestChecksUpdates(t *testing.T) {
	for args, want := range map[string]bool{
		"":             false,
		"update":       false,
		"version":      false,
		"help g":       false,
		"g -h":         false,
		"g -i OPS-7":   true,
		"backlog scan": true,
		"project list": true,
	} {
		if got := checksUpdates(strings.Fields(args)); got != want {
			t.Errorf("checksUpdates(%q) = %v", args, got)
		}
	}
}
