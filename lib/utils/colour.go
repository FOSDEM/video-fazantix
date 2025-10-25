package utils

import (
	"fmt"
	"image/color"
	"regexp"
)

func ColourValidate(c string) bool {
	match, err := regexp.Match(`#[0-9A-Fa-f]{8}`, []byte(c))
	if err != nil {
		panic(err)
	}
	return match
}

func ColourParse(s string) (c color.RGBA) {
	fmt.Sscanf(s, "#%02x%02x%02x%02x", &c.R, &c.G, &c.B, &c.A)
	return
}
