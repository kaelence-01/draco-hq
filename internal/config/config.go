package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
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

	// DeployChecklist overrides the global one: deploy steps depend on the stack.
	DeployChecklist []string `yaml:"deploy_checklist,omitempty"`

	// Backlog tunes `draco backlog scan` per repo of the project.
	Backlog *BacklogConfig `yaml:"backlog,omitempty"`
}

// BacklogConfig holds the `draco backlog scan` settings of a project's repos.
type BacklogConfig struct {
	Backend  *BacklogRepoConfig `yaml:"backend,omitempty"`
	Frontend *BacklogRepoConfig `yaml:"frontend,omitempty"`
}

// BacklogRepoConfig overrides the scan defaults for one repo; empty fields keep them.
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

// BacklogFor returns the settings of repo ("backend" or "frontend"), or nil.
func (p *ProjectConfig) BacklogFor(repo string) *BacklogRepoConfig {
	if p == nil || p.Backlog == nil {
		return nil
	}
	if repo == "frontend" {
		return p.Backlog.Frontend
	}
	return p.Backlog.Backend
}

// SetBacklogPatterns sets the class globs and modules of repo ("backend" or
// "frontend"), keeping its other settings; a repo or a backlog left with
// nothing is removed.
func (p *ProjectConfig) SetBacklogPatterns(repo string, classGlobs, modules []string) {
	if p.Backlog == nil {
		p.Backlog = &BacklogConfig{}
	}
	side := &p.Backlog.Backend
	if repo == "frontend" {
		side = &p.Backlog.Frontend
	}
	if *side == nil {
		*side = &BacklogRepoConfig{}
	}
	(*side).ClassGlobs, (*side).Modules = classGlobs, modules
	if reflect.DeepEqual(**side, BacklogRepoConfig{}) {
		*side = nil
	}
	if p.Backlog.Backend == nil && p.Backlog.Frontend == nil {
		p.Backlog = nil
	}
}

// QAReportConfig holds the names in the QA report header. Organization data,
// never compiled in (P-008).
type QAReportConfig struct {
	LiderTecnico string `yaml:"lider_tecnico,omitempty"`
	PMO          string `yaml:"pmo,omitempty"`
	QA           string `yaml:"qa,omitempty"`
}

// QAJiraConfig maps the QA report onto the user's Jira instance. Instance
// data, never compiled in (P-008).
type QAJiraConfig struct {
	Project string `yaml:"project,omitempty"`

	// Statuses (ids or names) of a task that reached QA or went past it.
	Statuses []string `yaml:"statuses,omitempty"`

	// TestingStatus opens the Kanban window; empty means DefaultTestingStatus.
	TestingStatus string `yaml:"testing_status,omitempty"`

	// SprintName is a template with {module} and {sprint}; empty means DefaultSprintName.
	SprintName string `yaml:"sprint_name,omitempty"`

	Fields QAJiraFields `yaml:"fields,omitempty"`

	// ReviewPrefixes start the summaries of QA review tasks: they are not rows
	// of the report, they are linked to the task they review. Empty means
	// DefaultReviewPrefixes.
	ReviewPrefixes []string `yaml:"review_prefixes,omitempty"`

	// ObservationMarker starts the comment text taken as an observation.
	// Empty means DefaultObservationMarker.
	ObservationMarker string `yaml:"observation_marker,omitempty"`
}

// DefaultReviewPrefixes is the ReviewPrefixes used when the config leaves it empty.
var DefaultReviewPrefixes = []string{"Revisión de Tarea", "Revision de Tarea", "QA"}

// DefaultObservationMarker is the ObservationMarker used when the config leaves it empty.
const DefaultObservationMarker = "Novedad::"

// Reviews returns ReviewPrefixes or its default.
func (q *QAJiraConfig) Reviews() []string {
	if len(q.ReviewPrefixes) == 0 {
		return DefaultReviewPrefixes
	}
	return q.ReviewPrefixes
}

// Marker returns ObservationMarker or its default.
func (q *QAJiraConfig) Marker() string {
	if strings.TrimSpace(q.ObservationMarker) == "" {
		return DefaultObservationMarker
	}
	return q.ObservationMarker
}

// QAJiraFields are the custom field ids (customfield_NNNNN) the report reads.
type QAJiraFields struct {
	CodingErrors string `yaml:"coding_errors,omitempty"` // number: errors found in review
	DevReturns   string `yaml:"dev_returns,omitempty"`   // number: times sent back to dev
	DevStatus    string `yaml:"dev_status,omitempty"`    // Development panel: PR state
}

// DefaultTestingStatus is the TestingStatus used when the config leaves it empty.
const DefaultTestingStatus = "Testing"

// DefaultSprintName is the SprintName used when the config leaves it empty.
const DefaultSprintName = "{module}_Sprint {sprint}"

// SprintNameFor renders SprintName for one module and sprint number.
func (q *QAJiraConfig) SprintNameFor(module string, sprint int) string {
	tmpl := q.SprintName
	if strings.TrimSpace(tmpl) == "" {
		tmpl = DefaultSprintName
	}
	return strings.NewReplacer("{module}", module, "{sprint}", strconv.Itoa(sprint)).Replace(tmpl)
}

// Empty reports whether nothing of qa_jira is set: QA is optional.
func (q *QAJiraConfig) Empty() bool {
	return q.Project == "" && len(q.Statuses) == 0 && q.TestingStatus == "" && q.SprintName == "" &&
		q.Fields == (QAJiraFields{}) && len(q.ReviewPrefixes) == 0 && q.ObservationMarker == ""
}

// ExpandHome replaces a leading ~ with the user's home folder: the shell does
// it for flags, but nothing does it for the paths in config.yaml.
func ExpandHome(path string) string {
	rest, ok := strings.CutPrefix(path, "~")
	if !ok || (rest != "" && rest[0] != '/' && rest[0] != filepath.Separator) {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, rest)
}

// SplitList reads a comma-separated list, without spaces or empty items.
func SplitList(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

var customFieldRe = regexp.MustCompile(`^customfield_\d+$`)

// Validate reports every missing or malformed value at once.
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
	// Without {sprint} every run would search the same sprint.
	if q.SprintName != "" && !strings.Contains(q.SprintName, "{sprint}") {
		problems = append(problems, fmt.Sprintf("qa_jira.sprint_name: debe incluir {sprint} (ej: %s)", DefaultSprintName))
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

// Config holds all configuration. Env vars win over the config file.
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

	// Profiles pick the platforms draco talks to; a config without them reads
	// as one Jira + Confluence profile (see Profile).
	DefaultProfile string              `yaml:"default_profile,omitempty"`
	Profiles       map[string]*Profile `yaml:"profiles,omitempty"`

	// DeployChecklist are the "A considerar" steps of a new deploy document.
	// Stack-specific, so config, never compiled in (P-008).
	DeployChecklist []string `yaml:"deploy_checklist,omitempty"`

	// Theme is the visual mode's theme (ansi, alto-contraste, mono); empty
	// is ansi.
	Theme string `yaml:"theme,omitempty"`
}

// ResolveDeployChecklist returns project > global > nil (the builder's default).
func (c *Config) ResolveDeployChecklist(proj *ProjectConfig) []string {
	if proj != nil && len(proj.DeployChecklist) > 0 {
		return proj.DeployChecklist
	}
	return c.DeployChecklist
}

// Load is LoadLocal plus the Atlassian credentials, which it requires.
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

// LoadLocal loads env vars over the config file without requiring credentials.
// A missing file is not an error; an unreadable or invalid one is.
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

// LoadFile reads only the config file, never the environment: what a command
// saves must not carry a token that came from ATLASSIAN_TOKEN. A missing file
// is an empty config.
func LoadFile() (*Config, error) {
	cfg, err := loadFromFile()
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = &Config{}
	}
	return cfg, nil
}

// HasAtlassianCredentials reports whether email, token and base URL are all set.
func (c *Config) HasAtlassianCredentials() bool {
	return c.AtlassianEmail != "" && c.AtlassianToken != "" && c.BaseURL != ""
}

// NormalizeBaseURL returns the site URL without spaces or a trailing slash;
// it must be https.
func NormalizeBaseURL(s string) (string, error) {
	u := strings.TrimRight(strings.TrimSpace(s), "/")
	if !strings.HasPrefix(u, "https://") {
		return "", errors.New("la URL debe empezar con https://")
	}
	return u, nil
}

func configFromEnv() *Config {
	return &Config{
		AtlassianEmail:     os.Getenv("ATLASSIAN_EMAIL"),
		AtlassianToken:     os.Getenv("ATLASSIAN_TOKEN"),
		BaseURL:            os.Getenv("ATLASSIAN_BASE_URL"),
		ConfluenceSpaceKey: os.Getenv("CONFLUENCE_SPACE_KEY"),
	}
}

// mergeMissing fills the fields still empty in cfg from the file.
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
	if cfg.DefaultProfile == "" {
		cfg.DefaultProfile = fileCfg.DefaultProfile
	}
	if cfg.Profiles == nil {
		cfg.Profiles = fileCfg.Profiles
	}
	if cfg.Theme == "" {
		cfg.Theme = fileCfg.Theme
	}
}

// GetProject resolves name > default_project > nil. An unknown name is an error.
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

// configPathHint is the config path as shown in messages; see ConfigPath.
const configPathHint = "~/.config/draco/config.yaml"

// TemplatesDir returns the folder of the user's document templates, one
// subfolder per kind (deploy, qa).
func TemplatesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "draco", "templates"), nil
}

// ConfigPath returns the path to the config file.
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "draco", "config.yaml"), nil
}

// loadFromFile returns (nil, nil) when there is no file to read.
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

	// 0600: the file holds the API token.
	return os.WriteFile(path, data, 0600)
}
