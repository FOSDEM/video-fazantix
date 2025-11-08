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
	match, err := regexp.Match(`#([0-9A-Fa-f]{2})*`, []byte(c))
	if err != nil {
		panic(err)
	}
	return match
}

func ColourParse(s string) Colour {
	var values [4]uint8
	n, _ := fmt.Sscanf(
		s, "#%02x%02x%02x%02x",
		&values[0], &values[1], &values[2], &values[3],
	)
	for i := n; i < 4; i++ {
		values[i] = 255
	}

	return Colour{
		R: float32(values[0]) / 255.0,
		G: float32(values[1]) / 255.0,
		B: float32(values[2]) / 255.0,
		A: float32(values[3]) / 255.0,
	}
}
