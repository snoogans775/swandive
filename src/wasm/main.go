package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"syscall/js"
)

//go:embed albums.json
var jsonData []byte

const (
	nTrackButtons          = 8
	trackButtonClassName   = "btn-hotspot"
	trackButtonIndexDataId = "trackButtonIndex"
)

type Album struct {
	Title       string  `json:"title"`
	TotalTracks int32   `json:"totalTracks"`
	CoverUrl    string  `json:"coverUrl"`
	Tracks      []Track `json:"tracks"`
}

type Track struct {
	Title    string `json:"title"`
	AudioUrl string `json:"audioUrl"`
}

type State struct {
	Albums                []Album
	CurrentAlbumIndex     int
	CurrentTrackIndex     int
	CurrentAlbumBank      int
	CurrentAlbumBankTotal int
	IsPlaying             bool
	AudioPlayer           js.Value
	Updates               chan UIUpdate
}

// State methods
func (s *State) SetCurrentTrackByButtonIndex(trackIndex int) error {
	bankedValue := (s.CurrentAlbumBank * nTrackButtons) + trackIndex
	if bankedValue < 0 || bankedValue >= len(s.Albums[s.CurrentAlbumIndex].Tracks) {
		return fmt.Errorf("track index of %d was out of bounds for the current album", bankedValue)
	}

	s.CurrentTrackIndex = bankedValue

	return nil
}

func (s *State) IncrementBank() error {
	//TODO: implement
	return nil
}

func (s *State) IncrementAlbum() {
	s.CurrentAlbumIndex = (s.CurrentAlbumIndex + 1) % len(s.Albums)
	s.CurrentTrackIndex = 0
	s.LoadTrack()
}

func (s *State) LoadTrack() {
	track := s.Albums[s.CurrentAlbumIndex].Tracks[s.CurrentTrackIndex]
	s.AudioPlayer.Set("src", track.AudioUrl)

	if s.IsPlaying {
		promise := s.AudioPlayer.Call("play")

		// Handle the error you're seeing in Go
		promise.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) any {
			slog.Warn("Playback blocked: Waiting for user gesture")
			s.IsPlaying = false
			return nil
		}))
	}
	s.Updates <- UIUpdate{Type: UpdatePlayback}
}

// End State methods

func initData(s *State) error {
	err := json.Unmarshal(jsonData, &s.Albums)
	if err != nil {
		return err
	}
	if len(s.Albums) == 0 {
		return fmt.Errorf("Failed to load collection from json. No albums were parsed.")
	}
	if len(s.Albums[0].Tracks) == 0 {
		return fmt.Errorf("Failed to load initial track. No tracks found in default album")
	}
	s.CurrentAlbumIndex = 0
	s.CurrentTrackIndex = 0
	s.IsPlaying = true
	s.Updates = make(chan UIUpdate, 1)
	s.Updates <- UIUpdate{Type: UpdateAlbum}

	return nil
}

func initAudio(s *State) {
	audioCtor := js.Global().Get("Audio")
	if !audioCtor.IsUndefined() {
		s.AudioPlayer = audioCtor.New()
		slog.Info("Audio player created successfully")
	} else {
		slog.Error("Browser does not support HTML5 Audio")
	}
}

func registerUICallbacks(s *State) error {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return fmt.Errorf("Could not retrieve document")
	}

	// Read sound bank button events
	buttons := document.Call("getElementsByClassName", trackButtonClassName)
	if !buttons.Truthy() {
		return fmt.Errorf("Could not retrieve elements with class name: %s", trackButtonClassName)
	}
	onClickButton := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		idStr := this.Get("dataset").Get(trackButtonIndexDataId).String()
		slog.Info("Button was clicked", "track_button_id", idStr)
		go func() {
			idValInt, _ := strconv.Atoi(idStr)
			s.SetCurrentTrackByButtonIndex(idValInt - 1)

			s.IsPlaying = true
			s.LoadTrack()
		}()
		return nil
	})
	// Register the callbacks
	for i := 0; i < buttons.Get("length").Int(); i++ {
		buttons.Call("item", i).Call("addEventListener", "click", onClickButton)
	}
	return nil
}

func registerUIListeners(s *State) error {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return fmt.Errorf("Could not retrieve document")
	}
	for update := range s.Updates {
		switch update.Type {
		case UpdateAlbum:
			album := s.Albums[s.CurrentAlbumIndex]
			track := album.Tracks[s.CurrentTrackIndex]

			displayString := fmt.Sprintf("%s - %s", album.Title, track.Title)
			document.Call("getElementById", "lcd-display").Set("textContent", displayString)
		}
	}

	return nil
}

func main() {
	state := State{}

	// 1. Basic Data Setup (Creates the channel)
	// We do NOT send to the channel inside initData yet.
	if err := initData(&state); err != nil {
		slog.Error("Initialization failed", "err", err)
		os.Exit(1)
	}

	// 2. Start the UI Listener Goroutine IMMEDIATELY
	// This ensures the channel is being "drained" as soon as we start sending.
	go func() {
		slog.Info("Starting UI Listener Loop...")
		if err := registerUIListeners(&state); err != nil {
			slog.Error("UI Listener crashed", "err", err)
		}
	}()

	// 3. Initialize Audio
	initAudio(&state)

	// 4. Load the first track
	// This sends to the channel, but the loop above is now ready to receive!
	slog.Info("Loading initial track metadata...")
	state.LoadTrack()

	// 5. Register User Interactions
	if err := registerUICallbacks(&state); err != nil {
		slog.Error("Failed to bind buttons", "err", err)
	}

	slog.Info("System online", "albums", len(state.Albums))

	// Keep the WASM process alive
	select {}
}
