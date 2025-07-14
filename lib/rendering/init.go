package rendering

import (
	"fmt"
	"log"

	"github.com/go-gl/gl/v4.1-core/gl"
)

func Init() error {
	err := gl.Init()
	if err != nil {
		return fmt.Errorf("could not initialise OpenGL context: %w", err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Printf("OpenGL version '%s'", version)

	return nil
}
