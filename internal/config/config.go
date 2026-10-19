package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProjectConfig holds the local paths and repo names for a project.
type ProjectConfig struct {
	BackendPath        string `yaml:"backend_path,omitempty"`
	BackendRepo        string `yaml:"backend_repo,omitempty"`
	FrontendPath       string `yaml:"frontend_path,omitempty"`
	FrontendRepo       string `yaml:"frontend_repo,omitempty"`
	VCSHost            string `yaml:"vcs_host,omitempty"`
	VCSOrg             string `yaml:"vcs_org,omitempty"`
	ConfluenceSpaceKey string `yaml:"confluence_space_key,omitempty"`

	// DeployChecklist overrides the global one for this project. The steps are
	// stack-specific (a Laravel backend and a Node one need different
	// commands), which is why the per-project layer exists.
	DeployChecklist []string `yaml:"deploy_checklist,omitempty"`

	// Backlog tunes `draco backlog scan` per repo of the project. Where the
	// classes and tests live is a convention of each stack, so it is config.
	Backlog *BacklogConfig `yaml:"backlog,omitempty"`
}

// BacklogConfig holds the `draco backlog scan` settings of a project's repos.
type BacklogConfig struct {
	Backend  *BacklogRepoConfig `yaml:"backend,omitempty"`
	Frontend *BacklogRepoConfig `yaml:"frontend,omitempty"`
}

// BacklogRepoConfig overrides the scan defaults for one repo. Empty fields
// keep the generic default of internal/backlog.
type BacklogRepoConfig struct {
	Since       string   `yaml:"since,omitempty"`
	Extensions  []string `yaml:"extensions,omitempty"`
	Exclude     []string `yaml:"exclude,omitempty"`
	MaxLines    int      `yaml:"max_lines,omitempty"`
	MinCommits  int      `yaml:"min_commits,omitempty"`
	FixKeywords []string `yaml:"fix_keywords,omitempty"`
	ClassGlobs  []string `yaml:"class_globs,omitempty"`
	TestsDir    string   `yaml:"tests_dir,omitempty"`
	TestGlobs   []string `yaml:"test_globs,omitempty"`
	Modules     []string `yaml:"modules,omitempty"`
}

// BacklogFor returns the backlog settings of the given repo ("backend" or
// "frontend"), or nil when the project does not configure it.
func (p *ProjectConfig) BacklogFor(repo string) *BacklogRepoConfig {
	if p == nil || p.Backlog == nil {
		return nil
	}
	if repo == "frontend" {
		return p.Backlog.Frontend
	}
	return p.Backlog.Backend
}

// QAReportConfig holds the names printed in the header of the QA consolidated
// report. They are organization data, not code: a change of personnel must not
// require a new release, and the names must not be embedded in a binary that
// gets distributed. See docs/security/patrones-seguros.md (P-008).
type QAReportConfig struct {
	LiderTecnico string `yaml:"lider_tecnico,omitempty"`
	PMO          string `yaml:"pmo,omitempty"`
	QA           string `yaml:"qa,omitempty"`
}

// QAJiraConfig identifies, in the user's Jira, which tasks the QA consolidated
// report covers and which fields it reads. Every value belongs to one Jira
// instance — project key, workflow statuses, custom field ids — so none of
// them is compiled in. See docs/security/patrones-seguros.md (P-008).
type QAJiraConfig struct {
	Project string `yaml:"project,omitempty"`

	// Statuses are the workflow statuses (ids or names) of a task that has
	// reached QA or gone past it: testing, in review, done, released.
	Statuses []string `yaml:"statuses,omitempty"`

	// TestingStatus is the status whose transition opens the Kanban window.
	// Empty means DefaultTestingStatus.
	TestingStatus string `yaml:"testing_status,omitempty"`

	Fields QAJiraFields `yaml:"fields,omitempty"`
}

// QAJiraFields are the custom field ids (customfield_NNNNN) the report reads.
type QAJiraFields struct {
	CodingErrors string `yaml:"coding_errors,omitempty"` // number: errors found in review
	DevReturns   string `yaml:"dev_returns,omitempty"`   // number: times sent back to dev
	DevStatus    string `yaml:"dev_status,omitempty"`    // Development panel: PR state
}

// DefaultTestingStatus is the TestingStatus used when the config leaves it empty.
const DefaultTestingStatus = "Testing"

var customFieldRe = regexp.MustCompile(`^customfield_\d+$`)

// Validate reports every missing or malformed value at once, so a user fixing
// config.yaml does not have to discover them one run at a time.
func (q *QAJiraConfig) Validate() error {
	if q == nil {
		return fmt.Errorf("falta la sección 'qa_jira' en %s", configPathHint)
	}
	var problems []string
	if strings.TrimSpace(q.Project) == "" {
		problems = append(problems, "qa_jira.project: clave del proyecto de Jira (ej: APP)")
	}
	if len(q.Statuses) == 0 {
		problems = append(problems, "qa_jira.statuses: estados (id o nombre) de una tarea que ya llegó a QA")
	}
	for name, v := range map[string]string{
		"coding_errors": q.Fields.CodingErrors,
		"dev_returns":   q.Fields.DevReturns,
		"dev_status":    q.Fields.DevStatus,
	} {
		if !customFieldRe.MatchString(v) {
			problems = append(problems, fmt.Sprintf("qa_jira.fields.%s: id del campo con la forma customfield_12345 (actual: %q)", name, v))
		}
	}
	if len(problems) == 0 {
		return nil
	}
	sort.Strings(problems)
	return fmt.Errorf("configuración de QA incompleta en %s:\n  - %s", configPathHint, strings.Join(problems, "\n  - "))
}

// Config holds all configuration needed by the CLI.
// Priority: env vars > ~/.config/draco/config.yaml
type Config struct {
	AtlassianEmail     string                    `yaml:"atlassian_email"`
	AtlassianToken     string                    `yaml:"atlassian_token"`
	BaseURL            string                    `yaml:"base_url"`
	QAEmail            string                    `yaml:"qa_email,omitempty"`
	QAReport           *QAReportConfig           `yaml:"qa_report,omitempty"`
	QAJira             *QAJiraConfig             `yaml:"qa_jira,omitempty"`
	ConfluenceSpaceKey string                    `yaml:"confluence_space_key,omitempty"`
	DefaultProject     string                    `yaml:"default_project,omitempty"`
	Projects           map[string]*ProjectConfig `yaml:"projects,omitempty"`

	// DeployChecklist are the steps of the "A considerar" section of a new
	// deploy document. A project's own list wins over this one; when both are
	// empty a generic built-in default is used. Steps tied to one stack
	// ("php artisan migrate") belong here, not compiled into the binary that
	// other teams run. See docs/security/patrones-seguros.md (P-008).
	DeployChecklist []string `yaml:"deploy_checklist,omitempty"`
}

// ResolveDeployChecklist returns the checklist for the given project, applying
// the layering: project list > global list > nil (the document builder then
// falls back to its generic default).
func (c *Config) ResolveDeployChecklist(proj *ProjectConfig) []string {
	if proj != nil && len(proj.DeployChecklist) > 0 {
		return proj.DeployChecklist
	}
	return c.DeployChecklist
}

// Load loads config following priority: env vars > config file, and requires
// the Atlassian credentials. Commands that never call Atlassian (they only read
// local repos) use LoadLocal instead, so they work before `draco init`.
func Load() (*Config, error) {
	cfg, err := LoadLocal()
	if err != nil {
		return nil, err
	}

	if !cfg.HasAtlassianCredentials() {
		return nil, fmt.Errorf("configuración incompleta. Corre: draco init")
	}

	return cfg, nil
}

// LoadLocal loads config with the same priority as Load (env vars > config
// file) without requiring the Atlassian credentials. A missing config file is
// not an error: env vars and each command's defaults still apply. A file that
// exists but cannot be read or parsed is, so a typo in config.yaml is reported
// instead of being silently ignored.
func LoadLocal() (*Config, error) {
	cfg := configFromEnv()

	fileCfg, err := loadFromFile()
	if err != nil {
		return nil, err
	}
	if fileCfg != nil {
		mergeMissing(cfg, fileCfg)
	}

	return cfg, nil
}

// HasAtlassianCredentials reports whether email, token and base URL are all set.
func (c *Config) HasAtlassianCredentials() bool {
	return c.AtlassianEmail != "" && c.AtlassianToken != "" && c.BaseURL != ""
}

// configFromEnv builds the env var layer of the config.
func configFromEnv() *Config {
	return &Config{
		AtlassianEmail:     os.Getenv("ATLASSIAN_EMAIL"),
		AtlassianToken:     os.Getenv("ATLASSIAN_TOKEN"),
		BaseURL:            os.Getenv("ATLASSIAN_BASE_URL"),
		ConfluenceSpaceKey: os.Getenv("CONFLUENCE_SPACE_KEY"),
	}
}

// mergeMissing fills every field still empty in cfg with the value from the
// file. Fields already set (by env vars) win.
func mergeMissing(cfg, fileCfg *Config) {
	if cfg.AtlassianEmail == "" {
		cfg.AtlassianEmail = fileCfg.AtlassianEmail
	}
	if cfg.AtlassianToken == "" {
		cfg.AtlassianToken = fileCfg.AtlassianToken
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = fileCfg.BaseURL
	}
	if cfg.QAEmail == "" {
		cfg.QAEmail = fileCfg.QAEmail
	}
	if cfg.QAReport == nil {
		cfg.QAReport = fileCfg.QAReport
	}
	if cfg.QAJira == nil {
		cfg.QAJira = fileCfg.QAJira
	}
	if len(cfg.DeployChecklist) == 0 {
		cfg.DeployChecklist = fileCfg.DeployChecklist
	}
	if cfg.ConfluenceSpaceKey == "" {
		cfg.ConfluenceSpaceKey = fileCfg.ConfluenceSpaceKey
	}
	if cfg.DefaultProject == "" {
		cfg.DefaultProject = fileCfg.DefaultProject
	}
	if cfg.Projects == nil {
		cfg.Projects = fileCfg.Projects
	}
}

// GetProject resolves which project to use.
// Priority: explicit name > default_project > nil (no project configured).
func (c *Config) GetProject(name string) (*ProjectConfig, string, error) {
	if name != "" {
		proj, ok := c.Projects[name]
		if !ok {
			return nil, "", fmt.Errorf("proyecto '%s' no encontrado. Usa: draco project list", name)
		}
		return proj, name, nil
	}
	if c.DefaultProject != "" && c.Projects != nil {
		proj, ok := c.Projects[c.DefaultProject]
		if ok {
			return proj, c.DefaultProject, nil
		}
	}
	return nil, "", nil
}

// configPathHint is how messages refer to the config file. The real path
// comes from ConfigPath; this is only for humans.
const configPathHint = "~/.config/draco/config.yaml"

// ConfigPath returns the path to the config file.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "draco", "config.yaml"), nil
}

// loadFromFile reads and unmarshals the config file. It returns (nil, nil) when
// there is no file to read (no home directory, or the file does not exist yet).
func loadFromFile() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer %s: %w", path, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("%s tiene un formato inválido: %w", path, err)
	}
	return cfg, nil
}

// Save marshals and writes the config to disk with restricted permissions.
func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	// 0600 = solo el usuario puede leer/escribir
	return os.WriteFile(path, data, 0600)
}
