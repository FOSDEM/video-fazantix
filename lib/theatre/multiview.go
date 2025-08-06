package theatre

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"log/slog"
	"os"

	"github.com/flopp/go-findfont"
	"github.com/fosdem/fazantix/lib/config"
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/fosdem/fazantix/lib/source/imgsource"
	"github.com/golang/freetype/truetype"
	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
)

type TallyData struct {
	Color color.RGBA
	Tally map[string]bool
}
type Multiview struct {
	Name string
	cfg  *config.MultiviewCfg

	overlayLayer string
	overlay      *imgsource.ImgSource
	overlayImage *image.RGBA
	theatre      *Theatre
	tallyToLayer map[string]*config.LayerCfg
	tallyCache   map[string]*TallyData
}

func drawRectangle(gc *draw2dimg.GraphicContext, x, y, w, h float64) {
	gc.MoveTo(x, y)
	gc.LineTo(x+w, y)
	gc.LineTo(x+w, y+h)
	gc.LineTo(x, y+h)
	gc.Close()
}

func drawTally(img *image.RGBA, x, y, width, height float32, colors []color.RGBA) {
	px := float64(x*float32(img.Bounds().Dx()) + 1)
	py := float64(y*float32(img.Bounds().Dy()) + 1)
	pw := float64(width*float32(img.Bounds().Dx()) - 2)
	ph := float64(height*float32(img.Bounds().Dy()) - 2)

	gc := draw2dimg.NewGraphicContext(img)
	gc.Save()
	if len(colors) == 0 {
		gc.SetStrokeColor(color.RGBA{R: 160, G: 160, B: 160, A: 255})
	} else {
		gc.SetStrokeColor(colors[0])
	}
	gc.SetFillColor(color.Transparent)
	gc.SetLineWidth(2)
	drawRectangle(gc, px, py, pw, ph)
	gc.Stroke()
}

func parseHexColor(s string) (c color.RGBA, err error) {
	c.A = 0xff
	switch len(s) {
	case 7:
		_, err = fmt.Sscanf(s, "#%02x%02x%02x", &c.R, &c.G, &c.B)
	case 4:
		_, err = fmt.Sscanf(s, "#%1x%1x%1x", &c.R, &c.G, &c.B)
		// Double the hex digits:
		c.R *= 17
		c.G *= 17
		c.B *= 17
	default:
		err = fmt.Errorf("invalid length, must be 7 or 4")

	}
	return
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

func makeLayerCfg(x, y, scale float32) *config.LayerCfg {
	return &config.LayerCfg{
		LayerStateCfg: config.LayerStateCfg{
			LayerTransform: layer.LayerTransform{
				X:       x,
				Y:       y,
				Scale:   scale,
				Opacity: 1,
			},
		},
		LayerCfgStub: config.LayerCfgStub{
			Warp: &config.LayerStateCfg{
				LayerTransform: layer.LayerTransform{
					X:       x,
					Y:       y,
					Scale:   scale,
					Opacity: 0,
				},
			},
		},
	}
}

func buildMultiviews(cfg *config.Config) []*Multiview {
	result := make([]*Multiview, 0)
	for multiviewName, multiview := range cfg.Multiviews {
		mv := &Multiview{
			Name:         multiviewName,
			cfg:          multiview,
			tallyToLayer: make(map[string]*config.LayerCfg),
			tallyCache:   make(map[string]*TallyData),
		}
		result = append(result, mv)
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

		layerCfg := make(map[string]*config.LayerCfg)
		overlay := image.NewRGBA(image.Rect(0, 0, multiview.Width, multiview.Height))
		index := 0
		positions := make([]*config.LayerCfg, 16)
		names := make([]string, 16)
		layers := make([]string, 16)
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
				positions[index] = makeLayerCfg(offsetX, offsetY, 0.25)
				index++
				positions[index] = makeLayerCfg(offsetX+0.25, offsetY, 0.25)
				index++
				positions[index] = makeLayerCfg(offsetX, offsetY+0.25, 0.25)
				index++
				positions[index] = makeLayerCfg(offsetX+0.25, offsetY+0.25, 0.25)
				index++
			} else {
				positions[index] = makeLayerCfg(offsetX, offsetY, 0.5)
				index++
			}
		}
		for idx, input := range multiview.Source {
			if input.Source != "" {
				layerCfg[input.Source] = &config.LayerCfg{
					LayerStateCfg: positions[idx].LayerStateCfg,
					LayerCfgStub:  positions[idx].LayerCfgStub,
				}
				names[idx] = input.Source
				layers[idx] = input.Source
				if cfg.Sources[input.Source].Label != "" {
					names[idx] = cfg.Sources[input.Source].Label
				}
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
			mv.tallyToLayer[layers[idx]] = box
		}
		mv.overlayImage = overlay

		overlayName := fmt.Sprintf("%s-overlay", multiviewName)
		mv.overlayLayer = overlayName
		cfg.Sources[overlayName] = &config.SourceCfg{
			SourceCfgStub: config.SourceCfgStub{
				Type:      "image",
				Z:         9999,
				MakeScene: false,
			},
			Cfg: &config.ImgSourceCfg{
				Width:   multiview.Width,
				Height:  multiview.Height,
				Inotify: false,
			},
		}
		layerCfg[overlayName] = makeLayerCfg(0, 0, 1)

		cfg.Scenes[multiviewName] = &config.SceneCfg{
			Tag:     "MV",
			Label:   multiviewName,
			Sources: layerCfg,
		}
	}
	return result
}

func (m *Multiview) Start(theatre *Theatre) {
	m.theatre = theatre
	m.overlay = theatre.Sources[m.overlayLayer].(*imgsource.ImgSource)
	err := m.overlay.SetImage(m.overlayImage)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to update overlay: %s", err.Error()), slog.String("module", m.Name))
		return
	}

	for stage, tallyConfig := range m.cfg.Tally {
		col, err := parseHexColor(tallyConfig.Color)
		if err != nil {
			panic("Invalid color")
		}
		m.tallyCache[stage] = &TallyData{
			Color: col,
			Tally: make(map[string]bool),
		}
	}

	theatre.AddEventListener("tally", m.onTally)
}

func (m *Multiview) onTally(t *Theatre, data interface{}) {
	tally := data.(*EventTallyData)
	if _, ok := m.cfg.Tally[tally.Stage]; ok {
		m.tallyCache[tally.Stage].Tally = tally.Tally
	}
	for name, box := range m.tallyToLayer {
		colors := make([]color.RGBA, 0)
		for _, tallyData := range m.tallyCache {
			if tallyData.Tally[name] {
				colors = append(colors, tallyData.Color)
			}
		}
		drawTally(m.overlayImage, box.X, box.Y, box.Scale, box.Scale, colors)
	}

	err := m.overlay.SetImage(m.overlayImage)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to update overlay: %s", err.Error()), slog.String("module", m.Name))
		return
	}
}
