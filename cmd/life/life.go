//go:build js && wasm

package main

import (
	"log"
	"math"
	"syscall/js"
)

const (
	ROWS = 20
	COLS = 20
)

var neighbours = [8][2]int{
	{-1, -1}, {-1, 0}, {-1, 1},
	{0, -1}, {0, 1},
	{1, -1}, {1, 0}, {1, 1},
}

// Conway's Game of Life:
// 1. Any live cell with fewer than two live neighbours dies.
// 2. Any live cell with more than three live neighbours dies.
// 3. Any dead cell with exactly three live neighbours become a living cell.
type game struct {
	canvas      js.Value
	canvasCtx   js.Value
	pauseButton js.Value
	board       [ROWS][COLS]bool
	cellSize    float64
	tickMs      float64
	paused      bool
}

func newGame(canvas, canvasCtx, pauseButton js.Value) *game {
	return &game{
		canvas:      canvas,
		canvasCtx:   canvasCtx,
		pauseButton: pauseButton,
		tickMs:      200,
		paused:      false,
	}
}

func (g *game) run() {
	newBoard := g.board
	for i := range ROWS {
		for j := range COLS {
			n := g.aliveNeighbours(i, j)
			if g.board[i][j] {
				newBoard[i][j] = n == 2 || n == 3
			} else {
				newBoard[i][j] = n == 3
			}
		}
	}
	g.board = newBoard
}

func (g *game) aliveNeighbours(i, j int) int {
	wrap := func(v, maxV int) int {
		v %= maxV
		if v < 0 {
			v += maxV
		}
		return v
	}

	count := 0
	for _, n := range neighbours {
		ni := wrap(i+n[0], ROWS)
		nj := wrap(j+n[1], COLS)
		if g.board[ni][nj] {
			count++
		}
	}
	return count
}

func (g *game) drawBoard() {
	cs := g.cellSize
	// Clear previous round board
	g.canvasCtx.Call("clearRect", 0, 0, COLS*cs, ROWS*cs)

	for i := range ROWS {
		for j := range COLS {
			x, y := float64(j)*cs, float64(i)*cs
			if g.board[i][j] {
				g.canvasCtx.Set("fillStyle", "#000000")
			} else {
				g.canvasCtx.Set("fillStyle", "#ffffff")
			}
			g.canvasCtx.Call("fillRect", x, y, cs, cs)
		}
	}

	// Grid lines
	g.canvasCtx.Set("strokeStyle", "#cccccc")
	for i := range ROWS {
		y := float64(i) * cs
		g.canvasCtx.Call("beginPath")
		g.canvasCtx.Call("moveTo", 0, y)
		g.canvasCtx.Call("lineTo", COLS*cs, y)
		g.canvasCtx.Call("stroke")
	}
	for j := range COLS {
		x := float64(j) * cs
		g.canvasCtx.Call("beginPath")
		g.canvasCtx.Call("moveTo", x, 0)
		g.canvasCtx.Call("lineTo", x, ROWS*cs)
		g.canvasCtx.Call("stroke")
	}
}

func (g *game) registerHandlers() {
	g.canvas.Call("addEventListener", "click", js.FuncOf(
		func(this js.Value, args []js.Value) any {
			x := args[0].Get("offsetX").Float()
			y := args[0].Get("offsetY").Float()
			row := int(y / g.cellSize)
			col := int(x / g.cellSize)
			if row >= 0 && row < ROWS && col >= 0 && col < COLS {
				g.board[row][col] = !g.board[row][col]
			}
			return nil
		},
	))

	g.pauseButton.Call("addEventListener", "click", js.FuncOf(
		func(this js.Value, args []js.Value) any {
			g.paused = !g.paused
			if g.paused {
				g.pauseButton.Set("innerText", "Resume")
			} else {
				g.pauseButton.Set("innerText", "Pause")
			}
			return nil
		},
	))

	js.Global().Call("addEventListener", "resize", js.FuncOf(
		func(this js.Value, args []js.Value) any {
			g.resize()
			g.drawBoard()
			return nil
		},
	))
}

func (g *game) resize() {
	width := g.canvas.Get("clientWidth").Int()
	width = max(width, COLS)
	g.cellSize = float64(width) / COLS
	g.canvas.Set("width", width)
	g.canvas.Set("height", int(math.Round(g.cellSize*ROWS)))
}

func (g *game) drawGlider() {
	g.board[1][1] = true
	g.board[2][2] = true
	g.board[2][3] = true
	g.board[1][3] = true
	g.board[0][3] = true
}

func main() {
	log.Println("game of life")

	doc := js.Global().Get("document")
	if !doc.Truthy() {
		log.Fatalln("unable to get document object")
	}
	canvas := doc.Call("getElementById", "board")
	if !canvas.Truthy() {
		log.Fatalln("unable to get canvas element")
	}
	canvasCtx := canvas.Call("getContext", "2d")
	if !canvasCtx.Truthy() {
		log.Fatalln("unable to get canvas context")
	}
	pauseButton := doc.Call("getElementById", "pauseButton")
	if !pauseButton.Truthy() {
		log.Fatalln("unable to get pause button element")
	}

	g := newGame(canvas, canvasCtx, pauseButton)
	g.resize()
	g.drawGlider()
	g.registerHandlers()

	var last float64
	var tick js.Func
	tick = js.FuncOf(func(this js.Value, args []js.Value) any {
		now := args[0].Float()
		if now-last >= g.tickMs {
			g.drawBoard()
			if !g.paused {
				g.run()
			}
			last = now
		}
		js.Global().Call("requestAnimationFrame", tick)
		return nil
	})
	js.Global().Call("requestAnimationFrame", tick)

	select {}
}
