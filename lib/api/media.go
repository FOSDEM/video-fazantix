package api

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"

	"github.com/fosdem/fazantix/lib/encdec"
	"github.com/fosdem/fazantix/lib/layer"
	"github.com/fosdem/fazantix/lib/source/imgsource"
)

type FrameForwarderObject interface {
	Frames() *layer.FrameForwarder
}

type MediaResponseType string

const (
	JPEG MediaResponseType = "jpeg"
	PNG  MediaResponseType = "png"
)

// @Summary	fetch a frame from a source or sink
// @Router		/api/media/source/{name} [get]
// @Router		/api/media/source/{name} [put]
// @Router		/api/media/source/{name}/{format} [get]
// @Router		/api/media/sink/{name} [get]
// @Router		/api/media/sink/{name}/{format} [get]
// @Tags		media
// @Param		name	path	string				true	"Name of the source to get the still from"
// @Param		format	path	MediaResponseType	true	"The image type to return"
// @Success	200
// @Failure	400	{string}	string	"The {name} parameter was not specified"
// @Failure	400	{string}	string	"The requested image format is not supported"
// @Failure	404	{string}	string	"The specified source does not exist in the configuration"
// @Failure	424	{string}	string	"The source does not have a frame ready to show"
// @Failure	500	{string}	string	"The API does not know how to convert this buffer to an image"
// @Produce	jpeg
func (a *Api) handleMediaSource(w http.ResponseWriter, req *http.Request) {
	sourceName := req.PathValue("source")
	sinkName := req.PathValue("sink")
	formatName := req.PathValue("format")
	if sourceName == "" && sinkName == "" {
		http.Error(w, "Missing source name", http.StatusBadRequest)
		return
	}
	var source FrameForwarderObject
	if sourceName != "" {
		source = a.theatre.Sources[sourceName]
		if source == nil {
			http.Error(w, "Source does not exist", http.StatusNotFound)
			return
		}
	} else {
		stage := a.theatre.Stages[sinkName]
		if stage == nil {
			http.Error(w, "Sink does not exist", http.StatusNotFound)
			return
		}
		source = stage.Sink
	}

	if req.Method == "GET" {
		frame := source.Frames().GetAnyFrameForReading()
		if frame == nil {
			http.Error(w, "No frame returned", http.StatusFailedDependency)
			return
		}
		defer source.Frames().FinishedReading(frame)

		bounds := image.Rectangle{
			Min: image.Point{},
			Max: image.Point{X: frame.Width, Y: frame.Height},
		}

		var img image.Image
		switch frame.Type {
		case encdec.RGBAFrames:
			img = image.NewNRGBA(bounds)
			copy(img.(*image.NRGBA).Pix, frame.Data)
		case encdec.RGBFrames:
			img = image.NewNRGBA(bounds)
			pix := img.(*image.NRGBA).Pix
			for i := range len(frame.Data) / 3 {
				pix[i*4+0] = frame.Data[i*3+0]
				pix[i*4+1] = frame.Data[i*3+1]
				pix[i*4+2] = frame.Data[i*3+2]
				pix[i*4+3] = 255
			}
		case encdec.YUV422Frames:
			img = image.NewYCbCr(bounds, image.YCbCrSubsampleRatio422)
			textureY, _, _ := frame.Texture(0)
			textureCb, _, _ := frame.Texture(1)
			textureCr, _, _ := frame.Texture(2)
			copy(img.(*image.YCbCr).Y, textureY)
			copy(img.(*image.YCbCr).Cb, textureCb)
			copy(img.(*image.YCbCr).Cr, textureCr)
		default:
			http.Error(w, "Unhandled frame type", http.StatusInternalServerError)
			return
		}
		switch formatName {
		case "":
			fallthrough
		case "jpeg":
			err := jpeg.Encode(w, img, &jpeg.Options{Quality: 80})
			if err != nil {
				http.Error(w, "Could not jpeg encode this frame", http.StatusInternalServerError)
				return
			}
		case "png":
			err := png.Encode(w, img)
			if err != nil {
				http.Error(w, "Could not png encode this frame", http.StatusInternalServerError)
				return
			}
		default:
			http.Error(w, "Unsupported format", http.StatusBadRequest)
		}
		return
	}

	imgSource, ok := source.(*imgsource.ImgSource)
	if !ok {
		http.Error(w, "not a valid image source", http.StatusBadRequest)
		return
	}

	if req.Method == "PUT" {
		newImage, ftype, err := image.Decode(req.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("not a valid image: %s", err), http.StatusBadRequest)
			return
		}
		log.Printf("Image source %s was updated with new %s image (%dx%d)\n", sourceName, ftype, newImage.Bounds().Dx(), newImage.Bounds().Dy())
		err = imgSource.SetImage(newImage)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not update image: %s", err), http.StatusBadRequest)
			return
		}
		return
	}

	http.Error(w, "Invalid method, only GET and PUT supported", http.StatusMethodNotAllowed)
}
