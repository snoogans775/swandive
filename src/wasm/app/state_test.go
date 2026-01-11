package app

import (
	"testing"
	"swandive/platform"
)

const jsonData = `
[
    {
        "title": "Album 1",
        "totalTracks": 2,
        "coverUrl": "cover1.jpg",
        "tracks": [
            {
                "title": "Track 1",
                "audioUrl": "audio1.mp3"
            },
            {
                "title": "Track 2",
                "audioUrl": "audio2.mp3"
            }
        ]
    }
]
`

func TestStateInitialization(t *testing.T) {
	mockPlatform := &platform.MockPlatform{}
	state, err := NewState([]byte(jsonData), mockPlatform)
	if err != nil {
		t.Fatalf("Failed to initialize state: %v", err)
	}

	if state.CurrentAlbumIndex != 0 {
		t.Errorf("Expected current album index to be 0, got %d", state.CurrentAlbumIndex)
	}

	if state.CurrentTrackIndex != 0 {
		t.Errorf("Expected current track index to be 0, got %d", state.CurrentTrackIndex)
	}

	if mockPlatform.LastAlbumTitle != "Album 1" {
		t.Errorf("Expected last album title to be 'Album 1', got '%s'", mockPlatform.LastAlbumTitle)
	}
}

func TestOnTrackButtonPressed(t *testing.T) {
	mockPlatform := &platform.MockPlatform{}
	state, _ := NewState([]byte(jsonData), mockPlatform)

	state.OnTrackButtonPressed(1)

	if state.CurrentTrackIndex != 1 {
		t.Errorf("Expected current track index to be 1, got %d", state.CurrentTrackIndex)
	}

	if !state.IsPlaying {
		t.Error("Expected IsPlaying to be true")
	}
}

func TestOnPianoKeyPressed(t *testing.T) {
	mockPlatform := &platform.MockPlatform{}
	state, _ := NewState([]byte(jsonData), mockPlatform)
	state.AudioPlayer.Set("duration", 100.0)

	state.OnPianoKeyPressed(10)

	currentTime := state.AudioPlayer.Get("currentTime")
	if currentTime != 30 {
		t.Errorf("Expected currentTime to be 30, got %v", currentTime)
	}
}
