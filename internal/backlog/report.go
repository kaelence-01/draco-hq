package backlog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaelence-01/draco-hq/internal/config"
)

// ReportName is <project>-<repo>.json, or <folder>-<repo>.json without a
// project; a module filter adds the modules, so a filtered report never
// overwrites the full one.
func ReportName(project, dir, repo string, modules []string) string {
	base := project
	if base == "" {
		if abs, err := filepath.Abs(dir); err == nil {
			base = filepath.Base(abs)
		}
	}
	name := base + "-" + repo
	for _, m := range modules {
		name += "-" + ModuleKey(m)
	}
	return name + ".json"
}

// SaveReport writes the report as UTF-8 JSON (PowerShell 5.1 `>` writes
// UTF-16) and returns where. Without a path it goes to a backlog folder next
// to config.yaml, as name; the report holds internal paths and commit
// messages, so it is private (0600).
func SaveReport(rep *Report, path, name string) (string, error) {
	if path == "" {
		cfgPath, err := config.ConfigPath()
		if err != nil {
			return "", fmt.Errorf("no se pudo resolver la carpeta de configuración: %w", err)
		}
		path = filepath.Join(filepath.Dir(cfgPath), "backlog", name)
	}
	path = filepath.Clean(path)

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", fmt.Errorf("no se pudo crear la carpeta de %s: %w", path, err)
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return "", fmt.Errorf("no se pudo guardar el reporte en %s: %w", path, err)
	}
	return path, nil
}

// ModuleSummary renders "Pagos 24 · Facturacion 9".
func ModuleSummary(counts []ModuleCount) string {
	parts := make([]string, 0, len(counts))
	for _, c := range counts {
		name := c.Module
		if name == "" {
			name = "(sin módulo)"
		}
		parts = append(parts, fmt.Sprintf("%s %d", name, c.Count))
	}
	return strings.Join(parts, " · ")
}
