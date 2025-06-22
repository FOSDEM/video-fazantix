package encdec

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
)

func DecodeYUYV422(buf []byte, image *image.YCbCr) error {
	if len(buf) < len(image.Cb)*4 {
		return fmt.Errorf("got a buf of len %d when %d was expected", len(buf), len(image.Cb)*4)
	}
	for i := range image.Cb {
		j := i * 4
		image.Y[i*2] = buf[j]
		image.Y[i*2+1] = buf[j+2]
		image.Cb[i] = buf[j+1]
		image.Cr[i] = buf[j+3]
	}
	return nil
}

func DecodeRGBfromImage(buf []byte) (*image.NRGBA, error) {
	img, _, err := image.Decode(bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	bounds := img.Bounds()
	nrgba := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(nrgba, nrgba.Bounds(), img, bounds.Min, draw.Src)
	return nrgba, nil
}
