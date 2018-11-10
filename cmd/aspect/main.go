package main

import (
	"fmt"
	"math"
)

const hexRadius = 128

type Point struct{ X, Y float64 }

var hexHeightScale = math.Sqrt(3) / 2

func hexTextureWidth(cols int) float64 {
	if cols == 0 {
		return 0
	}
	if cols < 0 {
		cols *= -1
	}
	return 3 + 1.5*(float64(cols)-1)
}

func hexTextureHeight(rows int) float64 {
	if rows == 0 {
		return 0
	}
	if rows < 0 {
		rows *= -1
	}
	return hexHeightScale*4 + (hexHeightScale*3)*(float64(rows)-1)
}

func main() {
	rows := 1
	cols := 2
	for n := 2; n < 50; n++ {
		// if this already fits, we don't need to do anything
		if n <= (rows * cols) {
			continue
		}
		// compute area if we add a row:
		nCols := int(math.Ceil(float64(n) / float64(rows+1)))
		aRow := hexTextureWidth(nCols) * hexTextureHeight(rows+1)
		// or a column
		aCol := hexTextureWidth(cols+1) * hexTextureHeight(rows)
		fmt.Printf("n = %d: aRow [%dx%d]: %.2f, aCol [%dx%d]: %.2f",
			n, nCols, rows+1, aRow,
			cols+1, rows, aCol)
		if aRow > aCol {
			fmt.Println(", adding col")
			cols++
		} else {
			fmt.Println(", adding row")
			rows++
			cols = nCols
		}
	}
}
