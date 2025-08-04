package kbdctl

import (
	"log"
	"maps"
	"slices"

	"github.com/fosdem/fazantix/lib/theatre"
	"github.com/fosdem/fazantix/lib/windowsink"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func SetupShortcutKeys(theatre *theatre.Theatre, ws *windowsink.WindowSink) {
	ws.Window.SetKeyCallback(keyCallback(theatre, ws.Frames().Name))
}

func Poll() {
	glfw.PollEvents()
}

func keyCallback(theatre *theatre.Theatre, stageName string) func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	names := make([]string, len(theatre.Scenes))
	copy(names, slices.Sorted(maps.Keys(theatre.Scenes)))

	return func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Release {
			if key == glfw.KeyQ &&
				mods&glfw.ModControl != 0 &&
				mods&glfw.ModShift != 0 {
				log.Println("told to quit, exiting")
				theatre.ShutdownRequested = true
			}
		}
		if action == glfw.Press {
			if key >= glfw.Key0 && key <= glfw.Key9 {
				selected := int(key - glfw.Key0)
				if selected > len(theatre.Scenes)-1 {
					log.Printf("Scene %d out of range\n", selected)
					return
				}
				log.Printf("set scene %s", names[selected])
				err := theatre.SetScene(stageName, names[selected], mods&glfw.ModShift != 0)
				if err != nil {
					log.Println(err)
					return
				}
			}
		}
	}
}
