package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"sync"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var Threads int
var BrokerWorld [][]byte
var cTurns int
var ImageHeight int
var ImageWidth int
var mutex_g sync.Mutex
var PState bool

var QState bool

// used to call the servers, run in parallel
func makeCall(client *rpc.Client, bh int, h int, turn int, th int, i int, out chan<- [][]byte, threads int) {
	request := stubs.BrokerRequest{Bh: bh, H: h, World: BrokerWorld, TurnNumber: turn, Threads: threads}
	response := new(stubs.ToBrokerResponse)
	client.Call(stubs.GolHandler, request, response)
	out <- response.SliceWorld
}

// Broker is considered to be the remote manager
type Broker struct {
	store []string
}

// PressP : used to pause
func (s *Broker) PressP(req stubs.PPress, res *stubs.StringM) (err error) {
	fmt.Println("PAUSED")
	PState = req.Status
	return
}

// ChangeQ : used to notify when client quits
func (s *Broker) ChangeQ(req stubs.StringM, res *stubs.StringM) (err error) {
	res.Msg = " q state changed"
	fmt.Println("Q Pressed")
	PState = true
	QState = true
	return
}

// GolCalc : Executes all turns of the Game of Life.
func (s *Broker) GolCalc(req stubs.Request, res *stubs.Response) (err error) {
	var turn int
	turnInit := 0
	if QState == true {
		turnInit = cTurns

	}
	PState = false
	// splits heights per server fairly
	heights := make([]int, len(s.store))
	for i := 0; i < ImageHeight; i++ {
		heights[i%len(s.store)]++
	}

	var clientArray []*rpc.Client

	for i := 0; i < len(s.store); i++ {
		client, _ := rpc.Dial("tcp", s.store[i])
		clientArray = append(clientArray, client)
	}

	for turn = turnInit; turn < req.NumTurns; {
		if PState == false {
			if QState == true {
				fmt.Println("Q Pressed")
			}
			bh := 0
			h := -1
			var sChanW []chan [][]byte

			for i := 0; i < len(s.store); i++ {

				chanW := make(chan [][]byte)
				sChanW = append(sChanW, chanW)

				h += heights[i]
				th := h - bh + 1

				go makeCall(clientArray[i], bh, h, turn, th, i, sChanW[i], Threads)

				bh += heights[i]
			}

			NewWorld := make([][]byte, ImageHeight)
			for i := range NewWorld {
				NewWorld[i] = make([]uint8, ImageWidth)
			}

			index := 0
			for i := 0; i < len(s.store); i++ {
				v := <-sChanW[i]
				for _, row := range v {
					NewWorld[index] = row
					index++
				}
			}
			turn++

			mutex_g.Lock()
			BrokerWorld = NewWorld
			cTurns = turn
			mutex_g.Unlock()

		}
	}

	res.Message = BrokerWorld

	return
}

// Quitting : used to quit all components
func (s *Broker) Quitting(req stubs.KPress, res *stubs.StringM) (err error) {
	if req.Status == true {

		// need to use the public address of aws
		//do a for loop through store to quit
		for i := 0; i < len(s.store); i++ {
			fmt.Println("quit")
			client, _ := rpc.Dial("tcp", s.store[i])
			kQuit := new(stubs.KQuit)
			kStatus := stubs.KQuit{Status: true}
			client.Call(stubs.Quitting, kStatus, kQuit)
		}

		res.Msg = "Quitting command has sent to factory"
		os.Exit(0)
	}

	return
}

// DataReceive : used to receive initial data from client
func (s *Broker) DataReceive(req stubs.Request, res *stubs.StringM) (err error) {
	if QState == false {
		fmt.Println("Here q false")
		BrokerWorld = req.OgWorld
		ImageHeight = req.ImageHeight
		ImageWidth = req.ImageWidth
		Threads = req.Threads
	}

	QState = false
	res.Msg = "world received from local machine"
	return

}

// GetStateB : used to send current data to the client
func (s *Broker) GetStateB(req stubs.Request, res *stubs.SavedC) (err error) {
	mutex_g.Lock()
	res.Turns = cTurns
	res.World = BrokerWorld
	mutex_g.Unlock()

	return
}

// Subscribe : used to accept any server to then send them work
func (s *Broker) Subscribe(req stubs.Subscription, res *stubs.StringM) (err error) {
	s.store = append(s.store, req.ServerAddress)
	if err != nil {
		res.Msg = "Error during subscription"
	}
	return
}

// QStatus : used to check if q has been pressed
func (s *Broker) QStatus(req stubs.StringM, res *stubs.QStatus) (err error) {
	res.Status = QState
	return
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rpc.Register(&Broker{store: []string{}})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
