package kbdctl

import (
	"log"
	"maps"
	"slices"

	"github.com/fosdem/fazantix/theatre"
	"github.com/fosdem/fazantix/windowsink"
	"github.com/go-gl/glfw/v3.3/glfw"
)

func SetupShortcutKeys(theatre *theatre.Theatre, ws *windowsink.WindowSink) {
	ws.Window.SetKeyCallback(keyCallback(theatre, ws.Frames().Name))
}

func keyCallback(theatre *theatre.Theatre, stageName string) func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
	scenes := slices.Sorted(maps.Keys(theatre.Scenes))
	names := make([]string, len(theatre.Scenes))
	for i, n := range scenes {
		names[i] = n
	}

	return func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		if action == glfw.Release {
			if key == glfw.KeyQ &&
				mods&glfw.ModControl != 0 &&
				mods&glfw.ModShift != 0 {
				log.Println("told to quit, exiting")
				w.SetShouldClose(true)
			}
		}
		if action == glfw.Press {
			if key >= glfw.Key0 && key <= glfw.Key9 {
				selected := int(key - glfw.Key0)
				if selected > len(theatre.Scenes)-1 {
					log.Printf("Scene %d out of range\n", selected)
					return
				}
				log.Printf("set scene %s\n", names[selected])
				err := theatre.SetScene(stageName, names[selected])
				if err != nil {
					log.Println(err)
					return
				}
			}
		}
	}
}
