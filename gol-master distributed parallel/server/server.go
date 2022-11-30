package main

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var globW [][]byte
var cTurns int
var ImageHeight int
var ImageWidth int

// GetLocalIP : used to get IP address of server
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

// goes through 2D array to get number of alive cells
func getNumberAliveCells() int {

	count := 0
	for i := 0; i < len(globW); i++ {
		for j := 0; j < len(globW[0]); j++ {
			if globW[i][j] == 255 {
				count++
			}
		}
	}
	return count
}

// capability to work simultaneously
func worker(baseh int, toph int, world [][]byte, out chan<- [][]byte) {

	world2 := make([][]byte, ImageHeight)
	for k := 0; k < (ImageWidth); k++ {
		world2[k] = make([]byte, ImageWidth)
		for i := 0; i < ImageWidth; i++ {
			world2[k] = append(world2[k], 0x00)
		}
	}

	for i := baseh; i <= toph; i++ {
		for j := 0; j < (ImageWidth); j++ {

			count := countAliveCellsAroundCell(ImageHeight, ImageWidth, world, j, i)
			count = 255 - count + 1
			if world[i][j] == 0xFF {
				if count < 2 {
					world2[i][j] = 0x00
				} else if count == 2 || count == 3 {
					world2[i][j] = 0xFF
				} else {
					world2[i][j] = 0x00
				}
			} else {
				if count == 3 {
					world2[i][j] = 0xFF
				} else {
					world2[i][j] = 0x00
				}
			}
		}
	}

	out <- world2[baseh : toph+1]

}

// sums up the number of alive cells around a specific cell
func countAliveCellsAroundCell(ImageHeight int, ImageWidth int, world [][]byte, x int, y int) byte {
	sum := world[(y+ImageHeight-1)%ImageHeight][(x+ImageWidth-1)%ImageWidth] + world[(y+ImageHeight-1)%ImageHeight][(x+ImageWidth)%ImageWidth] +
		world[(y+ImageHeight-1)%ImageHeight][(x+ImageWidth+1)%ImageWidth] + world[(y+ImageHeight)%ImageHeight][(x+ImageWidth-1)%ImageWidth] + world[(y+ImageHeight)%ImageHeight][(x+ImageWidth+1)%ImageWidth] +
		world[(y+ImageHeight+1)%ImageHeight][(x+ImageWidth-1)%ImageWidth] + world[(y+ImageHeight+1)%ImageHeight][(x+ImageWidth)%ImageWidth] +
		world[(y+ImageHeight+1)%ImageHeight][(x+ImageWidth+1)%ImageWidth]

	return sum
}

// Executes a turn of the Game of Life.
func golLogic(turns int, world [][]byte, bh int, h int, threads int) [][]byte {

	if (h - bh + 1) < threads {
		threads = h - bh + 1
	}

	heights := make([]int, threads)
	for i := 0; i < (h - bh + 1); i++ {
		heights[i%threads]++
	}

	baseh := bh

	toph := bh - 1
	var sChanW []chan [][]byte

	for i := 0; i < threads; i++ {

		chanW := make(chan [][]byte)
		sChanW = append(sChanW, chanW)

		toph += heights[i]

		go worker(baseh, toph, world, sChanW[i])

		baseh += heights[i]

	}
	NewWorld := make([][]byte, h-bh+1)
	for i := range NewWorld {
		NewWorld[i] = make([]uint8, ImageWidth)
	}

	index := 0
	for i := 0; i < threads; i++ {
		v := <-sChanW[i]
		for _, row := range v {
			NewWorld[index] = row
			index++
		}
	}
	return NewWorld
}

// GameOperations structure used as factory
type GameOperations struct{}

func (s *GameOperations) GolDist(req stubs.BrokerRequest, res *stubs.ToBrokerResponse) (err error) {

	if req.NumTurns < 0 {
		err = errors.New(" A number of turns must be specified")
		return
	}

	ImageWidth = len(req.World)
	ImageHeight = len(req.World)
	SWorld := golLogic(req.NumTurns, req.World, req.Bh, req.H, req.Threads)

	res.SliceWorld = SWorld
	fmt.Println("Ending: ", req.TurnNumber)
	return
}

// AliveCellsC : finds all alive cells and puts them in a slice
func (s *GameOperations) AliveCellsC(req stubs.Request, res *stubs.Response) (err error) {
	res.AliveCount = getNumberAliveCells()
	res.CurrentTurn = cTurns

	return
}

// Quitting : used to shut down server
func (s *GameOperations) Quitting(req stubs.KQuit, res *stubs.StringM) (err error) {
	if req.Status == true {
		fmt.Println("quitting")
		res.Msg = "quitting"
		os.Exit(0)

	}
	return
}

func main() {
	pAddr := flag.String("port", "8050", "port to listen on")
	brokerAddr := flag.String("broker", "127.0.0.1:8030", "Address of broker instance")
	flag.Parse()
	client, _ := rpc.Dial("tcp", *brokerAddr)
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&GameOperations{})
	listen, err := net.Listen("tcp", ":"+*pAddr)
	if err != nil {
		fmt.Println(err)
	}
	status := new(stubs.StringM)
	client.Call(stubs.Subscribe, stubs.Subscription{ServerAddress: GetLocalIP() + ":" + *pAddr, Callback: "Factory.Multiply"}, status)

	defer listen.Close()

	rpc.Accept(listen)
	flag.Parse()

}
