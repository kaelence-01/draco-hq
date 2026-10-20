package cmd

import (
	"context"

	"github.com/kaelence-01/draco-hq/internal/build"
	"github.com/kaelence-01/draco-hq/internal/tui"
)

// runTUI opens the visual mode; tests replace it so they never take over a
// terminal.
var runTUI = func(a *App) error {
	return tui.Run(context.Background(), tui.Deps{LoadConfig: loadConfig, Version: build.Version}, a.rawIn, a.Out)
}
