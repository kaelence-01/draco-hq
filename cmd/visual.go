package cmd

import (
	"context"

	"github.com/kaelence-01/draco-hq/internal/build"
	"github.com/kaelence-01/draco-hq/internal/config"
	"github.com/kaelence-01/draco-hq/internal/tui"
)

// runTUI opens the visual mode; tests replace it so they never take over a
// terminal.
var runTUI = func(a *App) error {
	d := tui.Deps{LoadConfig: loadConfig, Version: build.Version, LoadFile: config.LoadFile, SaveConfig: saveConfig, OpenProfile: openProfile, Scan: scanRepo, Now: now}
	return tui.Run(context.Background(), d, a.rawIn, a.Out)
}
