package platform

type Platform interface {
	// Audio methods
	NewAudio() Audio
	// UI methods
	RegisterUICallbacks(
		onClickTrackButton func(int),
		onClickBankSelectButton func(),
		onClickPianoKeyButton func(int),
		onClickAlbumIncrementButton func(),
		onClickAlbumDecrementButton func(),
	)
	UpdateUI(update UIUpdate, album string, track string)
}

type Audio interface {
	Set(prop string, value any)
	Get(prop string) any
	GetDuration() float64
	Call(m string, args ...any) any
	Play()
}

type UIUpdate struct {
	Type UpdateType
}

type UpdateType int

const (
	UpdateAlbum UpdateType = iota
	UpdateTrack
)
