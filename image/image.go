package image

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"

	"github.com/vegarsti/extract/box"
)

// AddBoxes adds bounding boxes to the base64 encoded image and returns a new base64 encoded image
func AddBoxes(imageBytes []byte, boxes []box.Box) (string, error) {
	imgReader := bytes.NewReader(imageBytes)
	img, _, err := image.Decode(imgReader)
	if err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}

	// Create a new image for the output
	bounds := img.Bounds()
	outputImg := image.NewRGBA(bounds)
	draw.Draw(outputImg, bounds, img, bounds.Min, draw.Src)

	// Draw the boxes
	for _, box := range boxes {
		drawBox(outputImg, box, bounds)
	}

	// Encode the modified image back to base64
	var buf bytes.Buffer
	err = png.Encode(&buf, outputImg)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// drawBox draws a single Box on the image
func drawBox(img *image.RGBA, box box.Box, bounds image.Rectangle) {
	col := color.RGBA{255, 0, 0, 255} // Red color for the box outline
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	// Convert normalized coordinates to pixel coordinates
	x1 := int(box.XLeft * float64(imgWidth))
	x2 := int(box.XRight * float64(imgWidth))
	y1 := int(box.YTop * float64(imgHeight))
	y2 := int(box.YBottom * float64(imgHeight))

	// Draw the rectangle outline
	for x := x1; x <= x2; x++ {
		img.Set(x, y1, col)
		img.Set(x, y2, col)
	}
	for y := y1; y <= y2; y++ {
		img.Set(x1, y, col)
		img.Set(x2, y, col)
	}
}
