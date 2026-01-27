package rendering

import (
	"github.com/go-gl/glfw/v3.3/glfw"
)

func SetVsync(syncToDisplay bool) {
	if syncToDisplay {
		glfw.SwapInterval(1)
	} else {
		glfw.SwapInterval(0)
	}
}
