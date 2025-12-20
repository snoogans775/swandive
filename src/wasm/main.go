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

//go:embed collection.json
var jsonData []byte

const (
	nTrackButtons          = 8
	trackButtonClassName   = "btn-hotspot"
	trackButtonIndexDataId = "track-button-index"
)

type Collection struct {
	Albums []Album `json:"albums"`
}

type Album struct {
	Title       string  `json:"title"`
	TotalTracks int32   `json:"totalTracks"`
	CoverUrl    string  `json:"coverUrl"`
	Tracks      []Track `json:"tracks"`
}

type Track struct {
	Title string `json:"title"`
}

type State struct {
	currentAlbum          Album
	currentTrack          Track
	currentAlbumBank      int
	currentAlbumBankTotal int
	isPlaying             bool
}

// State methods
func (s *State) SetCurrentTrackByButtonIndex(trackIndex int) error {
	bankedValue := (s.currentAlbumBank * nTrackButtons) + trackIndex
	if bankedValue < 0 || bankedValue >= len(s.currentAlbum.Tracks) {
		return fmt.Errorf("track index of %d was out of bounds for the current album", bankedValue)
	}

	s.currentTrack = s.currentAlbum.Tracks[bankedValue]

	return nil
}

func (s *State) IncrementBank() error {
	//TODO: implement
	return nil
}

func initData(c *Collection, s *State) error {
	err := json.Unmarshal(jsonData, &c)
	if err != nil {
		return err
	}
	if len(c.Albums) == 0 {
		return fmt.Errorf("Failed to load collection from json. No albums were parsed.")
	}
	if len(c.Albums[0].Tracks) == 0 {
		return fmt.Errorf("Failed to load initial track. No tracks found in default album")
	}
	s.currentAlbum = c.Albums[0]
	s.currentTrack = s.currentAlbum.Tracks[0]
	s.isPlaying = false

	return nil
}

func registerUiCallbacks(s *State) error {
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
		}()
		return nil
	})
	// Register the callbacks
	for i := 0; i < buttons.Get("length").Int(); i++ {
		buttons.Call("item", i).Call("addEventListener", "click", onClickButton)
	}
	return nil
}

func main() {
	done := make(chan struct{})
	var collection = Collection{}
	var state = State{}
	var err = initData(&collection, &state)
	if err != nil {
		slog.Error("Failed to initialize data", "err", err)
		os.Exit(1)
	}
	err = registerUiCallbacks(&state)
	if err != nil {
		slog.Error("Failed to bind callbacks to Javascript events")
		os.Exit(1)
	}
	slog.Info("Successfully initialized", "album_count", len(collection.Albums))

	<-done
}
