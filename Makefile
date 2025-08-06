CONFIG=examples/imagesource.yaml

fazantix: prereqs
	go build -o build/fazantix ./cmd/fazantix

fazantix-wayland: prereqs
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

lib/api/static/index.html:
	./web_ui/build.sh

lib/api/docs/swagger.json: lib/api/static/index.html
	# requires index.html because swag wants a non-failing go build
	swag init -g lib/api/api.go -o lib/api/docs

builddir:
	mkdir -p build

prereqs: builddir lib/api/docs/swagger.json lib/api/static/index.html

clean:
	rm -rvf build

all: fazantix

build: fazantix

.PHONY: FORCE
FORCE:;

.PHONY: clean run lint fazantix fazantix-wayland builddir validate-examples prereqs
