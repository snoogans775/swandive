# --- Makefile ---

.PHONY: build clean

GOOS_TARGET = js
GOARCH_TARGET = wasm
OUTPUT_PATH = ../../assets/swandive.wasm

build:
	GOOS=$(GOOS_TARGET) GOARCH=$(GOARCH_TARGET) go build -o $(OUTPUT_PATH) .

clean:
	rm -f $(OUTPUT_PATH)
