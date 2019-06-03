package main

import (
	"fmt"
	"seebs.net/modus/g"
)

type RandColler interface {
	RandCol() int
}

func main() {
	var ctx = g.NewContext(320, 200, false)
	var gr = ctx.NewSquareGrid(40, 1)
	var rc RandColler = gr
	fmt.Printf("%d\n", rc.RandCol())
}

