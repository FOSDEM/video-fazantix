package theatre

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"os"

	"github.com/flopp/go-findfont"
	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/golang/freetype/truetype"
	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
)

func drawRectangle(gc *draw2dimg.GraphicContext, x, y, w, h float64) {
	gc.MoveTo(x, y)
	gc.LineTo(x+w, y)
	gc.LineTo(x+w, y+h)
	gc.LineTo(x, y+h)
	gc.Close()
}

func drawMultiviewBox(img *image.RGBA, x, y, width, height float32, label string, size int, font *truetype.Font) {
	fontdata := draw2d.FontData{Name: "Font", Family: draw2d.FontFamilySans, Style: draw2d.FontStyleNormal}
	draw2d.RegisterFont(fontdata, font)

	px := float64(x * float32(img.Bounds().Dx()))
	py := float64(y * float32(img.Bounds().Dy()))
	pw := float64(width * float32(img.Bounds().Dx()))
	ph := float64(height * float32(img.Bounds().Dy()))

	gc := draw2dimg.NewGraphicContext(img)
	gc.Save()
	gc.SetStrokeColor(color.RGBA{R: 160, G: 160, B: 160, A: 255})
	gc.SetFillColor(color.Transparent)
	gc.SetLineWidth(4)
	drawRectangle(gc, px, py, pw, ph)
	gc.Stroke()
	if label == "" {
		return
	}
	gc.SetFontData(fontdata)
	gc.SetFillColor(color.Black)
	gc.SetFontSize(14)
	left, top, right, bottom := gc.GetStringBounds(label)
	twidth := right - left
	theight := bottom - top
	gc.SetFillColor(color.RGBA{R: 0, G: 0, B: 0, A: 150})
	hpad := float64(10)
	vpad := float64(5)
	drawRectangle(gc, px+(pw/2)-(twidth/2)-hpad, py+ph-theight-10-vpad, twidth+(2*hpad), theight+(2*vpad))
	gc.Fill()
	gc.SetFillColor(color.White)
	gc.FillStringAt(label, px+(pw/2)-(twidth/2), py+ph-10)
}

func buildMultiviews(cfg *config.Config) {
	for multiviewName, multiview := range cfg.Multiviews {
		fontPath, err := findfont.Find(multiview.Font)
		if err != nil {
			log.Printf("[%s] error finding font. %s\n", multiviewName, err)
			continue
		}
		fontData, err := os.ReadFile(fontPath)
		if err != nil {
			log.Printf("[%s] error loading font. %s\n", multiviewName, err)
			continue
		}
		font, err := truetype.Parse(fontData)
		if err != nil {
			log.Printf("[%s] error parsing font. %s\n", multiviewName, err)
			continue
		}

		scene := make(map[string]*config.LayerCfg)
		overlay := image.NewRGBA(image.Rect(0, 0, multiview.Width, multiview.Height))
		index := 0
		positions := make([]*layer.LayerState, 16)
		names := make([]string, 16)
		offsetX := float32(0.0)
		offsetY := float32(0.0)
		for quadrant, quadrantSplit := range multiview.Split {
			if quadrant%2 == 1 {
				offsetX = 0.5
			} else {
				offsetX = 0
			}
			if quadrant > 1 {
				offsetY = 0.5
			} else {
				offsetY = 0
			}
			if quadrantSplit {
				positions[index] = &layer.LayerState{
					X:       offsetX,
					Y:       offsetY,
					Scale:   0.25,
					Opacity: 1,
				}
				index++
				positions[index] = &layer.LayerState{
					X:       offsetX + 0.25,
					Y:       offsetY,
					Scale:   0.25,
					Opacity: 1,
				}
				index++
				positions[index] = &layer.LayerState{
					X:       offsetX,
					Y:       offsetY + 0.25,
					Scale:   0.25,
					Opacity: 1,
				}
				index++
				positions[index] = &layer.LayerState{
					X:       offsetX + 0.25,
					Y:       offsetY + 0.25,
					Scale:   0.25,
					Opacity: 1,
				}
				index++
			} else {
				positions[index] = &layer.LayerState{
					X:       offsetX,
					Y:       offsetY,
					Scale:   0.5,
					Opacity: 1,
				}
				index++
			}
		}
		for idx, input := range multiview.Source {
			if input.Source != "" {
				scene[input.Source] = &config.LayerCfg{
					LayerState: *positions[idx],
				}
				names[idx] = input.Source
			}
			if input.Label != "" {
				names[idx] = input.Label
			}
		}
		for idx, box := range positions {
			if box == nil {
				break
			}
			drawMultiviewBox(overlay, box.X, box.Y, box.Scale, box.Scale, names[idx], multiview.FontSize, font)
		}
		f, err := os.CreateTemp("", "multiview_*.png")
		if err != nil {
			log.Printf("[%s] error creating temp source. %s\n", multiviewName, err)
			continue
		}
		err = draw2dimg.SaveToPngFile(f.Name(), overlay)
		if err != nil {
			log.Printf("[%s] error writing overlay. %s\n", multiviewName, err)
			continue
		}

		overlayName := fmt.Sprintf("%s-overlay", multiviewName)
		cfg.Sources[overlayName] = &config.SourceCfg{
			SourceCfgStub: config.SourceCfgStub{
				Type:      "image",
				Z:         9999,
				MakeScene: false,
			},
			Cfg: &config.ImgSourceCfg{
				Path:    config.CfgPath(f.Name()),
				Inotify: false,
			},
		}
		scene[overlayName] = &config.LayerCfg{
			LayerState: layer.LayerState{
				X:       0,
				Y:       0,
				Scale:   1,
				Opacity: 1,
			},
		}

		cfg.Scenes[multiviewName] = scene
	}
}
