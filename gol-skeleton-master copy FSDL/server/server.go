package main

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

// initial world

var globW [][]byte
var cTurns int
var ImageHeight int
var ImageWidth int
var CellFlippedArray []util.Cell
var QState bool
var PPress bool

var mutex_g sync.Mutex

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

func countAliveCellsAroundCell(ImageHeight int, ImageWidth int, world [][]byte, x int, y int) byte {
	sum := world[(y+ImageHeight-1)%ImageHeight][(x+ImageWidth-1)%ImageWidth] + world[(y+ImageHeight-1)%ImageHeight][(x+ImageWidth)%ImageWidth] +
		world[(y+ImageHeight-1)%ImageHeight][(x+ImageWidth+1)%ImageWidth] + world[(y+ImageHeight)%ImageHeight][(x+ImageWidth-1)%ImageWidth] + world[(y+ImageHeight)%ImageHeight][(x+ImageWidth+1)%ImageWidth] +
		world[(y+ImageHeight+1)%ImageHeight][(x+ImageWidth-1)%ImageWidth] + world[(y+ImageHeight+1)%ImageHeight][(x+ImageWidth)%ImageWidth] +
		world[(y+ImageHeight+1)%ImageHeight][(x+ImageWidth+1)%ImageWidth]

	return sum
}

// golLogic : used to calculate next state
func golLogic(world [][]byte, bh int, h int) [][]byte {

	//Executes all turns of the Game of Life.
	var Cf []util.Cell
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
					Cf = append(Cf, util.Cell{j, i})
				} else if count == 2 || count == 3 {
					world2[i][j] = 0xFF
				} else {
					world2[i][j] = 0x00
					Cf = append(Cf, util.Cell{j, i})
				}
			} else {
				if count == 3 {
					world2[i][j] = 0xFF
					Cf = append(Cf, util.Cell{j, i})
				} else {
					world2[i][j] = 0x00
				}
			}
		}
	}
	CellFlippedArray = Cf
	return world2[bh : h+1]
}

type GameOperations struct{}

// GolDist : used to get next state
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

	res.CellFlippedArray = CellFlippedArray

	fmt.Println("Ending: ", req.TurnNumber)
	return
}

func (s *GameOperations) AliveCellsC(req stubs.Request, res *stubs.Response) (err error) {
	res.AliveCount = getNumberAliveCells()
	res.CurrentTurn = cTurns

	return
}

func (s *GameOperations) GetState(req stubs.Request, res *stubs.SavedC) (err error) {
	mutex_g.Lock()

	res.Turns = cTurns
	res.World = globW
	mutex_g.Unlock()

	res.ImageHeight = ImageHeight
	res.ImageWidth = ImageWidth
	return
}

/*func (s *GameOperations) ChangeQ(req stubs.StringM, res *stubs.StringM) (err error) {
	res.Msg = " q state changed"
	QState = true
	return
}*/

func (s *GameOperations) Quitting(req stubs.KQuit, res *stubs.StringM) (err error) {
	if req.Status == true {
		fmt.Println("quitting")
		res.Msg = "quitting"
		os.Exit(0)

	}
	return
}

/*func (s *GameOperations) PPressed(req stubs.PPress, res *stubs.StringM) {
	PPress = req.Status
	return
}*/

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
