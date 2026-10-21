package cmd

import (
	"context"
	"os"
	"time"

	"github.com/kaelence-01/draco-hq/internal/build"
	"github.com/kaelence-01/draco-hq/internal/installer"
	"github.com/kaelence-01/draco-hq/internal/updater"
)

// Start is Run plus what only a person at a terminal gets: the offer to
// install a downloaded binary and the new-version notice.
func (a *App) Start(args []string) int {
	if exe, err := executable(); err == nil {
		updater.CleanOld(exe)
		if a.tty {
			if dir, err := installDir(); err == nil && !installer.IsInstalled(exe, dir) {
				if done, code := a.offerInstall(exe, dir, len(args) == 0); done {
					return code
				}
			}
		}
	}
	notice := a.updateNotice(args)
	code := a.Run(args)
	if v := notice(); v != "" {
		a.printf("\n%s %s (tienes %s) → draco update\n", a.style(yellow+bold, "Nueva versión disponible:"), v, build.Version)
	}
	return code
}

// offerInstall installs draco when the person agrees. Without arguments (a
// double click on Windows) it ends there; with a command, the command runs.
func (a *App) offerInstall(exe, dir string, alone bool) (done bool, code int) {
	a.title("draco", "no está instalado")
	a.printf("  Se copiará a %s y esa carpeta se agregará a tu PATH si no está.\n\n", dir)
	if !a.confirm("¿Instalar draco ahora?", true) {
		a.printf("No se instaló. Puedes copiarlo a mano a una carpeta de tu PATH.\n")
		return a.endInstall(alone, exitOK)
	}
	r, err := installer.Install(exe, dir)
	if err != nil {
		a.errorf("%v", err)
		return a.endInstall(alone, exitError)
	}
	a.ok("Instalado en " + r.Binary)
	switch {
	case r.PathFile != "":
		a.ok("Carpeta agregada al PATH en " + r.PathFile)
		a.printf("    Abre una terminal nueva o corre: source %s\n", r.PathFile)
	case r.PathChanged:
		a.ok("Carpeta agregada al PATH del usuario")
		a.printf("    Cierra y vuelve a abrir la terminal.\n")
	}
	a.printf("\n  Siguiente paso: %s\n\n", a.style(bold, "draco init"))
	return a.endInstall(alone, exitOK)
}

func (a *App) endInstall(alone bool, code int) (bool, int) {
	if alone {
		a.ask("\nPresiona Enter para cerrar...")
	}
	return alone, code
}

// updateNotice starts the new-version check and returns what to print after
// the command. The cached answer costs no network; a stale cache is refreshed
// in the background, waited for briefly only when nothing is cached.
func (a *App) updateNotice(args []string) func() string {
	none := func() string { return "" }
	if updatesOff() || !a.tty || !checksUpdates(args) {
		return none
	}
	n := notifier(newUpdater())
	cached := n.Cached(build.Version)
	if !n.Stale() {
		return func() string { return cached }
	}
	fresh := make(chan string, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		fresh <- n.Refresh(ctx, build.Version)
	}()
	return func() string {
		if cached != "" {
			return cached
		}
		select {
		case v := <-fresh:
			return v
		case <-time.After(1500 * time.Millisecond):
			return ""
		}
	}
}

// updatesOff is true for a development build or with DRACO_NO_UPDATE_CHECK.
func updatesOff() bool {
	return build.Version == "dev" || os.Getenv("DRACO_NO_UPDATE_CHECK") != ""
}

// latestVersion is the newer published version for the visual mode, or "",
// with the rules and the cache of the command mode's notice. The visual mode
// asks in the background, so it can wait for GitHub.
func latestVersion(ctx context.Context) string {
	if updatesOff() {
		return ""
	}
	n := notifier(newUpdater())
	if !n.Stale() {
		return n.Cached(build.Version)
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if v := n.Refresh(ctx, build.Version); v != "" {
		return v
	}
	return n.Cached(build.Version)
}

// checksUpdates leaves out the commands where a notice is noise or redundant.
func checksUpdates(args []string) bool {
	if len(args) == 0 || isHelpArg(args[0]) || wantsHelp(args[1:]) {
		return false
	}
	switch args[0] {
	case "update", "version", "--version", "-v":
		return false
	}
	return true
}
