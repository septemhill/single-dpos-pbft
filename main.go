package main

import (
	"context"
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	//numberOfPeers int = 6 //For 15 nodes
	numberOfPeers    int   = 4 //For 10 nodes
	slotTimeInterval int64 = 3
)

var (
	currentForger     int64 = -1
	numberOfNodes           = 0
	numberOfDelegates       = int64(0)
	maxFPNode               = int64(0)
)

func gobInterfaceRegister() {
	gob.Register(Block{})
	gob.Register(Transaction{})
	gob.Register(StageMessage{})
}

func init() {
	gobInterfaceRegister()
}

func main() {
	var nodeID, port int64
	ctx, cancel := context.WithCancel(context.Background())
	sysdone := make(chan struct{}, 1)
	sigCh := make(chan os.Signal, 1)

	flag.Int64Var(&nodeID, "i", 0, "Setup ID for current node")
	flag.Int64Var(&port, "p", 10000, "Setup port for current node")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		sysdone <- struct{}{}
	}()

	//Create New Node
	node := NewNode(ctx, nodeID, port)
	time.Sleep(time.Second * 5)

	//Connect to Peers
	node.Connect(ctx)
	time.Sleep(time.Second * 5)
	fmt.Println("Node", node.ID, "Peers", node.Peers)

	//Start DPOS Forging
	node.StartForging()

	<-sysdone
	cancel()
}
