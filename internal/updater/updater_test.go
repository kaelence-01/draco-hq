package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeGitHub serves a release with one asset; sums is checksums.txt.
func fakeGitHub(t *testing.T, latest, asset string, binary []byte, sums string) *Client {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/"+Repo+"/releases/latest", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"tag_name":"` + latest + `"}`))
	})
	base := "/" + Repo + "/releases/download/" + latest + "/"
	mux.HandleFunc(base+"checksums.txt", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(sums)) })
	mux.HandleFunc(base+asset, func(w http.ResponseWriter, _ *http.Request) { w.Write(binary) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &Client{API: srv.URL, Download: srv.URL, HTTP: srv.Client()}
}

func sha(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func TestIsNewer(t *testing.T) {
	cases := []struct {
		c, b string
		want bool
	}{
		{"v0.2.0", "v0.1.9", true},
		{"v1.0.0", "v0.9.9", true},
		{"v0.1.10", "v0.1.9", true},
		{"v0.1.0", "v0.1.0", false},
		{"v0.1.0", "v0.2.0", false},
		{"v0.2.0", "dev", false},
		{"dev", "v0.1.0", false},
		{"v0.2", "v0.1.0", false},
	}
	for _, c := range cases {
		if got := IsNewer(c.c, c.b); got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v", c.c, c.b, got)
		}
	}
}

func TestAssetName(t *testing.T) {
	if AssetName("windows", "amd64") != "draco-windows-amd64.exe" || AssetName("darwin", "arm64") != "draco-darwin-arm64" {
		t.Error("nombres de los binarios del release")
	}
}

func TestFetchVerifiesTheChecksum(t *testing.T) {
	bin := []byte("binario v0.2.0")
	asset := "draco-linux-amd64"
	good := sha(bin) + "  " + asset + "\n" + sha([]byte("otro")) + "  draco-darwin-arm64\n"

	c := fakeGitHub(t, "v0.2.0", asset, bin, good)
	if latest, err := c.Latest(context.Background()); err != nil || latest != "v0.2.0" {
		t.Fatalf("Latest: %q %v", latest, err)
	}
	if data, err := c.fetch(context.Background(), "v0.2.0", asset); err != nil || string(data) != string(bin) {
		t.Errorf("descarga verificada: %q %v", data, err)
	}

	tampered := fakeGitHub(t, "v0.2.0", asset, []byte("binario alterado"), good)
	if _, err := tampered.fetch(context.Background(), "v0.2.0", asset); err == nil || !strings.Contains(err.Error(), "no coincide") {
		t.Errorf("un binario alterado nunca se devuelve: %v", err)
	}
	missing := fakeGitHub(t, "v0.2.0", asset, bin, sha(bin)+"  draco-darwin-arm64\n")
	if _, err := missing.fetch(context.Background(), "v0.2.0", asset); err == nil || !strings.Contains(err.Error(), "no tiene el hash") {
		t.Errorf("sin hash publicado no se instala: %v", err)
	}
	none := httptest.NewServer(http.NotFoundHandler())
	defer none.Close()
	empty := &Client{API: none.URL, HTTP: none.Client()}
	if _, err := empty.Latest(context.Background()); !errors.Is(err, ErrNoReleases) {
		t.Errorf("un repo sin releases: %v", err)
	}
	if _, err := c.fetch(context.Background(), "v9.9.9", asset); err == nil {
		t.Error("una versión que no existe es error")
	}
}

func TestReplaceSwapsTheFile(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "draco")
	if err := os.WriteFile(exe, []byte("viejo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Replace(exe, []byte("nuevo")); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(exe)
	info, _ := os.Stat(exe)
	if string(b) != "nuevo" || info.Mode().Perm()&0o100 == 0 {
		t.Errorf("reemplazo: %q %v", b, info.Mode())
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(exe), ".draco-update-*")); len(left) != 0 {
		t.Errorf("no quedan temporales: %v", left)
	}
}

func TestNotifierCachesForADay(t *testing.T) {
	now := time.Date(2026, 10, 19, 9, 0, 0, 0, time.UTC)
	n := &Notifier{Client: fakeGitHub(t, "v0.2.0", "x", nil, ""), Cache: filepath.Join(t.TempDir(), "version_check.json"), Now: func() time.Time { return now }}

	if !n.Stale() || n.Cached("v0.1.0") != "" {
		t.Fatal("sin caché hay que consultar")
	}
	if got := n.Refresh(context.Background(), "v0.1.0"); got != "v0.2.0" {
		t.Errorf("Refresh = %q", got)
	}
	if n.Stale() || n.Cached("v0.1.0") != "v0.2.0" || n.Cached("v0.2.0") != "" {
		t.Error("la caché responde sin red durante el día")
	}
	now = now.Add(25 * time.Hour)
	if !n.Stale() {
		t.Error("al día siguiente se vuelve a consultar")
	}

	broken := &Notifier{Client: &Client{API: "http://127.0.0.1:1", HTTP: http.DefaultClient}, Cache: n.Cache, Now: n.Now}
	if got := broken.Refresh(context.Background(), "v0.1.0"); got != "" || broken.Cached("v0.1.0") != "v0.2.0" {
		t.Error("una consulta fallida no borra lo que se sabía")
	}
}
