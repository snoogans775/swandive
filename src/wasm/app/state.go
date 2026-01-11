package app

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"swandive/platform"
)

const (
	nTrackButtons            = 8
	pianoKeyOffsetMultiplier = 3
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
	AudioPlayer           platform.Audio
	Platform              platform.Platform
}

func NewState(data []byte, p platform.Platform) (*State, error) {
	s := &State{
		Platform: p,
	}

	if err := json.Unmarshal(data, &s.Albums); err != nil {
		return nil, err
	}
	if len(s.Albums) == 0 {
		return nil, fmt.Errorf("failed to load collection from json: no albums parsed")
	}
	if len(s.Albums[0].Tracks) == 0 {
		return nil, fmt.Errorf("failed to load initial track: no tracks in default album")
	}

	s.AudioPlayer = p.NewAudio()
	if s.AudioPlayer == nil {
		return nil, fmt.Errorf("failed to create audio player")
	}

	s.ComputeCurrentAlbumBankTotal()
	s.UpdateUI(platform.UpdateAlbum)
	return s, nil
}

// Event Handlers
func (s *State) OnTrackButtonPressed(trackIndex int) {
	bankedValue := (s.CurrentAlbumBankIndex * nTrackButtons) + trackIndex
	slog.Info("Setting Track index", "bank", s.CurrentAlbumBankIndex, "button", trackIndex)
	s.CurrentTrackIndex = bankedValue % len(s.Albums[s.CurrentAlbumIndex].Tracks)
	s.IsPlaying = true
	s.LoadTrack()
}

func (s *State) OnPianoKeyPressed(keyIndex int) {
	duration := s.AudioPlayer.GetDuration()
	if duration == 0 {
		slog.Warn("Could not get track duration or duration is zero")
		return
	}
	s.AudioPlayer.Set("currentTime", int(float64(keyIndex*pianoKeyOffsetMultiplier))%int(duration))
}

func (s *State) OnBankSelectPressed() {
	s.ComputeCurrentAlbumBankTotal()
	s.CurrentAlbumBankIndex = (s.CurrentAlbumBankIndex + 1) % s.CurrentAlbumBankTotal
}

func (s *State) OnAlbumIncrementPressed() {
	s.CurrentAlbumIndex = (s.CurrentAlbumIndex + 1) % len(s.Albums)
	s.CurrentTrackIndex = 0
	s.LoadTrack()
}

func (s *State) OnAlbumDecrementPressed() {
	total := len(s.Albums)
	s.CurrentAlbumIndex = (s.CurrentAlbumIndex - 1 + total) % total
	s.CurrentTrackIndex = 0
	s.LoadTrack()
}

// Private methods
func (s *State) LoadTrack() {
	track := s.Albums[s.CurrentAlbumIndex].Tracks[s.CurrentTrackIndex]
	s.AudioPlayer.Set("src", track.AudioUrl)

	if s.IsPlaying {
		s.AudioPlayer.Play()
	}
	s.UpdateUI(platform.UpdateTrack)
}

func (s *State) ComputeCurrentAlbumBankTotal() {
	nCurrentAlbumTracks := len(s.Albums[s.CurrentAlbumIndex].Tracks)
	s.CurrentAlbumBankTotal = int(math.Ceil(float64(nCurrentAlbumTracks) / float64(nTrackButtons)))
}

func (s *State) UpdateUI(updateType platform.UpdateType) {
	album := s.Albums[s.CurrentAlbumIndex]
	track := album.Tracks[s.CurrentTrackIndex]
	s.Platform.UpdateUI(platform.UIUpdate{Type: updateType}, album.Title, track.Title)
}
