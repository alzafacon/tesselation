// Package pattern runs Conway's game of life on a tesselation pattern.
package pattern

import (
	"fmt"
)

// Cell conveniently wraps a 2D index (row, col)
type Cell struct {
	Row, Col int
}

// Offset is a synonym for Cell as a (readability) convenience
type Offset Cell

// Pattern represents a 2D pattern for Conway's Game of Life as a tessellation
type Pattern struct {
	// rows and cols are dimensions of rectangular array containing tile.
	rows, cols int

	// mask stores the cell id's.
	// A value of zero represents the cell is not part of the tile.
	// This can be used to determine whether a cell is in the tile or not.
	mask [][]int

	// Cells is an array of cell coordinates indexed by cell id.
	// These coordinates correspond the the cells that are part of the tile.
	// Cells that are in the array but are not part of the tile are excluded.
	// Note: Cells is reverse index to mask.
	Cells []Cell

	// Border is a map indexed by cell id to a slice of cell coordinates.
	// These coordinates are used to fill in the Border around a tile.
	Border map[int][]Cell
}

const alive = true
const dead = false

// New makes a tile based on a tile mask and rules for tesselating.
// The mask says which cells are in the tile. Must be rectangular. All cells on edge must be false.
// The rules say how to slide copies of the tile so the original is completely surrounded.
func New(mask [][]bool, rules []Offset) (*Pattern, error) {

	t := &Pattern{}

	t.rows = len(mask)
	t.cols = len(mask[0])

	// allocate t.mask
	t.mask = make([][]int, t.rows)
	underlying := make([]int, t.rows*t.cols)
	for i := range t.mask {
		t.mask[i], underlying = underlying[:t.cols], underlying[t.cols:]
	}

	// allocate t.Cells
	t.Cells = make([]Cell, 1) // "append" n times, up to needed size (amortized O(1))

	// Assign each cell in the tile an id.
	// also fill t.Cells
	id := 0
	for i, row := range mask {
		for j, cell := range row {
			if cell == alive {
				id++
				t.mask[i][j] = id
				t.Cells = append(t.Cells, Cell{i, j})
			}
		}
	}

	// Calculate Border by tessellating

	// Apply rules. Each rule creates a new copy of the tile.
	t.Border = make(map[int][]Cell)
	for _, rule := range rules {
		for id, c := range t.Cells {
			row := c.Row + rule.Row
			col := c.Col + rule.Col

			// check if offset cell is in range
			if (0 <= row && row < t.rows) && (0 <= col && col < t.cols) {
				// we assumed that the rules correctly tesselate the plane
				// here we just double check that the tiled copy is not causing overlap
				if mask[row][col] == dead {
					// check that the cell is neighbor to tile
					if countNeighbors(mask, row, col) > 0 {
						t.Border[id] = append(t.Border[id], Cell{row, col})
					}
				} else {
					return nil, fmt.Errorf("rule %v caused overlap r:%v c:%v, id:%v", rule, row, col, id)
				}
			}
		}
	}

	return t, nil
}

// Rows returns the number of rows in the underlying tile.
func (t *Pattern) Rows() int {
	return t.rows
}

// Cols returns the number of columns in the underlying tile.
func (t *Pattern) Cols() int {
	return t.cols
}

// Evolve finds the next generation in Conway's game of life
// Argument tile will have a border added to it.
func (t *Pattern) Evolve(tile [][]bool, newTile [][]bool) {

	// fill in the border around tile
	for id, v := range t.Border {
		tc := t.Cells[id] // find tile cell (tc) by id
		// each border cell (bc) with the above id gets the value at tc
		for _, bc := range v {
			tile[bc.Row][bc.Col] = tile[tc.Row][tc.Col]
		}
	}

	// cell id starts at 1, hence slice from 1
	for _, c := range t.Cells[1:] {
		newTile[c.Row][c.Col] = evolveCell(tile, c.Row, c.Col)
	}
}

// evolveCell applies Conway's rules to find new state of cell
func evolveCell(tile [][]bool, row, col int) bool {
	// require (row, col) in range of tile mask

	currentState := tile[row][col]
	liveNeighbors := countNeighbors(tile, row, col)

	if currentState == alive {
		if liveNeighbors < 2 { // lonely
			return dead
		}
		if liveNeighbors > 3 { // overpopulation
			return dead
		}

		return alive // otherwise stable

	} else if liveNeighbors == 3 {
		return alive // birth!
	}

	return dead // stays dead
}

// countNeighbors counts the number of adjacent cells on the board that are live
func countNeighbors(tile [][]bool, row, col int) int {

	// check if row or col are out of bounds
	if row < 0 || row >= len(tile) || col < 0 || col >= len(tile[0]) {
		return 0
	}

	nNeighbors := 0

	for r := row - 1; r <= row+1; r++ {
		for c := col - 1; c <= col+1; c++ {
			if r == row && c == col {
				continue
			}
			if r < 0 || r >= len(tile) || c < 0 || c >= len(tile[0]) {
				continue
			}

			if tile[r][c] == alive {
				nNeighbors++
			}
		}
	}

	return nNeighbors
}
