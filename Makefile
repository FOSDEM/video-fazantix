CONFIG=examples/imagesource.yaml

fazantix: builddir
	go build -o build/fazantix ./cmd/fazantix

builddir:
	mkdir -p build

fazantix-wayland: builddir
	go build -o build/fazantix -tags "wayland,vulkan" ./cmd/fazantix

fazantix-window: builddir
	go build -o build/fazantix-window ./cmd/fazantix-window

run: fazantix
	./build/fazantix $(CONFIG)

run-wayland: fazantix-wayland
	./build/fazantix $(CONFIG)

run-cage: fazantix-wayland
	cage -- ./build/fazantix $(CONFIG)

lint:
	golangci-lint run
	golangci-lint fmt

examples/%.yaml: FORCE
	go run ./cmd/fazantix-validate-config $@

validate-examples: $(wildcard examples/*.yaml)

clean:
	rm -rvf build

all: fazantix

build: fazantix

.PHONY: FORCE
FORCE:;

.PHONY: clean run lint fazantix fazantix-wayland builddir validate-examples
