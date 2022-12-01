package gol

import (
	"fmt"
	"net/rpc"
	"os"
	"strconv"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	keyPresses <-chan rune
}

// OutputWorldImage : used to send the world to convert it to a PGM image via ioOutput
func OutputWorldImage(c distributorChannels, p Params, world [][]byte, turn int) {
	c.ioCommand <- ioCheckIdle
	Idle := <-c.ioIdle
	if Idle == true {
		n := strconv.Itoa(turn)
		t := strconv.Itoa(p.ImageWidth)
		t = t + "x" + t

		c.ioCommand <- ioOutput
		c.ioFilename <- t + "x" + n

		for i := 0; i < p.ImageHeight; i++ {
			for j := 0; j < p.ImageWidth; j++ {
				c.ioOutput <- world[j][i]
			}
		}
	}

}

// goes through 2D array to get number of alive cells
func getNumberAliveCells(w [][]byte) int {

	count := 0
	for i := 0; i < len(w); i++ {
		for j := 0; j < len(w[0]); j++ {
			if w[i][j] == 255 {
				count++
			}
		}
	}
	return count
}

// finds all alive cells and puts them in a slice
func calculateAliveCells1(p Params, world [][]byte) []util.Cell {

	var slice []util.Cell
	for i := 0; i < p.ImageWidth; i++ {
		for j := 0; j < p.ImageHeight; j++ {
			if world[i][j] == 0xFF {
				slice = append(slice, util.Cell{i, j})
			}
		}
	}
	return slice
}

// Calls the broker, runs in parallel
func makeCall(client *rpc.Client, p Params, world [][]byte, out chan<- [][]byte, c distributorChannels) {
	request := stubs.Request{NumTurns: p.Turns, OgWorld: world, ImageHeight: p.ImageHeight, ImageWidth: p.ImageWidth}
	response := new(stubs.Response)
	client.Call(stubs.GolCalc, request, response)
	c.events <- AliveCellsCount{response.CurrentTurn, response.AliveCount}
	out <- response.Message

}

func distributor(p Params, c distributorChannels) {

	t := strconv.Itoa(p.ImageWidth)
	t = t + "x" + t
	c.ioCommand <- ioInput
	c.ioFilename <- t
	world := make([][]uint8, p.ImageHeight)
	for i := range world {
		world[i] = make([]uint8, p.ImageWidth)
	}

	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			world[j][i] = <-c.ioInput
			if world[j][i] == 0xFF {
				c.events <- CellFlipped{0, util.Cell{i, j}}
			}
		}
	}

	// connects to the RPC server and send the request(s)
	client, err := rpc.Dial("tcp", "127.0.0.1:8030")
	if err != nil {
		fmt.Println(err)
	}
	defer client.Close()

	// checks if the broker is already running
	qStatusReq := stubs.StringM{}
	qStatus := new(stubs.QStatus)
	client.Call(stubs.QStatusReq, qStatusReq, qStatus)

	if qStatus.Status == false {
		data := stubs.Request{OgWorld: world, ImageHeight: p.ImageHeight, ImageWidth: p.ImageWidth}
		response := new(stubs.StringM)
		client.Call(stubs.DataReceive, data, response)
		fmt.Println(response)
	}

	turns := 0
	var w [][]byte
	qCurrent := false

	chanW := make(chan [][]byte)

	ticker := time.NewTicker(2 * time.Second)
	ticker2 := time.NewTicker(2 * time.Millisecond)
	done := make(chan bool, 1)

	go makeCall(client, p, world, chanW, c)

	go func() {
		for {
			// different conditions
			select {
			case <-done:
				return

			case <-ticker2.C:
				requestC := stubs.StringM{}
				responseC := new(stubs.FlippedRes)
				client.Call(stubs.Flipped, requestC, responseC)
				for i := 0; i < len(responseC.CellF); i++ {
					c.events <- CellFlipped{responseC.TurnsForF, responseC.CellF[i]}
					//c.events <- TurnComplete{responseC.TurnsForF}
				}
				c.events <- TurnComplete{responseC.TurnsForF}

			case <-ticker.C:

				requestA := stubs.Request{}
				responseA := new(stubs.SavedC)
				client.Call(stubs.GetStateB, requestA, responseA)
				turns = responseA.Turns
				w = responseA.World
				aliveCount := getNumberAliveCells(w)
				c.events <- AliveCellsCount{turns, aliveCount}

			case command := <-c.keyPresses:
				switch command {
				case 's':
					c.events <- StateChange{turns, Executing}
					OutputWorldImage(c, p, w, turns)
				case 'q':
					reply := new(stubs.StringM)
					stringSent := stubs.StringM{Msg: "request about change q status"}
					client.Call(stubs.ChangeQ, stringSent, reply)
					c.events <- StateChange{turns, Quitting}
					qCurrent = true
				case 'p':

					c.events <- StateChange{turns, Paused}
					OutputWorldImage(c, p, w, turns)
					reply := new(stubs.StringM)
					PStatus := stubs.PPress{Status: true}
					client.Call(stubs.PressP, PStatus, reply)
					pState := 0

					for {
						command := <-c.keyPresses
						switch command {
						case 'p':
							fmt.Println("Continuing")
							c.events <- StateChange{turns, Executing}
							c.events <- TurnComplete{turns}
							reply := new(stubs.StringM)
							PStatus := stubs.PPress{Status: false}
							client.Call(stubs.PressP, PStatus, reply)
							pState = 1
						}
						if pState == 1 {
							break
						}
					}
				case 'k':
					c.events <- StateChange{turns, Quitting}
					OutputWorldImage(c, p, w, turns)
					c.ioCommand <- ioCheckIdle
					Idle := <-c.ioIdle
					kQuitMsg := new(stubs.KQuit)
					kState := stubs.KPress{Status: true}
					client.Call(stubs.QuittingB, kState, kQuitMsg)

					if Idle == true {
						c.events <- FinalTurnComplete{0, make([]util.Cell, 0)}
						os.Exit(0)
					}
				}
			default:
			}
			if qCurrent == true {
				break
			}
		}
		if qCurrent == true {
			os.Exit(0)
		}

	}()

	ResWorld := <-chanW
	done <- false

	var alive []util.Cell
	alive = calculateAliveCells1(p, ResWorld)

	// Reports the final state using FinalTurnCompleteEvent.
	// Makes sure that the Io has finished any output before exiting.

	c.events <- FinalTurnComplete{p.Turns, alive}

	c.ioCommand <- ioCheckIdle
	Idle := <-c.ioIdle
	if Idle == true {
		n := strconv.Itoa(p.Turns)
		c.ioCommand <- ioOutput

		c.ioFilename <- t + "x" + n

		for i := 0; i < p.ImageHeight; i++ {
			for j := 0; j < p.ImageWidth; j++ {
				c.ioOutput <- ResWorld[j][i]
			}
		}
	}

	c.ioCommand <- ioCheckIdle
	Idle = <-c.ioIdle

	if Idle == true {

		c.events <- StateChange{p.Turns, Quitting}

	}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
