//go:build js && wasm

package main

import (
	_ "embed"
	"log/slog"
	"os"
	"swandive/app"
	"swandive/platform"
)

//go:embed albums.json
var jsonData []byte

func main() {
	browserPlatform := &platform.BrowserPlatform{}

	state, err := app.NewState(jsonData, browserPlatform)
	if err != nil {
		slog.Error("Initialization failed", "err", err)
		os.Exit(1)
	}

	browserPlatform.RegisterUICallbacks(
		state.OnTrackButtonPressed,
		state.OnBankSelectPressed,
		state.OnPianoKeyPressed,
		state.OnAlbumIncrementPressed,
		state.OnAlbumDecrementPressed,
	)

	slog.Info("System online")

	select {}
}
