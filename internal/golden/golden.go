// Package golden compares test output with files under testdata/. Set
// DRACO_UPDATE_GOLDEN=1 to rewrite them after an intended change.
package golden

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// UpdateEnv is the environment variable that rewrites the golden files.
const UpdateEnv = "DRACO_UPDATE_GOLDEN"

// Assert compares got, as indented JSON, with testdata/<name>.golden.json.
func Assert(t *testing.T, name string, got any) {
	t.Helper()
	data, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("serializando %s: %v", name, err)
	}
	compare(t, name+".golden.json", append(data, '\n'))
}

// AssertText compares got with testdata/<file>, for output that reads better
// as plain text (Markdown).
func AssertText(t *testing.T, file, got string) {
	t.Helper()
	compare(t, file, []byte(got))
}

func compare(t *testing.T, file string, data []byte) {
	t.Helper()
	path := filepath.Join("testdata", file)

	if os.Getenv(UpdateEnv) == "1" {
		if err := os.MkdirAll("testdata", 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("falta %s (créalo con %s=1): %v", path, UpdateEnv, err)
	}
	if !bytes.Equal(data, want) {
		t.Errorf("%s cambió (si es intencional, regenera con %s=1)\n%s", path, UpdateEnv, firstDiff(want, data))
	}
}

func firstDiff(want, got []byte) string {
	w := strings.Split(string(want), "\n")
	g := strings.Split(string(got), "\n")
	for i := 0; i < len(w) || i < len(g); i++ {
		var a, b string
		if i < len(w) {
			a = w[i]
		}
		if i < len(g) {
			b = g[i]
		}
		if a != b {
			return "  línea " + strconv.Itoa(i+1) + ":\n    esperado: " + a + "\n    obtenido: " + b
		}
	}
	return ""
}
