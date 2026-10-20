// Package updater finds, downloads and installs draco's releases from its
// GitHub repository, verifying each binary against the release's
// checksums.txt (P-006).
package updater

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

// Repo is draco's own repository, where its releases are published.
const Repo = "kaelence-01/draco-hq"

// Platforms are the GOOS/GOARCH pairs every release publishes a binary for.
var Platforms = []string{"linux/amd64", "linux/arm64", "darwin/amd64", "darwin/arm64", "windows/amd64"}

// ErrNoReleases means the repository has no published release yet.
var ErrNoReleases = errors.New("todavía no hay versiones publicadas")

// maxBinary bounds a download, so a wrong URL cannot fill the disk.
const maxBinary = 200 << 20

// Client talks to GitHub; tests point API and Download at a fake server.
type Client struct {
	API      string // https://api.github.com
	Download string // https://github.com
	HTTP     *http.Client
}

// New returns a client for github.com.
func New() *Client {
	return &Client{API: "https://api.github.com", Download: "https://github.com", HTTP: http.DefaultClient}
}

// Latest returns the tag of the latest release (pre-releases excluded).
func (c *Client) Latest(ctx context.Context) (string, error) {
	body, err := c.get(ctx, fmt.Sprintf("%s/repos/%s/releases/latest", c.API, Repo), 1<<20)
	var status statusError
	if errors.As(err, &status) && status == http.StatusNotFound {
		return "", ErrNoReleases
	}
	if err != nil {
		return "", err
	}
	var r struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &r); err != nil || r.TagName == "" {
		return "", fmt.Errorf("respuesta inesperada de GitHub al buscar la última versión")
	}
	return r.TagName, nil
}

// AssetName is the release file of a platform: draco-linux-amd64,
// draco-windows-amd64.exe.
func AssetName(goos, goarch string) string {
	name := "draco-" + goos + "-" + goarch
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

// Fetch downloads the binary of version for this platform and returns it only
// if its SHA-256 matches the release's checksums.txt.
func (c *Client) Fetch(ctx context.Context, version string) ([]byte, error) {
	if !slices.Contains(Platforms, runtime.GOOS+"/"+runtime.GOARCH) {
		return nil, fmt.Errorf("no se publican binarios para %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return c.fetch(ctx, version, AssetName(runtime.GOOS, runtime.GOARCH))
}

func (c *Client) fetch(ctx context.Context, version, asset string) ([]byte, error) {
	base := fmt.Sprintf("%s/%s/releases/download/%s/", c.Download, Repo, version)
	sums, err := c.get(ctx, base+"checksums.txt", 1<<20)
	if err != nil {
		return nil, fmt.Errorf("no se pudo obtener checksums.txt: %w", err)
	}
	want, err := checksum(sums, asset)
	if err != nil {
		return nil, err
	}
	data, err := c.get(ctx, base+asset, maxBinary)
	if err != nil {
		return nil, fmt.Errorf("no se pudo descargar %s: %w", asset, err)
	}
	got := sha256.Sum256(data)
	if hex.EncodeToString(got[:]) != want {
		return nil, fmt.Errorf("el SHA-256 de %s no coincide con checksums.txt: no se instala", asset)
	}
	return data, nil
}

// checksum reads the hash of asset from a sha256sum-style file.
func checksum(sums []byte, asset string) (string, error) {
	sc := bufio.NewScanner(strings.NewReader(string(sums)))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) == 2 && strings.TrimPrefix(f[1], "*") == asset {
			return strings.ToLower(f[0]), nil
		}
	}
	return "", fmt.Errorf("checksums.txt no tiene el hash de %s", asset)
}

func (c *Client) get(ctx context.Context, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json, application/octet-stream")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, statusError(resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("la descarga es más grande de lo esperado")
	}
	return data, nil
}

type statusError int

func (s statusError) Error() string { return fmt.Sprintf("GitHub respondió %d", int(s)) }

// IsNewer reports whether candidate is a later vMAJOR.MINOR.PATCH than base;
// anything else (like "dev") is never newer nor older.
func IsNewer(candidate, base string) bool {
	c, ok1 := semver(candidate)
	b, ok2 := semver(base)
	if !ok1 || !ok2 {
		return false
	}
	for i := range c {
		if c[i] != b[i] {
			return c[i] > b[i]
		}
	}
	return false
}

func semver(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// Replace writes data over the executable at exe. The new file is written
// next to it and renamed into place, so a failure never leaves half a binary.
// Windows cannot overwrite a running .exe: it is renamed to .old first and
// CleanOld removes it on the next run.
func Replace(exe string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(exe), ".draco-update-*")
	if err != nil {
		return fmt.Errorf("no se pudo escribir junto a %s: %w", exe, err)
	}
	name := tmp.Name()
	_, err = tmp.Write(data)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(name, 0o755)
	}
	if err != nil {
		os.Remove(name)
		return fmt.Errorf("no se pudo escribir la nueva versión: %w", err)
	}
	if runtime.GOOS == "windows" {
		old := exe + ".old"
		os.Remove(old)
		if err := os.Rename(exe, old); err != nil {
			os.Remove(name)
			return fmt.Errorf("no se pudo apartar el ejecutable actual: %w", err)
		}
	}
	if err := os.Rename(name, exe); err != nil {
		os.Remove(name)
		return fmt.Errorf("no se pudo reemplazar el ejecutable: %w", err)
	}
	return nil
}

// CleanOld removes the .old copy a Windows update leaves behind.
func CleanOld(exe string) {
	if runtime.GOOS == "windows" {
		os.Remove(exe + ".old")
	}
}
