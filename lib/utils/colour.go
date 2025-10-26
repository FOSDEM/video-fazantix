package utils

import (
	"fmt"
	"regexp"
)

type Colour struct {
	R float32
	G float32
	B float32
	A float32
}

func ColourValidate(c string) bool {
	match, err := regexp.Match(`#[0-9A-Fa-f]{8}`, []byte(c))
	if err != nil {
		panic(err)
	}
	return match
}

func ColourParse(s string) Colour {
	var rb, gb, bb, ab uint8
	fmt.Sscanf(s, "#%02x%02x%02x%02x", &rb, &gb, &bb, &ab)
	return Colour{
		R: float32(rb) / 255.0,
		G: float32(gb) / 255.0,
		B: float32(bb) / 255.0,
		A: float32(ab) / 255.0,
	}
}
