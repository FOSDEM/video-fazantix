CONFIG=examples/fosdem.yaml

builddir:
	mkdir -p build

fazantix: builddir
	go build -o build/fazantix ./cmd/fazantix

fazantix-wayland: builddir
	go build -o build/fazantix -tags "wayland,vulkan" ./cmd/fazantix

run: fazantix
	./build/fazantix $(CONFIG)

run-wayland: fazantix-wayland
	./build/fazantix $(CONFIG)

run-cage: fazantix-wayland
	cage -- ./build/fazantix $(CONFIG)

lint:
	golangci-lint run
	golangci-lint fmt

clean:
	rm -rvf build

all: fazantix

build: fazantix

.PHONY: clean run lint fazantix fazantix-wayland builddir
