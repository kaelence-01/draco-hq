//go:build windows

package installer

import (
	"errors"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// addToPath adds dir to the user PATH in the registry, without spawning a
// shell. The value keeps its type: rewriting a REG_EXPAND_SZ PATH as REG_SZ
// would freeze every %VAR% entry of other tools into a literal.
func addToPath(dir string) (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()

	current, kind, err := key.GetStringValue("Path")
	if errors.Is(err, registry.ErrNotExist) {
		current, kind = "", registry.EXPAND_SZ
	} else if err != nil {
		return "", err
	}
	if inPath(current, dir) {
		return "", nil
	}
	value := dir
	if trimmed := strings.TrimRight(current, "; "); trimmed != "" {
		value = trimmed + ";" + dir
	}
	if kind == registry.EXPAND_SZ {
		return "", key.SetExpandStringValue("Path", value)
	}
	return "", key.SetStringValue("Path", value)
}
