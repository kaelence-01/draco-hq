package cmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/kaelence-01/draco-hq/internal/build"
	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/installer"
	"github.com/kaelence-01/draco-hq/internal/updater"
)

func init() {
	register(&command{run: runUpdate, help: updateHelp, summary: "Actualiza draco a la última versión publicada"}, "update")
}

// newUpdater, executable and installDir are replaced in tests, which use a
// fake GitHub and temporary folders.
var (
	newUpdater = updater.New
	executable = installer.Executable
	installDir = installer.Dir
)

func notifier(c *updater.Client) *updater.Notifier {
	path, _ := config.ConfigPath()
	return &updater.Notifier{Client: c, Cache: filepath.Join(filepath.Dir(path), "version_check.json"), Now: time.Now}
}

func runUpdate(a *App, args []string) error {
	f, err := parseFlags(args, []flag{{long: "--check"}})
	if err != nil {
		return usageError("%v", err)
	}
	if len(f.args) > 0 {
		return usageError("argumento inesperado: %s", f.args[0])
	}
	current := build.Version
	a.printf("%s %s\n", a.style(bold, "Versión actual:"), a.style(cyan, current))

	ctx, cancel := a.context()
	defer cancel()
	c := newUpdater()
	latest, err := c.Latest(ctx)
	if errors.Is(err, updater.ErrNoReleases) {
		a.warn(fmt.Sprintf("%s en %s", err, updater.Repo))
		return nil
	}
	if err != nil {
		return fmt.Errorf("no se pudo consultar la última versión: %w", err)
	}
	notifier(c).Remember(latest)

	if !updater.IsNewer(latest, current) {
		if current == "dev" {
			a.warn("versión de desarrollo (go run o compilada a mano): no se actualiza sola. La última publicada es " + latest)
			return nil
		}
		a.ok("Ya tienes la versión más reciente (" + latest + ")")
		return nil
	}
	a.printf("%s %s\n\n", a.style(bold, "Nueva versión:"), a.style(green+bold, latest))
	if f.has("--check") {
		a.printf("Actualiza con: draco update\n")
		return nil
	}

	exe, err := executable()
	if err != nil {
		return fmt.Errorf("no se pudo ubicar el ejecutable: %w", err)
	}
	a.step(1, 2, "Descargando "+latest+" y verificando su SHA-256...")
	data, err := c.Fetch(ctx, latest)
	if err != nil {
		return err
	}
	a.step(2, 2, "Reemplazando "+exe+"...")
	if err := updater.Replace(exe, data); err != nil {
		return err
	}
	a.ok(a.style(green+bold, "Actualizado a "+latest))
	return nil
}

func updateHelp(a *App) {
	a.title("draco update", "Actualiza draco a la última versión publicada")

	a.section("Uso:")
	a.printf("  draco update [--check]\n\n")

	a.section("Flags:")
	a.printf("      --check   Solo consulta si hay una versión nueva\n")
	a.printf("  -h, --help    Esta ayuda\n\n")

	a.printf("  Descarga el binario de tu sistema desde los releases de %s y lo\n", updater.Repo)
	a.printf("  instala solo si su SHA-256 coincide con el checksums.txt del mismo release.\n")
	a.printf("  Una versión de desarrollo (go run) nunca se reemplaza.\n\n")

	a.section("Aviso de versión nueva:")
	a.printf("  Al usar draco en una terminal, avisa si hay una versión nueva. Consulta GitHub\n")
	a.printf("  como mucho una vez al día y nunca en pipes ni en CI; DRACO_NO_UPDATE_CHECK=1\n")
	a.printf("  lo apaga.\n")
}
