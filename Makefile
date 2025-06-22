fazant:
	go build -o fazant 'github.com/fosdem/fazantix/cmd/mixer'

run:
	go run 'github.com/fosdem/fazantix/cmd/mixer'

clean:
	rm mixer

all: fazant

build: fazant

.PHONY: clean run fazant
