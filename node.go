package main

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

//For 15 nodes
var connectionTable = [][]int{
	[]int{1, 4, 7, 9, 11, 14},
	[]int{2, 4, 6, 8, 10, 13},
	[]int{0, 3, 5, 8, 11, 12},
	[]int{1, 4, 6, 7, 9, 13},
	[]int{2, 5, 7, 10, 12, 14},
	[]int{0, 3, 6, 8, 11, 14},
	[]int{0, 2, 4, 7, 9, 12},
	[]int{1, 5, 8, 10, 11, 13},
	[]int{0, 3, 6, 9, 12, 14},
	[]int{1, 2, 5, 7, 10, 13},
	[]int{0, 3, 6, 8, 11, 13},
	[]int{1, 3, 4, 8, 12, 14},
	[]int{0, 3, 5, 9, 10, 13},
	[]int{2, 4, 5, 6, 11, 14},
	[]int{1, 2, 7, 9, 10, 12},
}

//For 10 nodes
//var connectionTable = [][]int{
//	[]int{2, 5, 7, 9},
//	[]int{0, 3, 6, 8},
//	[]int{1, 4, 7, 9},
//	[]int{0, 2, 5, 8},
//	[]int{1, 3, 6, 9},
//	[]int{1, 4, 6, 8},
//	[]int{0, 2, 3, 7},
//	[]int{1, 4, 5, 8},
//	[]int{0, 2, 4, 9},
//	[]int{3, 5, 6, 7},
//}

//Node n
type Node struct {
	Mutex    sync.RWMutex
	ID       int64
	Peers    map[int64]*Peer
	PeerIds  []int64
	Listener net.Listener
	Chain    *Blockchain
	Pbft     *Pbft
	LastSlot int64
}

//PeerInfo read from config.json
type PeerInfo struct {
	ID          int64  `json:"id"`
	Destination string `json:"dest"`
}

var PeerInfos = make([]PeerInfo, 0)

func loadPeersAddressList() {
	type peerinfos struct {
		Peers []PeerInfo
	}

	var infos peerinfos
	fd, err := os.Open("./config.json")
	if err != nil {
		fmt.Println("[Open Failed]", err)
	}

	byteArray, err := ioutil.ReadAll(fd)
	if err != nil {
		fmt.Println("[ReadAll Failed]", err)
	}

	err = json.Unmarshal(byteArray, &infos)

	if err != nil {
		fmt.Println("[Unmarshal Failed]", err)
	}

	PeerInfos = append(PeerInfos, infos.Peers...)

	numberOfNodes = len(PeerInfos)
	numberOfDelegates = int64(numberOfNodes)
	maxFPNode = int64(math.Floor(float64((numberOfDelegates - 1) / 3)))
}

func init() {
	loadPeersAddressList()
}

func handleConnection(ctx context.Context, conn net.Conn, dec *gob.Decoder, node *Node) {
	//END_PEER_CONNECTION:
	for {
		var msg Message
		ReceiveMessage(&msg, dec)
		node.ProcessMessage(&msg, conn)
		//time.Sleep(time.Millisecond * time.Duration(rand.Intn(1000)))
		//select {
		//case <-ctx.Done():
		//	fmt.Println("Connect Down")
		//	break END_PEER_CONNECTION
		//default:
		//}
	}
}

func newServer(ctx context.Context, node *Node, port int64) net.Listener {
	listener, err := net.Listen("tcp", ":"+strconv.FormatInt(int64(port), 10))

	if err != nil {
		log.Println("NewServer Failed", node.ID, port)
	}

	go func(ctx context.Context, listener net.Listener) {
		conns := make([]net.Conn, 0)
	END_LISTENER:
		for {
			conn, err := listener.Accept()

			if err != nil {
				log.Println("Accept Failed", err)
			}

			conns = append(conns, conn)
			dec := gob.NewDecoder(conn)

			go handleConnection(ctx, conn, dec, node)

			fmt.Println("NodeId", node.ID, "Accepted")
			select {
			case <-ctx.Done():
				for _, v := range conns {
					v.Close()
				}
				listener.Close()
				fmt.Println("End all connections and listener")
				break END_LISTENER
			default:
			}

			//time.Sleep(time.Millisecond * 100)
		}
	}(ctx, listener)

	return listener

}

//NewNode create a new node
func NewNode(ctx context.Context, id int64, port int64) *Node {
	node := &Node{
		ID:      id,
		Peers:   make(map[int64]*Peer, 0),
		PeerIds: make([]int64, 0),
	}

	node.Listener = newServer(ctx, node, port)
	node.Chain = NewBlockchain(node)
	node.Pbft = NewPbft(node)

	return node
}

//Connect connect to peers
func (n *Node) Connect(ctx context.Context) {
	for i := 0; i < numberOfPeers; i++ {
		peerID := connectionTable[n.ID][i]
		peer := NewPeer(ctx, PeerInfos[peerID], n)
		n.Mutex.Lock()
		n.Peers[int64(peerID)] = peer
		n.Mutex.Unlock()
		n.PeerIds = append(n.PeerIds, int64(peerID))
	}
}

//func (n *Node) Connect(ctx context.Context) {
//	rand.Seed(time.Now().UnixNano())
//	i := 0
//	for {
//		rand := rand.Int63n(int64(len(PeerInfos)))
//
//		n.Mutex.RLock()
//		_, ok := n.Peers[rand]
//		n.Mutex.RUnlock()
//
//		if rand != n.ID && !ok {
//			peer := NewPeer(ctx, PeerInfos[rand], n)
//			n.Mutex.Lock()
//			n.Peers[rand] = peer
//			n.Mutex.Unlock()
//			n.PeerIds = append(n.PeerIds, rand)
//			i++
//		} else {
//			continue
//		}
//
//		fmt.Println("NodeID:", n.ID, "PeerID:", rand)
//		if i >= numberOfPeers {
//			break
//		}
//	}
//
//}

//StartForging start forging
func (n *Node) StartForging() {
	for {
		currentSlot := GetSlotNumber(0)
		lastBlock := n.Chain.GetLastBlock()
		lastSlot := GetSlotNumber(GetTime(lastBlock.GetTimestamp()))

		if currentSlot == lastSlot {
			time.Sleep(time.Millisecond * 100)
			continue
		}

		if currentSlot == n.LastSlot {
			time.Sleep(time.Millisecond * 100)
			continue
		}

		delegateID := currentSlot % numberOfDelegates

		if delegateID == n.ID {
			currentForger = n.ID
			newBlock := n.Chain.CreateBlock()

			n.Broadcast(BlockMessage( /*n.ID, */ *newBlock))
			n.Pbft.AddBlock(newBlock, GetSlotNumber(GetTime(newBlock.GetTimestamp())))

			fmt.Println("[NODE", n.ID, " NewBlock]", newBlock)
			n.LastSlot = currentSlot
		}

		time.Sleep(time.Second * 1)
	}
}

//Broadcast broadcast message to peers
func (n *Node) Broadcast(msg *Message) {
	for _, peer := range n.Peers {
		//fmt.Println("Forger ID:", currentForger, "NodeID", n.ID, "PeerID", peer.ID)
		SendMessage(msg, peer.ConnEncoder, n.ID)
	}
}

func (n *Node) handleInitMessage(msg *Message, conn net.Conn) {
	peerID := msg.Body.(int64)
	n.Mutex.RLock()
	_, ok := n.Peers[peerID]
	n.Mutex.RUnlock()
	if !ok {
		n.Mutex.Lock()
		n.Peers[peerID] = &Peer{
			ID:          peerID,
			NodeID:      n.ID,
			Conn:        conn,
			ConnEncoder: gob.NewEncoder(conn),
		}
		n.Mutex.Unlock()
	}
}

func (n *Node) handleBlockMessage(msg *Message) {
	block := msg.Body.(Block)
	n.Pbft.Mutex.Lock()
	_, ok := n.Pbft.PendingBlocks[block.GetHash()]
	n.Pbft.Mutex.Unlock()
	if !n.Chain.HasBlock(block.GetHash()) && !ok && n.Chain.ValidateBlock(&block) {
		n.Broadcast(msg)
		n.Pbft.AddBlock(&block, GetSlotNumber(GetTime(block.GetTimestamp())))
	}
}

//ProcessMessage process message from message
func (n *Node) ProcessMessage(msg *Message, conn net.Conn) {
	switch msg.Type {
	case MessageTypeInit:
		n.handleInitMessage(msg, conn)
	case MessageTypeBlock:
		n.handleBlockMessage(msg)
	default:
		n.Pbft.ProcessStageMessage(msg)
	}
}
