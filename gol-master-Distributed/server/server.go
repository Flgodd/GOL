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

var ImageHeight int
var ImageWidth int

// GetLocalIP : used to get IP address of server
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback then display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

// sums up the number of alive cells around a specific cell
func countAliveCellsAroundCell(ImageHeight int, ImageWidth int, world [][]byte, x int, y int) byte {
	sum := world[(y+ImageHeight-1)%ImageHeight][(x+ImageWidth-1)%ImageWidth] + world[(y+ImageHeight-1)%ImageHeight][(x+ImageWidth)%ImageWidth] +
		world[(y+ImageHeight-1)%ImageHeight][(x+ImageWidth+1)%ImageWidth] + world[(y+ImageHeight)%ImageHeight][(x+ImageWidth-1)%ImageWidth] + world[(y+ImageHeight)%ImageHeight][(x+ImageWidth+1)%ImageWidth] +
		world[(y+ImageHeight+1)%ImageHeight][(x+ImageWidth-1)%ImageWidth] + world[(y+ImageHeight+1)%ImageHeight][(x+ImageWidth)%ImageWidth] +
		world[(y+ImageHeight+1)%ImageHeight][(x+ImageWidth+1)%ImageWidth]

	return sum
}

// executes a turn of game of life
func golLogic(world [][]byte, bh int, h int) [][]byte {

	world2 := make([][]byte, ImageHeight)
	for k := 0; k < (ImageWidth); k++ {
		world2[k] = make([]byte, ImageWidth)
		for i := 0; i < ImageWidth; i++ {
			world2[k] = append(world2[k], 0x00)
		}
	}

	for i := bh; i <= h; i++ {
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

	return world2[bh : h+1]
}

// GameOperations structure used as factory
type GameOperations struct{}

func (s *GameOperations) GolDist(req stubs.BrokerRequest, res *stubs.ToBrokerResponse) (err error) {

	if req.NumTurns < 0 {
		err = errors.New(" A number of turns must be specified")
		return
	}
	fmt.Println("Starting: ", req.TurnNumber)
	ImageWidth = len(req.World)
	ImageHeight = len(req.World)
	SWorld := golLogic(req.World, req.Bh, req.H)
	res.SliceWorld = SWorld
	fmt.Println("Ending: ", req.TurnNumber)
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
	brokerAddr := flag.String("broker", "54.88.252.205:8030", "Address of broker instance")
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
