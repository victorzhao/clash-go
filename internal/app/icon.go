package app

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// The app's mark: a rounded blue blossom with a white lightning bolt at
// its centre, drawn in a 24×24 view box. The blossom is a circle whose
// radius swells and narrows ten times, like FlClash's traffic button.
var bolt = [][2]float64{{13, 2}, {4, 14}, {11, 14}, {10, 22}, {20, 10}, {13, 10}}

const (
	blossomR  = 9.0
	petals    = 10
	petalWave = 0.32
)

// IconPNG draws the app icon: a blue blossom with a white bolt on a
// transparent background, size pixels square. The same image is used for
// the executable, the window and the tray, so they always look alike.
func IconPNG(size int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	const box = 24.0
	scale := float64(size) / box
	center := float64(size) / 2

	blue := color.NRGBA{R: 37, G: 99, B: 235, A: 255} // #2563eb
	white := color.NRGBA{R: 255, G: 255, B: 255, A: 255}

	blossom := func(r, theta float64) bool {
		wave := 1 + petalWave*math.Cos(petals*theta)/2
		return r <= blossomR*scale*wave
	}

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			px, py := float64(x)+0.5-center, float64(y)+0.5-center
			r := math.Hypot(px, py)
			theta := math.Atan2(py, px)
			if blossom(r, theta) {
				img.Set(x, y, blue)
			}
		}
	}

	// The bolt takes 58% of the icon, centred, to sit inside the blossom.
	boltScale := float64(size) * 0.58 / box
	off := float64(size)/2 - 12*boltScale
	points := make([][2]float64, len(bolt))
	for i, p := range bolt {
		points[i] = [2]float64{p[0]*boltScale + off, p[1]*boltScale + off}
	}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if inside(points, float64(x)+0.5, float64(y)+0.5) {
				img.Set(x, y, white)
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil
	}
	return buf.Bytes()
}

// inside reports whether px,py is inside the polygon, by the even-odd rule.
func inside(poly [][2]float64, px, py float64) bool {
	in := false
	j := len(poly) - 1
	for i := 0; i < len(poly); i++ {
		if (poly[i][1] > py) != (poly[j][1] > py) &&
			px < (poly[j][0]-poly[i][0])*(py-poly[i][1])/(poly[j][1]-poly[i][1])+poly[i][0] {
			in = !in
		}
		j = i
	}
	return in
}
