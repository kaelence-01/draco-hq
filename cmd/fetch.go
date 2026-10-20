package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kaelence-01/draco-hq/internal/fetch"
	"github.com/kaelence-01/draco-hq/internal/profile"
)

func init() {
	register(&command{run: runFetch, help: fetchHelp, summary: "Exporta a .txt el documento de una tarea"}, "fetch", "f")
}

var fetchFlags = []flag{
	{long: "--issue", short: "-i", value: true},
	{long: "--output", short: "-o", value: true},
	{long: "--space", short: "-s", value: true},
	{long: "--profile", value: true},
}

func runFetch(a *App, args []string) error {
	f, err := parseFlags(args, fetchFlags)
	if err != nil {
		return usageError("%v", err)
	}
	if len(f.args) > 0 {
		return usageError("argumento inesperado: %s", f.args[0])
	}
	if f.get("--issue") == "" {
		return usageError("falta -i/--issue, la clave de la tarea")
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}
	target, err := fetch.Resolve(cfg, fetch.Request{Issue: f.get("--issue"), Space: f.get("--space")})
	if err != nil {
		return err
	}
	bundle, err := openProfile(cfg, profile.Options{Name: f.get("--profile"), Scope: target.SpaceKey, AnyScope: true})
	if err != nil {
		return explainProfile(err)
	}

	ctx, cancel := a.context()
	defer cancel()
	s := fetch.NewSession(bundle.Docs, target)

	a.step(1, 3, "Buscando documentos que mencionen "+a.style(bold, target.Issue)+"...")
	if err := s.Search(ctx); err != nil {
		return err
	}
	page := &s.Pages[0]
	if len(s.Pages) == 1 {
		a.ok("Encontrado: " + page.Title)
	} else {
		a.printf("\n")
		for i, p := range s.Pages {
			a.printf("  %s %s\n", a.style(bold+yellow, fmt.Sprintf("[%d]", i+1)), p.Title)
		}
		answer := a.ask(fmt.Sprintf("\n  Documento a exportar (1-%d) [1]: ", len(s.Pages)))
		if answer != "" {
			n, err := strconv.Atoi(answer)
			if err != nil || n < 1 || n > len(s.Pages) {
				return fmt.Errorf("opción inválida: ingresa un número entre 1 y %d", len(s.Pages))
			}
			page = &s.Pages[n-1]
		}
		a.ok("Elegido: " + page.Title)
	}
	a.printf("\n")

	a.step(2, 3, "Descargando el contenido...")
	exp, err := s.Export(ctx, page)
	if err != nil {
		return err
	}
	a.ok(fmt.Sprintf("%d caracteres", len(exp.Text)))
	a.printf("\n")

	a.step(3, 3, "Guardando...")
	path := f.get("--output")
	if path == "" && isTerminal(a.Out) {
		path = saveDialog(exp.FileName)
	}
	if path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		path = filepath.Join(cwd, exp.FileName)
	}
	if err := exp.Save(path); err != nil {
		return err
	}
	a.ok("Exportado → " + a.style(cyan, path))
	a.printf("\n")
	return nil
}

func fetchHelp(a *App) {
	a.title("draco f, fetch", "Exporta a .txt el documento de una tarea")

	a.section("Uso:")
	a.printf("  draco f -i <TAREA> [flags]\n\n")
	a.printf("  Busca los documentos que mencionan la tarea, convierte el elegido a texto\n")
	a.printf("  plano (títulos, listas, tablas, tareas y paneles) y abre el diálogo del\n")
	a.printf("  sistema para elegir dónde guardarlo; sin diálogo, lo guarda en la carpeta actual.\n\n")

	a.section("Flags:")
	a.printf("  -i, --issue    Clave de la tarea                 %s\n", a.style(yellow, "(requerido)"))
	a.printf("  -o, --output   Archivo de salida                 (por defecto: OPS-7_Titulo.txt)\n")
	a.printf("  -s, --space    Space de Confluence o carpeta Markdown donde buscar\n")
	a.printf("      --profile  Perfil de la config\n")
	a.printf("  -h, --help     Esta ayuda\n\n")

	a.section("Ejemplos:")
	a.printf("  draco f -i OPS-7\n")
	a.printf("  draco f -i OPS-7 -o despliegue.txt\n")
	a.printf("  draco f -i OPS-7 --space DEV\n")
}
