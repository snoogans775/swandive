package main

type UpdateType int

const (
	UpdateAlbum UpdateType = iota
	UpdateTrack
	UpdatePlayback
)

type UIUpdate struct {
	Type UpdateType
}
