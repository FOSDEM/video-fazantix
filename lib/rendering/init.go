package rendering

import (
	"fmt"

	"github.com/go-gl/gl/v4.1-core/gl"
)

func Init() error {
	err := gl.Init()
	if err != nil {
		return fmt.Errorf("could not initialise OpenGL context: %w", err)
	}
	return nil
}
