package config

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// Profile combines the platforms of one way of working: where the issues are,
// where the documents go and which template they use.
type Profile struct {
	Tracker  ProviderConfig `yaml:"tracker"`
	Docs     ProviderConfig `yaml:"docs"`
	Template string         `yaml:"template,omitempty"` // templates folder, one subfolder per kind; see TemplateDir
}

// ProviderConfig names an adapter and its settings. Jira and Confluence take
// the credentials and base_url of the config.
type ProviderConfig struct {
	Provider string `yaml:"provider"`        // jira | confluence | markdown
	Space    string `yaml:"space,omitempty"` // confluence: space key
	Path     string `yaml:"path,omitempty"`  // markdown: folder
}

// TemplateDir is the profile's folder for the template of kind (deploy, qa),
// laid out like the user's templates folder; empty without one. It overrides
// the repo's and the user's.
func (p *Profile) TemplateDir(kind string) string {
	if p.Template == "" {
		return ""
	}
	return filepath.Join(ExpandHome(p.Template), kind)
}

// LegacyProfile names the profile a config without profiles reads as.
const LegacyProfile = "atlassian"

// Profile returns the profile to use: name, else default_profile, else the
// only one. A config without profiles reads as LegacyProfile (Jira +
// Confluence with confluence_space_key) without rewriting the file.
func (c *Config) Profile(name string) (*Profile, string, error) {
	if len(c.Profiles) == 0 {
		if name != "" && name != LegacyProfile {
			return nil, "", fmt.Errorf("perfil '%s' no encontrado: la configuración no tiene perfiles", name)
		}
		return &Profile{
			Tracker: ProviderConfig{Provider: "jira"},
			Docs:    ProviderConfig{Provider: "confluence", Space: c.ConfluenceSpaceKey},
		}, LegacyProfile, nil
	}
	if name == "" {
		name = c.DefaultProfile
	}
	if name == "" && len(c.Profiles) == 1 {
		for only := range c.Profiles {
			name = only
		}
	}
	if name == "" {
		return nil, "", fmt.Errorf("hay varios perfiles y ninguno por defecto (%s): elige uno o define default_profile", strings.Join(c.ProfileNames(), ", "))
	}
	p, ok := c.Profiles[name]
	if !ok || p == nil {
		return nil, "", fmt.Errorf("perfil '%s' no encontrado. Perfiles: %s", name, strings.Join(c.ProfileNames(), ", "))
	}
	return p, name, nil
}

// ProfileNames returns the profile names, sorted.
func (c *Config) ProfileNames() []string { return sortedKeys(c.Profiles) }

// ProjectNames returns the project names, sorted.
func (c *Config) ProjectNames() []string { return sortedKeys(c.Projects) }

// SetProject adds or replaces a project.
func (c *Config) SetProject(name string, p *ProjectConfig) error {
	if err := ValidName("proyecto", name); err != nil {
		return err
	}
	if c.Projects == nil {
		c.Projects = map[string]*ProjectConfig{}
	}
	c.Projects[name] = p
	return nil
}

// RemoveProject deletes a project; when it was the default, there is no
// default afterwards.
func (c *Config) RemoveProject(name string) error {
	if _, ok := c.Projects[name]; !ok {
		return fmt.Errorf("proyecto '%s' no encontrado", name)
	}
	delete(c.Projects, name)
	if c.DefaultProject == name {
		c.DefaultProject = ""
	}
	return nil
}

// SetDefaultProject makes an existing project the default.
func (c *Config) SetDefaultProject(name string) error {
	if _, ok := c.Projects[name]; !ok {
		return fmt.Errorf("proyecto '%s' no encontrado", name)
	}
	c.DefaultProject = name
	return nil
}

// SetProfile adds or replaces a profile.
func (c *Config) SetProfile(name string, p *Profile) error {
	if err := ValidName("perfil", name); err != nil {
		return err
	}
	if c.Profiles == nil {
		c.Profiles = map[string]*Profile{}
	}
	c.Profiles[name] = p
	return nil
}

// KeepImplicitProfile saves the implicit profile, as the default, before the
// first profile named name is added, so the commands that used it keep
// working. It returns the profile it saved, or nil when there was nothing to
// keep.
func (c *Config) KeepImplicitProfile(name string) (*Profile, error) {
	if len(c.Profiles) > 0 || name == LegacyProfile {
		return nil, nil
	}
	implicit, _, err := c.Profile("")
	if err != nil {
		return nil, err
	}
	if err := c.SetProfile(LegacyProfile, implicit); err != nil {
		return nil, err
	}
	c.DefaultProfile = LegacyProfile
	return implicit, nil
}

// RemoveProfile deletes a profile; when it was the default, there is no
// default afterwards.
func (c *Config) RemoveProfile(name string) error {
	if _, ok := c.Profiles[name]; !ok {
		return fmt.Errorf("perfil '%s' no encontrado", name)
	}
	delete(c.Profiles, name)
	if c.DefaultProfile == name {
		c.DefaultProfile = ""
	}
	return nil
}

// SetDefaultProfile makes an existing profile the default.
func (c *Config) SetDefaultProfile(name string) error {
	if _, ok := c.Profiles[name]; !ok {
		return fmt.Errorf("perfil '%s' no encontrado", name)
	}
	c.DefaultProfile = name
	return nil
}

// ValidName checks a project or profile name; kind names it in the error.
func ValidName(kind, name string) error {
	if strings.TrimSpace(name) == "" || strings.ContainsAny(name, " \t\n/") {
		return fmt.Errorf("nombre de %s inválido %q: sin espacios ni /", kind, name)
	}
	return nil
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
