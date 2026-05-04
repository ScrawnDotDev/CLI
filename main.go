package main

import (
	"os"

	"github.com/ScrawnDotDev/scrawn-cli/internal/app"
	"github.com/ScrawnDotDev/scrawn-cli/internal/ui"
)

func main() {
	if err := app.Run(os.Args[1:]); err != nil {
		ui.RenderError(err)
		os.Exit(1)
	}
}
