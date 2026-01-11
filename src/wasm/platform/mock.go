package platform

import "log/slog"

type MockPlatform struct {
	TrackButtonCallback      func(int)
	BankSelectButtonCallback func()
	PianoKeyButtonCallback   func(int)
	AlbumIncrementCallback   func()
	AlbumDecrementCallback   func()
	LastUIUpdate             UIUpdate
	LastAlbumTitle           string
	LastTrackTitle           string
}

func (p *MockPlatform) NewAudio() Audio {
	return &MockAudio{}
}

func (p *MockPlatform) RegisterUICallbacks(
	onClickTrackButton func(int),
	onClickBankSelectButton func(),
	onClickPianoKeyButton func(int),
	onClickAlbumIncrementButton func(),
	onClickAlbumDecrementButton func(),
) {
	p.TrackButtonCallback = onClickTrackButton
	p.BankSelectButtonCallback = onClickBankSelectButton
	p.PianoKeyButtonCallback = onClickPianoKeyButton
	p.AlbumIncrementCallback = onClickAlbumIncrementButton
	p.AlbumDecrementCallback = onClickAlbumDecrementButton
}

func (p *MockPlatform) UpdateUI(update UIUpdate, albumTitle string, trackTitle string) {
	slog.Info("Mock UpdateUI called", "update", update, "album", albumTitle, "track", trackTitle)
	p.LastUIUpdate = update
	p.LastAlbumTitle = albumTitle
	p.LastTrackTitle = trackTitle
}

type MockAudio struct {
	properties map[string]any
}

func (a *MockAudio) Set(prop string, value any) {
	if a.properties == nil {
		a.properties = make(map[string]any)
	}
	slog.Info("MockAudio Set", "prop", prop, "value", value)
	a.properties[prop] = value
}

func (a *MockAudio) Get(prop string) any {
	if a.properties == nil {
		return nil
	}
	return a.properties[prop]
}

func (a *MockAudio) GetDuration() float64 {
	if a.properties == nil {
		return 0
	}
	duration, ok := a.properties["duration"].(float64)
	if !ok {
		return 0
	}
	return duration
}

func (a *MockAudio) Call(m string, args ...any) any {
	// Not needed for mock
	return nil
}

func (a *MockAudio) Play() {
	slog.Info("MockAudio Play called")
}
