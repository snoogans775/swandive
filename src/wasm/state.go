package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"syscall/js"
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
	AudioPlayer           AudioSource
	Updates               chan UIUpdate
}

// State methods
func (s *State) SetCurrentTrackByButtonIndex(trackIndex int) error {
	bankedValue := (s.CurrentAlbumBankIndex * nTrackButtons) + trackIndex
	slog.Info("Setting Track index based on bank and track button", "bank_index", s.CurrentAlbumBankIndex, "track_button_index", trackIndex)
	s.CurrentTrackIndex = bankedValue % len(s.Albums[s.CurrentAlbumIndex].Tracks)

	return nil
}

func (s *State) UpdatePlaybackOffsetByPianoKeyButtonIndex(index int) error {
	trackDuration := s.AudioPlayer.Get("duration").Int()
	s.AudioPlayer.Set("currentTime", index*pianoKeyOffsetMultiplier%trackDuration)
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

func (s *State) Init(data []byte) error {
	err := json.Unmarshal(data, &s.Albums)
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

func (s *State) InitAudio(audioCtor js.Value) error {
	if !audioCtor.IsUndefined() {
		s.AudioPlayer = audioCtor.New()
		s.AudioPlayer.Set("preload", "true")
		slog.Info("Audio player created successfully")
	} else {
		return fmt.Errorf("Browser does not support HTML5 Audio")
	}
	return nil
}

func (s *State) RegisterUICallbacks() error {
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
		slog.Info("Button was clicked", "button_data_id", idStr)
		go func() {
			idValInt, _ := strconv.Atoi(idStr)
			err := s.SetCurrentTrackByButtonIndex(idValInt)
			if err != nil {
				slog.Warn("Failed to set the current track", "button_index", idValInt)
			}

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

	// Read piano key button events
	pianoKeyButtons := document.Call("getElementsByClassName", pianoKeyButtonClassName)
	if !pianoKeyButtons.Truthy() {
		return fmt.Errorf("Could not retrieve elements with class name: %s", pianoKeyButtonClassName)
	}
	onClickPianoKeyButton := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		idStr := this.Get("dataset").Get(pianoKeyButtonIndexDataId).String()
		go func() {
			idValInt, _ := strconv.Atoi(idStr)
			err := s.UpdatePlaybackOffsetByPianoKeyButtonIndex(idValInt)
			if err != nil {
				slog.Warn("Failed to set the current piano key", "button_index", idValInt)
			}
		}()
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
	for i := 0; i < pianoKeyButtons.Get("length").Int(); i++ {
		pianoKeyButtons.Call("item", i).Call("addEventListener", "click", onClickPianoKeyButton)
	}
	bankSelectButton.Call("addEventListener", "click", onClickBankSelectButton)
	btnInc.Call("addEventListener", "click", onClickAlbumIncrementButton)
	btnDec.Call("addEventListener", "click", onClickAlbumDecrementButton)
	return nil
}

func (s *State) RegisterUIListeners() error {
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
