CONFIG=imagesource.yaml

fazant:
	go build -o fazantix 'github.com/fosdem/fazantix/cmd/mixer'

run:
	go run 'github.com/fosdem/fazantix/cmd/mixer' $(CONFIG)

clean:
	rm mixer

all: fazantix

build: fazantix

.PHONY: clean run fazantix
