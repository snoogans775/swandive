package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
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
	bankSelectButtonID     = "btn-bank-select"
	albumIncrementButtonID = "btn-album-inc"
	albumDecrementButtonID = "btn-album-dec"
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
	CurrentAlbumBankIndex int
	CurrentAlbumBankTotal int
	IsPlaying             bool
	AudioPlayer           js.Value
	Updates               chan UIUpdate
}

// State methods
func (s *State) SetCurrentTrackByButtonIndex(trackIndex int) error {
	bankedValue := (s.CurrentAlbumBankIndex * nTrackButtons) + trackIndex
	slog.Info("Setting Track index based on bank and track button", "bankIndex", s.CurrentAlbumBankIndex, "track_button_index", trackIndex)
	if bankedValue < 0 || bankedValue >= len(s.Albums[s.CurrentAlbumIndex].Tracks) {
		return fmt.Errorf("track index of %d was out of bounds for the current album", bankedValue)
	}

	s.CurrentTrackIndex = bankedValue

	return nil
}

func (s *State) IncrementBank() error {
	s.ComputeCurrentAlbumBankTotal()
	s.CurrentAlbumBankIndex = (s.CurrentAlbumBankIndex + 1) % s.CurrentAlbumBankTotal
	return nil
}

func (s *State) IncrementAlbum() {
	s.CurrentAlbumIndex = (s.CurrentAlbumIndex + 1) % len(s.Albums)
	s.CurrentTrackIndex = 0
	s.LoadTrack()
}

func (s *State) DecrementAlbum() {
	total := len(s.Albums)
	s.CurrentAlbumIndex = (s.CurrentAlbumIndex - 1 + total) % total
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
	s.Updates <- UIUpdate{Type: UpdateTrack}
}

func (s *State) ComputeCurrentAlbumBankTotal() error {
	nCurrentAlbumTracks := len(s.Albums[s.CurrentAlbumIndex].Tracks)
	s.CurrentAlbumBankTotal = int(math.Ceil(float64(nCurrentAlbumTracks) / float64(nTrackButtons)))
	return nil
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
	s.IsPlaying = false
	s.ComputeCurrentAlbumBankTotal()
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
	trackButtons := document.Call("getElementsByClassName", trackButtonClassName)
	if !trackButtons.Truthy() {
		return fmt.Errorf("Could not retrieve elements with class name: %s", trackButtonClassName)
	}
	onClickTrackButton := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
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
	bankSelectButton := document.Call("getElementById", bankSelectButtonID)
	if !bankSelectButton.Truthy() {
		return fmt.Errorf("Could not retrieve element with id: %s", bankSelectButtonID)
	}
	onClickBankSelectButton := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		slog.Info("Bank select button was clicked")
		s.IncrementBank()
		return nil
	})

	// Read album select button events
	btnInc := document.Call("getElementById", albumIncrementButtonID)
	if !btnInc.Truthy() {
		return fmt.Errorf("Could not retrieve element with id: %s", albumIncrementButtonID)
	}
	onClickAlbumIncrementButton := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		slog.Info("Button was clicked", "id", albumIncrementButtonID)
		s.IncrementAlbum()
		s.LoadTrack()
		return nil
	})

	btnDec := document.Call("getElementById", albumDecrementButtonID)
	if !btnDec.Truthy() {
		return fmt.Errorf("Could not retrieve element with id: %s", albumDecrementButtonID)
	}
	onClickAlbumDecrementButton := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		slog.Info("Button was clicked", "id", albumDecrementButtonID)
		s.DecrementAlbum()
		s.LoadTrack()
		return nil
	})

	// Register the callbacks
	for i := 0; i < trackButtons.Get("length").Int(); i++ {
		trackButtons.Call("item", i).Call("addEventListener", "click", onClickTrackButton)
	}
	bankSelectButton.Call("addEventListener", "click", onClickBankSelectButton)
	btnInc.Call("addEventListener", "click", onClickAlbumIncrementButton)
	btnDec.Call("addEventListener", "click", onClickAlbumDecrementButton)
	return nil
}

func registerUIListeners(s *State) error {
	document := js.Global().Get("document")
	if !document.Truthy() {
		return fmt.Errorf("Could not retrieve document")
	}
	for update := range s.Updates {
		switch update.Type {
		case UpdateAlbum, UpdateTrack:
			album := s.Albums[s.CurrentAlbumIndex]
			track := album.Tracks[s.CurrentTrackIndex]
			displayString := fmt.Sprintf("%s - %s", album.Title, track.Title)

			window := js.Global()
			lcdDisplay := document.Call("getElementById", "lcd-display")
			style := lcdDisplay.Get("style")

			computedStyle := window.Call("getComputedStyle", lcdDisplay)
			currentAnimation := computedStyle.Call("getPropertyValue", "animation")
			lcdDisplay.Set("textContent", displayString)
			style.Call("setProperty", "animation", "none")
			_ = lcdDisplay.Get("offsetHeight")
			style.Call("setProperty", "animation", currentAnimation)
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
	// This sends to the channel, but the loop above is now ready to receive
	slog.Info("Loading initial track metadata...")
	state.LoadTrack()

	// 5. Register User Interactions
	if err := registerUICallbacks(&state); err != nil {
		slog.Error("Failed to bind buttons", "err", err)
	}

	slog.Info("System online", "albums", len(state.Albums))

	select {}
}
