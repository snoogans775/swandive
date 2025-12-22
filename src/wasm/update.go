package main

type UpdateType int

const (
	UpdateAlbum UpdateType = iota
	UpdatePlayback
)

type UIUpdate struct {
	Type UpdateType
}
