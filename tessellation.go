package main

import (
	"encoding/csv"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"log"
	"os"

	"github.com/fidelcoria/tessellation/pattern"
)

// GIF colors
var on = color.RGBA{163, 73, 164, 255}          // purplish
var off = color.RGBA{200, 191, 231, 255}        // light lila
var background = color.RGBA{164, 149, 120, 255} // light brown

var palette = color.Palette{
	on,
	off,
	background,
}

// Circle is used as a mask shape to draw the GIF.
type Circle struct {
	P image.Point
	R int
}

// ColorModel returns color.Model of Circle; implements Image interface.
func (c *Circle) ColorModel() color.Model {
	return color.AlphaModel
}

// Bounds returns bounds of circle; implements Image interface.
func (c *Circle) Bounds() image.Rectangle {
	return image.Rect(c.P.X-c.R, c.P.Y-c.R, c.P.X+c.R, c.P.Y+c.R)
}

// At finds if (x, y) is in the circle or not.
func (c *Circle) At(x, y int) color.Color {
	xx, yy, rr := float64(x-c.P.X)+0.5, float64(y-c.P.Y)+0.5, float64(c.R)
	if xx*xx+yy*yy < rr*rr {
		return color.Alpha{255} // opaque
	}
	return color.Alpha{0} // transparent
}

func main() {
	maskData := readCSV("data/tile mask.csv")
	mask := make([][]bool, len(maskData))
	for i, record := range maskData {
		mask[i] = make([]bool, len(record))
		for j, field := range record {
			if field == "1" {
				mask[i][j] = true
			}
		}
	}

	tileData := readCSV("data/tile.csv")
	aTile := make([][]bool, len(tileData))
	for i, record := range tileData {
		aTile[i] = make([]bool, len(record))
		for j, field := range record {
			if field == "X" {
				aTile[i][j] = true
			}
		}
	}

	bTile := make([][]bool, len(aTile))
	for i := range bTile {
		bTile[i] = make([]bool, len(aTile[0]))
	}

	// translations are the rules used to slide the tile and tessellate all around the first tile
	translations := []pattern.Offset{
		{Row: -10},
		{Row: 10},

		{Row: -20, Col: 10},
		{Row: -10, Col: 10},

		{Row: 20, Col: -10},
		{Row: 10, Col: -10},
	}

	tess, err := pattern.New(mask, translations)
	if err != nil {
		fmt.Println(err)
		return
	}

	// number of frames to calculate (0.gif not included)
	nFrames := 42

	names := make([]string, nFrames+1)

	// save initial frame (the frames directory must already exist)
	names[0] = "frames/0.gif"
	saveGIFFrame(tess, aTile, names[0])

	for i, j := 1, 2; j <= nFrames; i, j = i+2, j+2 {
		// the tile is evolved twice each iteration
		// this avoids having to allocate new arrays

		tess.Evolve(aTile, bTile)
		names[i] = fmt.Sprintf("frames/%d.gif", i)
		saveGIFFrame(tess, bTile, names[i])

		tess.Evolve(bTile, aTile)
		names[j] = fmt.Sprintf("frames/%d.gif", j)
		saveGIFFrame(tess, aTile, names[j])
	}

	composeGIF(names, "evolution.gif")
}

func readCSV(name string) [][]string {
	fileReader, err := os.Open(name)
	r := csv.NewReader(fileReader)

	records, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	fileReader.Close()

	return records
}

func saveGIFFrame(t *pattern.Pattern, tile [][]bool, name string) {
	// create masks for painting cells
	// these are colored solid and masked with a circle
	onSrc := &image.Uniform{on}
	offSrc := &image.Uniform{off}

	// I am visualizing the grid per the docs, so x=cols and y=rows
	// each cell is getting a 10x10 square
	img := image.NewPaletted(image.Rect(0, 0, 10*t.Cols(), 10*t.Rows()), palette)
	// set background color
	draw.Draw(img, img.Bounds(), &image.Uniform{background}, image.ZP, draw.Src)

	for _, cell := range t.Cells[1:] {

		cellRegion := image.Rectangle{
			image.Point{cell.Col * 10, cell.Row * 10},
			image.Point{cell.Col*10 + 10, cell.Row*10 + 10},
		}

		center := image.Point{cell.Col*10 + 5, cell.Row*10 + 5}

		var src *image.Uniform

		if tile[cell.Row][cell.Col] {
			src = onSrc
		} else {
			src = offSrc
		}

		// 4 is one less than 5, the radius of the square
		dot := &Circle{P: center, R: 4}
		draw.DrawMask(img, cellRegion, src, image.ZP, dot, dot.Bounds().Min, draw.Over)
	}

	f, _ := os.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0600)
	defer f.Close()
	gif.Encode(f, img, nil)
}

// http://tech.nitoyon.com/en/blog/2016/01/07/go-animated-gif-gen/
// TODO: there's a better way... only draw the parts that have changed
func composeGIF(frames []string, name string) {
	outGIF := &gif.GIF{}
	for _, file := range frames {
		f, _ := os.Open(file)
		inGIF, _ := gif.Decode(f)
		f.Close()

		outGIF.Image = append(outGIF.Image, inGIF.(*image.Paletted)) // type assertion
		outGIF.Delay = append(outGIF.Delay, 0)
	}

	f, _ := os.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0600)
	defer f.Close()
	gif.EncodeAll(f, outGIF)
}

func tilePrint(g [][]bool) {
	for _, record := range g {
		for _, field := range record {
			if field {
				fmt.Print("1")
			} else {
				fmt.Print(" ")
			}

		}
		fmt.Println()
	}
	fmt.Println("=================================================")
}
