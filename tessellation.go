// Package tessellation TODO
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

const (
	maskFile = "data/mask.csv"
	tileFile = "data/tile.csv"
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
	maskData := readCSV(maskFile)
	mask := make([][]bool, len(maskData))
	for i, record := range maskData {
		mask[i] = make([]bool, len(record))
		for j, field := range record {
			if field == "1" {
				mask[i][j] = true
			}
		}
	}

	tileData := readCSV(tileFile)
	aTile := make([][]bool, len(tileData))
	for i, record := range tileData {
		aTile[i] = make([]bool, len(record))
		for j, field := range record {
			if field == "X" {
				aTile[i][j] = true
			}
		}
	}

	// for bordering TODO read from file, maybe?
	translations := []pattern.Offset{
		{Row: -10, Col: -10},
		{Row: -10, Col: 0},
		{Row: -10, Col: 10},
		{Row: 0, Col: -10},
		{Row: 0, Col: 10},
		{Row: 10, Col: -10},
		{Row: 10, Col: 0},
		{Row: 10, Col: 10},
	}

	tess, err := pattern.New(mask, translations)
	if err != nil {
		fmt.Println(err)
		return
	}

	// these additional translations are used to tile the entire GIF frame
	translations = append(translations,
		pattern.Offset{Row: 20, Col: -10},
		pattern.Offset{Row: 20, Col: 0},
		pattern.Offset{Row: 20, Col: 10},
		pattern.Offset{Row: 20, Col: 20},

		pattern.Offset{Row: -10, Col: 20},
		pattern.Offset{Row: 0, Col: 20},
		pattern.Offset{Row: 10, Col: 20},
	)
	// number of frames to calculate (0.gif not included)
	nFrames := 42 // found by trial and error...

	play(tess, aTile, translations, 2, 2, nFrames)
}

// play runs the simulation and creates the GIFs
// pat has information about the tile pattern
// aTile is the original (first generation) tile
// shifts indicate how to shift tile to tessellate the GIF frame
// nFrames is the number of generations to calculate
func play(pat *pattern.Pattern, aTile [][]bool, shifts []pattern.Offset, repH, repV int, nFrames int) {

	bTile := make([][]bool, len(aTile))
	for i := range bTile {
		bTile[i] = make([]bool, len(aTile[0]))
	}

	names := make([]string, nFrames+1)

	// save initial frame (the frames directory must already exist)
	names[0] = "frames/0.gif"
	saveGIFFrame(pat, shifts, repH, repV, aTile, names[0])

	for i, j := 1, 2; j <= nFrames; i, j = i+2, j+2 {
		// the tile is evolved twice each iteration
		// this avoids having to allocate new arrays

		pat.Evolve(aTile, bTile)
		names[i] = fmt.Sprintf("frames/%d.gif", i)
		saveGIFFrame(pat, shifts, repH, repV, bTile, names[i])

		pat.Evolve(bTile, aTile)
		names[j] = fmt.Sprintf("frames/%d.gif", j)
		saveGIFFrame(pat, shifts, repH, repV, aTile, names[j])
	}

	composeGIF(names, "evolution.gif")
}

// readCSV wraps boiler plate code for reading a CSV.
// name is the name of the csv file
func readCSV(name string) [][]string {
	fileReader, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	r := csv.NewReader(fileReader)

	records, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	fileReader.Close() // why not defer like for GIFs

	return records
}

// saveGIFFrame saves a GIF of the tile passed.
// pat has information about the tile pattern
// shifts are offsets for tiling the GIF frame
// repH, for size of GIF, counts how many times to repeat horizontally
// repV, for size of GIF, counts how many times to repeat vertically
// tile contains shape of pattern
// name is name of output GIF
func saveGIFFrame(pat *pattern.Pattern, shifts []pattern.Offset, repH, repV int, tile [][]bool, name string) {
	// create masks for painting cells
	// these are colored solid and masked with a circle
	onSrc := &image.Uniform{on}
	offSrc := &image.Uniform{off}

	// each cell (dot) is in a square of size squarePix
	squarePix := 10

	// I am visualizing the grid per the docs, so x=cols and y=rows
	// each cell is getting a 10x10 square
	img := image.NewPaletted(image.Rect(0, 0, squarePix*pat.Cols()*repH, squarePix*pat.Rows()*repV), palette)
	// set background color
	draw.Draw(img, img.Bounds(), &image.Uniform{background}, image.ZP, draw.Src)

	shifts = append(shifts, pattern.Offset{Row: 0, Col: 0})

	for _, cell := range pat.Cells {
		for _, rule := range shifts {
			offsetCol, offsetRow := cell.Col+rule.Col, cell.Row+rule.Row

			cellRegion := image.Rect(
				offsetCol*squarePix, offsetRow*squarePix,
				offsetCol*squarePix+squarePix, offsetRow*squarePix+squarePix,
			)

			var src *image.Uniform

			if tile[cell.Row][cell.Col] {
				src = onSrc
			} else {
				src = offSrc
			}

			// 4 is one less than 5, the radius of the square
			dot := &Circle{R: 4} // center doesn't matter since shape gets aligned to cellRegion
			draw.DrawMask(img, cellRegion,
				src, image.ZP,
				dot, dot.Bounds().Min.Add(image.Point{-1, -1}), // shift by -1,-1 to center dots
				draw.Over,
			)
		}
	}

	f, _ := os.OpenFile(name, os.O_WRONLY|os.O_CREATE, 0600)
	defer f.Close() // why defer instead of closing after encoding
	gif.Encode(f, img, nil)
}

// composeGIF composes a group of GIF images into a single one.
// frames is a slice with the names of the GIFs to compose
// name is the name of the final GIF
// credits: http://tech.nitoyon.com/en/blog/2016/01/07/go-animated-gif-gen/
// TODO: there's a better way... only draw the parts that have changed
//			that would require decoupling play, saveGIFFrame and composeGIF
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

// tilePrint is convenient for printing the tile to console.
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
