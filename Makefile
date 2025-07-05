CONFIG=imagesource.yaml

fazantix:
	go build -o fazantix 'github.com/fosdem/fazantix/cmd/mixer'

fazantix-wayland:
	go build -o fazantix -tags "wayland,vulkan" 'github.com/fosdem/fazantix/cmd/mixer'

run: fazantix
	./fazantix $(CONFIG)

run-wayland: fazantix-wayland
	./fazantix $(CONFIG)

run-cage: fazantix-wayland
	cage -- ./fazantix $(CONFIG)

clean:
	rm -f fazantix fazantix-wayland

all: fazantix

build: fazantix

.PHONY: clean run fazantix
