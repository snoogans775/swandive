//go:build js && wasm

package platform

import (
	"fmt"
	"log/slog"
	"strconv"
	"syscall/js"
)

const (
	nTrackButtons             = 8
	trackButtonClassName      = "btn-hotspot"
	trackButtonIndexDataId    = "trackButtonIndex"
	pianoKeyButtonClassName   = "btn-piano-key"
	pianoKeyButtonIndexDataId = "pianoKeyButtonIndex"
	pianoKeyOffsetMultiplier  = 3
	bankSelectButtonID        = "btn-bank-select"
	albumIncrementButtonID    = "btn-album-inc"
	albumDecrementButtonID    = "btn-album-dec"
)

type BrowserPlatform struct{}

func (p *BrowserPlatform) NewAudio() Audio {
	audioCtor := js.Global().Get("Audio")
	if !audioCtor.IsUndefined() {
		audioPlayer := audioCtor.New()
		audioPlayer.Set("preload", "true")
		slog.Info("Audio player created successfully")
		return &BrowserAudio{player: audioPlayer}
	} else {
		slog.Error("Browser does not support HTML5 Audio")
		return nil
	}
}

func (p *BrowserPlatform) RegisterUICallbacks(
	onClickTrackButton func(int),
	onClickBankSelectButton func(),
	onClickPianoKeyButton func(int),
	onClickAlbumIncrementButton func(),
	onClickAlbumDecrementButton func(),
) {
	document := js.Global().Get("document")
	if !document.Truthy() {
		slog.Error("Could not retrieve document")
		return
	}

	// Track buttons
	trackButtons := document.Call("getElementsByClassName", trackButtonClassName)
	if !trackButtons.Truthy() {
		slog.Error("Could not retrieve elements with class name", "className", trackButtonClassName)
		return
	}
	cbTrack := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		idStr := this.Get("dataset").Get(trackButtonIndexDataId).String()
		idValInt, err := strconv.Atoi(idStr)
		if err != nil {
			slog.Warn("Failed to parse track button index", "error", err)
			return nil
		}
		go onClickTrackButton(idValInt)
		return nil
	})
	for i := 0; i < trackButtons.Get("length").Int(); i++ {
		trackButtons.Call("item", i).Call("addEventListener", "click", cbTrack)
	}

	// Bank select button
	bankSelectButton := document.Call("getElementById", bankSelectButtonID)
	if !bankSelectButton.Truthy() {
		slog.Error("Could not retrieve element with id", "id", bankSelectButtonID)
		return
	}
	cbBank := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go onClickBankSelectButton()
		return nil
	})
	bankSelectButton.Call("addEventListener", "click", cbBank)

	// Piano key buttons
	pianoKeyButtons := document.Call("getElementsByClassName", pianoKeyButtonClassName)
	if !pianoKeyButtons.Truthy() {
		slog.Error("Could not retrieve elements with class name", "className", pianoKeyButtonClassName)
		return
	}
	cbPiano := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		idStr := this.Get("dataset").Get(pianoKeyButtonIndexDataId).String()
		idValInt, err := strconv.Atoi(idStr)
		if err != nil {
			slog.Warn("Failed to parse piano key button index", "error", err)
			return nil
		}
		go onClickPianoKeyButton(idValInt)
		return nil
	})
	for i := 0; i < pianoKeyButtons.Get("length").Int(); i++ {
		pianoKeyButtons.Call("item", i).Call("addEventListener", "click", cbPiano)
	}

	// Album increment button
	btnInc := document.Call("getElementById", albumIncrementButtonID)
	if !btnInc.Truthy() {
		slog.Error("Could not retrieve element with id", "id", albumIncrementButtonID)
		return
	}
	cbInc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go onClickAlbumIncrementButton()
		return nil
	})
	btnInc.Call("addEventListener", "click", cbInc)

	// Album decrement button
	btnDec := document.Call("getElementById", albumDecrementButtonID)
	if !btnDec.Truthy() {
		slog.Error("Could not retrieve element with id", "id", albumDecrementButtonID)
		return
	}
	cbDec := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		go onClickAlbumDecrementButton()
		return nil
	})
	btnDec.Call("addEventListener", "click", cbDec)
}

func (p *BrowserPlatform) UpdateUI(update UIUpdate, albumTitle string, trackTitle string) {
	document := js.Global().Get("document")
	if !document.Truthy() {
		slog.Error("Could not retrieve document for UI update")
		return
	}
	switch update.Type {
	case UpdateAlbum, UpdateTrack:
		displayString := fmt.Sprintf("%s - %s", albumTitle, trackTitle)

		window := js.Global()
		lcdDisplay := document.Call("getElementById", "lcd-display")
		style := lcdDisplay.Get("style")

		computedStyle := window.Call("getComputedStyle", lcdDisplay)
		currentAnimation := computedStyle.Call("getPropertyValue", "animation")
		lcdDisplay.Set("textContent", displayString)
		style.Call("setProperty", "animation", "none")
		_ = lcdDisplay.Get("offsetHeight") // Trigger a reflow
		style.Call("setProperty", "animation", currentAnimation)
	}
}

type BrowserAudio struct {
	player js.Value
}

func (a *BrowserAudio) Set(prop string, value any) {
	a.player.Set(prop, value)
}

func (a *BrowserAudio) Get(prop string) any {
	return a.player.Get(prop)
}

func (a *BrowserAudio) GetDuration() float64 {
	return a.player.Get("duration").Float()
}

func (a *BrowserAudio) Call(m string, args ...any) any {
	return a.player.Call(m, args...)
}

func (a *BrowserAudio) Play() {
	promise := a.player.Call("play")
	promise.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) any {
		slog.Warn("Playback blocked: Waiting for user gesture")
		return nil
	}))
}
