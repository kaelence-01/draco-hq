package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"

	"gopkg.in/yaml.v3"
)

// gttDirs are where gtt keeps its config.yaml under ~/.config: the current
// folder first, then the one of versions before its rename.
var gttDirs = []string{"gtt", "deploy-doc"}

// GttConfig reads the config of gtt, the CLI draco comes from: its fields are
// a subset of draco's, so it reads as a draco config. It returns where it was;
// nil and "" when gtt has none.
func GttConfig() (*Config, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, "", nil
	}
	for _, dir := range gttDirs {
		path := filepath.Join(home, ".config", dir, "config.yaml")
		data, err := os.ReadFile(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, path, fmt.Errorf("no se pudo leer %s: %w", path, err)
		}
		cfg := &Config{}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, path, fmt.Errorf("%s tiene un formato inválido: %w", path, err)
		}
		if url, err := NormalizeBaseURL(cfg.BaseURL); err == nil {
			cfg.BaseURL = url
		}
		return cfg, path, nil
	}
	return nil, "", nil
}

// IsEmpty reports a config with nothing in it: importing over it loses nothing.
func (c *Config) IsEmpty() bool { return reflect.DeepEqual(*c, Config{}) }

// ImportNotes describes an imported config, never with its token, and says
// what it still lacks.
func ImportNotes(c *Config) []string {
	notes := []string{"Cuenta: " + orNone(c.AtlassianEmail) + " en " + orNone(c.BaseURL)}
	if c.ConfluenceSpaceKey != "" {
		notes = append(notes, "Space de Confluence: "+c.ConfluenceSpaceKey)
	}
	switch n := len(c.Projects); {
	case n == 0:
		notes = append(notes, "Sin proyectos: agrega uno para generar el documento")
	case c.DefaultProject != "":
		notes = append(notes, fmt.Sprintf("Proyectos: %d (por defecto: %s)", n, c.DefaultProject))
	default:
		notes = append(notes, fmt.Sprintf("Proyectos: %d", n))
	}
	if (c.QAEmail != "" || c.QAReport != nil) && c.QAJira == nil {
		notes = append(notes, "El consolidado QA necesita qa_jira (proyecto, estados y campos de Jira), que gtt no guardaba: complétalo en Configuración › QA")
	}
	return notes
}

func orNone(s string) string {
	if s == "" {
		return "(sin dato)"
	}
	return s
}
