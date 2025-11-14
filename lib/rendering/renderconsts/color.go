package renderconsts

import (
	"github.com/go-gl/gl/v4.1-core/gl"
)

type Color int32

const (
	RED   Color = gl.RED
	GREEN Color = gl.GREEN
	BLUE  Color = gl.BLUE
	ALPHA Color = gl.ALPHA
	ZERO  Color = gl.ZERO
	ONE   Color = gl.ONE
)
