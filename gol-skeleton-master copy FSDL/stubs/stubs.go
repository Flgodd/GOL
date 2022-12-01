package stubs

import "uk.ac.bris.cs/gameoflife/util"

var GolHandler = "GameOperations.GolDist"
var Quitting = "GameOperations.Quitting"
var ChangeQ = "Broker.ChangeQ"
var QuittingB = "Broker.Quitting"
var GolCalc = "Broker.GolCalc"
var GetStateB = "Broker.GetStateB"
var DataReceive = "Broker.DataReceive"
var PressP = "Broker.PressP"
var Flipped = "Broker.Flipped"
var QStatusReq = "Broker.QStatus"
var Subscribe = "Broker.Subscribe"

type Response struct {
	Message     [][]byte
	CurrentTurn int
	AliveCount  int
}

type Request struct {
	NumTurns     int
	OgWorld      [][]byte
	ImageWidth   int
	ImageHeight  int
	CurrentWorld [][]byte
	checker      bool
	Threads      int
}

type SavedC struct {
	World       [][]byte
	Threads     int
	ImageWidth  int
	ImageHeight int
	Turns       int
}
type PPress struct {
	Status bool
}

type QPress struct {
	Status bool
}
type QStatus struct {
	Status bool
}
type KPress struct {
	Status bool
}
type KQuit struct {
	Status bool
}

type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

type StringM struct {
	Msg string
}

type BrokerRequest struct {
	H          int
	Bh         int
	Th         int
	Index      int
	World      [][]byte
	NumTurns   int
	TurnNumber int
}

type ToBrokerResponse struct {
	SliceWorld       [][]byte
	CellFlippedArray []util.Cell
}

type BrokerSavedState struct {
	BWorld [][]byte
	BTurns int
}

type FlippedRes struct {
	CellF     []util.Cell
	TurnsForF int
}

type Subscription struct {
	Topic         string
	ServerAddress string
	Callback      string
}
