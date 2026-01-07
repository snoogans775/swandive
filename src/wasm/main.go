package main

import (
	_ "embed"
	"log/slog"
	"os"
	"syscall/js"
)

//go:embed albums.json
var jsonData []byte

const (
	nTrackButtons             = 8
	trackButtonClassName      = "btn-hotspot"
	trackButtonIndexDataId    = "trackButtonIndex"
	pianoKeyButtonClassName   = "btn-piano-key"
	pianoKeyButtonIndexDataId = "pianoKeyButtonIndex"
	pianoKeyOffsetMultiplier  = 3
	bankSelectButtonID        = "btn-bank-select"
	albumIncrementButtonID    = "btn-album-inc"
	albumDecrementButtonID    = "btn-album-dec"
)

func main() {
	state := State{}

	// 1. Basic Data Setup (Creates the channel)
	// We do NOT send to the channel inside Init yet.
	if err := state.Init(jsonData); err != nil {
		slog.Error("Initialization failed", "err", err)
		os.Exit(1)
	}

	// 2. Start the UI Listener Goroutine IMMEDIATELY
	// This ensures the channel is being "drained" as soon as we start sending.
	go func() {
		slog.Info("Starting UI Listener Loop...")
		if err := state.RegisterUIListeners(); err != nil {
			slog.Error("UI Listener crashed", "err", err)
		}
	}()

	// 3. Initialize Audio
	audioCtor := js.Global().Get("Audio")
	if err := state.InitAudio(audioCtor); err != nil {
		slog.Error("Initialization of audio failed", "err", err)
		os.Exit(1)
	}

	// 4. Load the first track
	// This sends to the channel, but the loop above is now ready to receive
	slog.Info("Loading initial track metadata...")
	state.LoadTrack()

	// 5. Register User Interactions
	if err := state.RegisterUICallbacks(); err != nil {
		slog.Error("Failed to bind buttons", "err", err)
	}

	slog.Info("System online", "albums", len(state.Albums))

	select {}
}
