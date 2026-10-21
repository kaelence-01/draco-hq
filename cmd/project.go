package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/kaelence-01/draco-hq/internal/cli"
	"github.com/kaelence-01/draco-hq/internal/config"
)

func init() {
	register(&command{run: runProject, help: projectHelp, summary: "Lista, agrega, elige o elimina proyectos", subs: configSubs("proyecto", "proyectos")}, "project")
}

// configSubs are the subcommands project and profile share.
func configSubs(one, many string) []cli.Command {
	return []cli.Command{
		{Names: []string{"list", "ls"}, Summary: "Lista los " + many + " (* el de por defecto)"},
		{Names: []string{"add"}, Summary: "Agrega o reemplaza un " + one + ", con preguntas"},
		{Names: []string{"default"}, Summary: "Cambia el " + one + " por defecto"},
		{Names: []string{"remove", "rm"}, Summary: "Elimina un " + one},
	}
}

func runProject(a *App, args []string) error {
	return a.subcommand("project", map[string]func(*App, []string) error{
		"list":    runProjectList,
		"ls":      runProjectList,
		"add":     runProjectAdd,
		"default": runProjectDefault,
		"remove":  runProjectRemove,
		"rm":      runProjectRemove,
	}, projectHelp, args)
}

// subcommand runs args[0] from subs; without arguments it shows the help.
func (a *App) subcommand(name string, subs map[string]func(*App, []string) error, help func(*App), args []string) error {
	if len(args) == 0 {
		help(a)
		return nil
	}
	fn, ok := subs[args[0]]
	if !ok {
		return usageError("subcomando desconocido: %s %s", name, args[0])
	}
	return fn(a, args[1:])
}

// names reads between min and max positional arguments and no flags.
func names(args []string, min, max int, usage string) ([]string, error) {
	f, err := cli.Parse(args, nil)
	if err != nil {
		return nil, usageError("%v", err)
	}
	if len(f.Args) < min || len(f.Args) > max {
		return nil, usageError("uso: %s", usage)
	}
	return f.Args, nil
}

func saveConfig(cfg *config.Config) error {
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("no se pudo guardar la configuración: %w", err)
	}
	return nil
}

func runProjectList(a *App, args []string) error {
	if _, err := names(args, 0, 0, "draco project list"); err != nil {
		return err
	}
	cfg, err := config.LoadFile()
	if err != nil {
		return err
	}
	if len(cfg.Projects) == 0 {
		a.printf("No hay proyectos configurados. Agrega uno con: draco project add\n")
		return nil
	}
	w := tabwriter.NewWriter(a.Out, 0, 0, 3, ' ', 0)
	fmt.Fprintf(w, "  PROYECTO\tBACKEND\tFRONTEND\tSPACE\n")
	for _, n := range cfg.ProjectNames() {
		p := cfg.Projects[n]
		mark := " "
		if n == cfg.DefaultProject {
			mark = "*"
		}
		fmt.Fprintf(w, "%s %s\t%s\t%s\t%s\n", mark, n, orDash(p.BackendPath), orDash(p.FrontendPath), orDash(p.ConfluenceSpaceKey))
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if cfg.DefaultProject != "" {
		a.printf("\n* proyecto por defecto\n")
	}
	return nil
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func runProjectAdd(a *App, args []string) error {
	n, err := names(args, 0, 1, "draco project add [nombre]")
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
	a.title("draco project add", "Agregar un proyecto")
	return a.addProject(cfg, name)
}

// addProject asks for a project and saves it. Replacing a project keeps what
// the wizard does not ask (checklist, backlog).
func (a *App) addProject(cfg *config.Config, name string) error {
	var err error
	if name == "" {
		if name, err = a.askValue("Nombre del proyecto (ej: pagos)", "", true); err != nil {
			return err
		}
	}
	if err := config.ValidName("proyecto", name); err != nil {
		return err
	}
	p := &config.ProjectConfig{}
	if old, ok := cfg.Projects[name]; ok {
		if !a.confirm(fmt.Sprintf("  El proyecto '%s' ya existe. ¿Reemplazarlo?", name), false) {
			a.printf("Cancelado.\n")
			return nil
		}
		*p = *old
	}

	for _, side := range []struct {
		label      string
		path, repo *string
	}{{"backend", &p.BackendPath, &p.BackendRepo}, {"frontend", &p.FrontendPath, &p.FrontendRepo}} {
		if *side.path, err = a.askValue("Ruta local del repositorio "+side.label, *side.path, false); err != nil {
			return err
		}
		if *side.path == "" {
			continue
		}
		def := *side.repo
		if def == "" {
			def = config.DefaultRepoName(*side.path)
		}
		if *side.repo, err = a.askValue("Nombre del repositorio "+side.label+" en el VCS", def, true); err != nil {
			return err
		}
	}
	if err := p.Complete(); err != nil {
		a.warn(err.Error() + ": el proyecto no se guardó")
		return nil
	}
	if p.VCSHost, err = a.askValue("Host del VCS (ej: https://github.com)", p.VCSHost, false); err != nil {
		return err
	}
	if p.VCSOrg, err = a.askValue("Organización o workspace del VCS (ej: mi-org)", p.VCSOrg, false); err != nil {
		return err
	}
	if p.ConfluenceSpaceKey, err = a.askValue("Space de Confluence del proyecto (si no, el del perfil)", p.ConfluenceSpaceKey, false); err != nil {
		return err
	}

	if err := cfg.SetProject(name, p); err != nil {
		return err
	}
	if cfg.DefaultProject == "" || cfg.DefaultProject != name && a.confirm("  ¿Usarlo como proyecto por defecto?", true) {
		cfg.DefaultProject = name
	}
	if err := saveConfig(cfg); err != nil {
		return err
	}
	a.ok(fmt.Sprintf("Proyecto '%s' guardado", name))
	return nil
}

func runProjectDefault(a *App, args []string) error {
	n, err := names(args, 1, 1, "draco project default <nombre>")
	if err != nil {
		return err
	}
	cfg, err := config.LoadFile()
	if err != nil {
		return err
	}
	if err := cfg.SetDefaultProject(n[0]); err != nil {
		return fmt.Errorf("%w. Ver: draco project list", err)
	}
	if err := saveConfig(cfg); err != nil {
		return err
	}
	a.ok(fmt.Sprintf("Proyecto por defecto: '%s'", n[0]))
	return nil
}

func runProjectRemove(a *App, args []string) error {
	n, err := names(args, 1, 1, "draco project remove <nombre>")
	if err != nil {
		return err
	}
	cfg, err := config.LoadFile()
	if err != nil {
		return err
	}
	wasDefault := cfg.DefaultProject == n[0]
	if err := cfg.RemoveProject(n[0]); err != nil {
		return fmt.Errorf("%w. Ver: draco project list", err)
	}
	if !a.confirm(fmt.Sprintf("¿Eliminar el proyecto '%s'?", n[0]), false) {
		a.printf("Cancelado.\n")
		return nil
	}
	if err := saveConfig(cfg); err != nil {
		return err
	}
	a.ok(fmt.Sprintf("Proyecto '%s' eliminado", n[0]))
	if wasDefault {
		a.warn("era el proyecto por defecto: elige otro con draco project default <nombre>")
	}
	return nil
}

func projectHelp(a *App) {
	a.title("draco project", "Proyectos: rutas de los repos, VCS y space de cada uno")

	a.section("Uso:")
	a.printf("  draco project <subcomando>\n\n")

	a.section("Subcomandos:")
	a.printf("  list, ls            Lista los proyectos (* el de por defecto)\n")
	a.printf("  add [nombre]        Agrega o reemplaza un proyecto, con preguntas\n")
	a.printf("  default <nombre>    Cambia el proyecto por defecto\n")
	a.printf("  remove, rm <nombre> Elimina un proyecto\n\n")

	a.printf("  Se guarda en %s. Al reemplazar un proyecto se conserva lo que\n", configPathHint())
	a.printf("  el asistente no pregunta (deploy_checklist, backlog). Guardar reescribe el\n")
	a.printf("  archivo: los comentarios que tenga a mano se pierden.\n\n")

	a.section("Ejemplos:")
	a.printf("  draco project list\n")
	a.printf("  draco project add pagos\n")
	a.printf("  draco project default pagos\n")
	a.printf("  draco project remove facturacion\n")
}
