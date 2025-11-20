CONFIG=examples/imagesource.yaml
SRCFILES := $(wildcard *.go) $(wildcard *.frag) $(wildcard *.vert) Makefile
WEBFILES := $(wildcard web_ui/*)

LDFLAGS=-compressdwarf=false

TAGS := dummy

.PHONY: build
build: prereqs build/fazantix build/fazantix-wayland

.PHONY: prebuild
prebuild: lib/api/static/index.html

.PHONY: develop
develop: prereqs
	./web_ui/build.sh serve

build/%: $(SRCFILES) go.sum lib/api/static/index.html
	go build -ldflags=$(LDFLAGS) -o $@ -tags "$(TAGS)" ./cmd/$*

build/fazantix-wayland: $(SRCFILES) go.sum lib/api/static/index.html
	go build -ldflags=$(LDFLAGS) -o $@ -tags "$(TAGS),wayland" ./cmd/fazantix

.PHONY: run
run: build/fazantix
	./build/fazantix $(CONFIG)

.PHONY: run-cage
run-cage: build/fazantix
	cage -- ./build/fazantix $(CONFIG)

.PHONY: lint
lint: prereqs
	golangci-lint run
	golangci-lint fmt

examples/%.yaml: FORCE
	go run ./cmd/fazantix-validate-config $@

.PHONY: validate-examples
validate-examples: $(wildcard examples/*.yaml)

lib/api/static/index.html:$(WEBFILES)
	./web_ui/build.sh

lib/api/docs/swagger.json: lib/api/static/index.html $(SRCFILES)
	# requires index.html because swag wants a non-failing go build
	go tool swag init -g lib/api/api.go -o lib/api/docs

.PHONY: builddir
builddir:
	@mkdir -p build

prereqs: builddir lib/api/docs/swagger.json lib/api/static/index.html

.PHONY: clean
clean:
	rm -rvf build
	rm -vf lib/api/static/index.html
	rm -vf lib/api/docs/*.{json,yaml}
	rm -vf lib/api/docs/docs.go

.PHONY: all
all: build

.PHONY: FORCE
FORCE:;
