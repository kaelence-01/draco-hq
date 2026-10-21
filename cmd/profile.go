package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/kaelence-01/draco-hq/internal/config"
)

func init() {
	register(&command{run: runProfile, help: profileHelp, summary: "Lista, agrega, elige o elimina perfiles (tracker y destino)"}, "profile")
}

func runProfile(a *App, args []string) error {
	return a.subcommand("profile", map[string]func(*App, []string) error{
		"list":    runProfileList,
		"ls":      runProfileList,
		"add":     runProfileAdd,
		"default": runProfileDefault,
		"remove":  runProfileRemove,
		"rm":      runProfileRemove,
	}, profileHelp, args)
}

func runProfileList(a *App, args []string) error {
	if _, err := names(args, 0, 0, "draco profile list"); err != nil {
		return err
	}
	cfg, err := config.LoadFile()
	if err != nil {
		return err
	}
	if len(cfg.Profiles) == 0 {
		p, name, _ := cfg.Profile("")
		a.printf("Sin perfiles: se usa el perfil implícito '%s' (%s).\n", name, describeProfile(p))
		a.printf("Agrega uno con: draco profile add\n")
		return nil
	}
	w := tabwriter.NewWriter(a.Out, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "  PERFIL\tTRACKER → DESTINO\tPLANTILLA\n")
	for _, n := range cfg.ProfileNames() {
		mark := " "
		if n == cfg.DefaultProfile {
			mark = "*"
		}
		p := cfg.Profiles[n]
		fmt.Fprintf(w, "%s %s\t%s\t%s\n", mark, n, describeProfile(p), orDash(p.Template))
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if cfg.DefaultProfile != "" {
		a.printf("\n* perfil por defecto\n")
	}
	return nil
}

func describeProfile(p *config.Profile) string {
	docs := p.Docs.Provider
	switch {
	case p.Docs.Provider == "confluence" && p.Docs.Space != "":
		docs += " (space " + p.Docs.Space + ")"
	case p.Docs.Provider == "markdown":
		docs += " (" + p.Docs.Path + ")"
	}
	return p.Tracker.Provider + " → " + docs
}

func runProfileAdd(a *App, args []string) error {
	n, err := names(args, 0, 1, "draco profile add [nombre]")
	if err != nil {
		return err
	}
	cfg, err := config.LoadFile()
	if err != nil {
		return err
	}
	name := ""
	if len(n) == 1 {
		name = n[0]
	}
	a.title("draco profile add", "Agregar un perfil")
	if name == "" {
		if name, err = a.askValue("Nombre del perfil (ej: equipo)", "", true); err != nil {
			return err
		}
	}
	if err := config.ValidName("perfil", name); err != nil {
		return err
	}
	legacy, err := cfg.KeepImplicitProfile(name)
	if err != nil {
		return err
	}
	if legacy != nil {
		a.printf("  También se guarda el perfil '%s' (%s), el que se usaba hasta ahora.\n\n", config.LegacyProfile, describeProfile(legacy))
	}

	p := &config.Profile{}
	if old, ok := cfg.Profiles[name]; ok {
		if !a.confirm(fmt.Sprintf("  El perfil '%s' ya existe. ¿Reemplazarlo?", name), false) {
			a.printf("Cancelado.\n")
			return nil
		}
		*p = *old
	}
	if err := a.askProfile(p, cfg.ConfluenceSpaceKey); err != nil {
		return err
	}
	if p.Template, err = a.askValue("Carpeta de plantillas del perfil (con deploy/ y qa/ dentro)", p.Template, false); err != nil {
		return err
	}
	if err := cfg.SetProfile(name, p); err != nil {
		return err
	}
	if cfg.DefaultProfile == "" || cfg.DefaultProfile != name && a.confirm("  ¿Usarlo como perfil por defecto?", false) {
		cfg.DefaultProfile = name
	}
	if err := saveConfig(cfg); err != nil {
		return err
	}
	a.ok(fmt.Sprintf("Perfil '%s' guardado: %s", name, describeProfile(p)))
	return nil
}

// askProfile asks where the documents go; the tracker is Jira, the only one
// available.
func (a *App) askProfile(p *config.Profile, space string) error {
	p.Tracker = config.ProviderConfig{Provider: "jira"}
	a.printf("  Tracker: jira\n")
	def := "1"
	if p.Docs.Provider == "markdown" {
		def = "2"
	}
	a.printf("  Destino de los documentos:\n    [1] Confluence\n    [2] Carpeta Markdown local\n")
	choice, err := a.askValue("Opción", def, true)
	if err != nil {
		return err
	}
	switch choice {
	case "1":
		if p.Docs.Provider == "confluence" && p.Docs.Space != "" {
			space = p.Docs.Space
		}
		p.Docs = config.ProviderConfig{Provider: "confluence"}
		p.Docs.Space, err = a.askValue("Space de Confluence (ej: DEV)", space, false)
	case "2":
		path := ""
		if p.Docs.Provider == "markdown" {
			path = p.Docs.Path
		}
		p.Docs = config.ProviderConfig{Provider: "markdown"}
		p.Docs.Path, err = a.askValue("Carpeta de los documentos (ej: ./docs/despliegues)", path, true)
	default:
		return fmt.Errorf("opción inválida: %s", choice)
	}
	return err
}

func runProfileDefault(a *App, args []string) error {
	n, err := names(args, 1, 1, "draco profile default <nombre>")
	if err != nil {
		return err
	}
	cfg, err := config.LoadFile()
	if err != nil {
		return err
	}
	if err := cfg.SetDefaultProfile(n[0]); err != nil {
		return fmt.Errorf("%w. Ver: draco profile list", err)
	}
	if err := saveConfig(cfg); err != nil {
		return err
	}
	a.ok(fmt.Sprintf("Perfil por defecto: '%s'", n[0]))
	return nil
}

func runProfileRemove(a *App, args []string) error {
	n, err := names(args, 1, 1, "draco profile remove <nombre>")
	if err != nil {
		return err
	}
	cfg, err := config.LoadFile()
	if err != nil {
		return err
	}
	wasDefault := cfg.DefaultProfile == n[0]
	if err := cfg.RemoveProfile(n[0]); err != nil {
		return fmt.Errorf("%w. Ver: draco profile list", err)
	}
	if !a.confirm(fmt.Sprintf("¿Eliminar el perfil '%s'?", n[0]), false) {
		a.printf("Cancelado.\n")
		return nil
	}
	if err := saveConfig(cfg); err != nil {
		return err
	}
	a.ok(fmt.Sprintf("Perfil '%s' eliminado", n[0]))
	if wasDefault && len(cfg.Profiles) > 1 {
		a.warn("era el perfil por defecto: elige otro con draco profile default <nombre>")
	}
	return nil
}

func profileHelp(a *App) {
	a.title("draco profile", "Perfiles: de dónde salen las tareas y a dónde van los documentos")

	a.section("Uso:")
	a.printf("  draco profile <subcomando>\n\n")

	a.section("Subcomandos:")
	a.printf("  list, ls            Lista los perfiles (* el de por defecto)\n")
	a.printf("  add [nombre]        Agrega o reemplaza un perfil, con preguntas\n")
	a.printf("  default <nombre>    Cambia el perfil por defecto\n")
	a.printf("  remove, rm <nombre> Elimina un perfil\n\n")

	a.section("Perfiles:")
	a.printf("  Cada perfil combina un tracker (hoy Jira), un destino (Confluence o una\n")
	a.printf("  carpeta Markdown) y, si quieres, una carpeta de plantillas. Los comandos\n")
	a.printf("  usan el de por defecto, u otro con --profile. Una config sin perfiles\n")
	a.printf("  funciona como el perfil '%s' (Jira y Confluence); al agregar el primero,\n", config.LegacyProfile)
	a.printf("  ese también se guarda para que nada cambie.\n\n")

	a.section("Ejemplos:")
	a.printf("  draco profile list\n")
	a.printf("  draco profile add externo     # documentos a una carpeta Markdown\n")
	a.printf("  draco profile default externo\n")
	a.printf("  draco g -i OPS-7 -b a1b2c3d --profile externo\n")
}
