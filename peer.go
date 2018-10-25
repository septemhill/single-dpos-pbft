package main

import (
	"context"
	"encoding/gob"
	"log"
	"net"
)

//Peer p
type Peer struct {
	ID          int64
	NodeID      int64
	Conn        net.Conn
	ConnEncoder *gob.Encoder
}

//NewPeer create a new peer
func NewPeer(ctx context.Context, peerInfo PeerInfo, node *Node) *Peer {
	conn, err := net.Dial("tcp", peerInfo.Destination)
	if err != nil {
		log.Println("[Dial Failed]", err)
	}

	peer := &Peer{
		ID:          peerInfo.ID,
		NodeID:      node.ID,
		Conn:        conn,
		ConnEncoder: gob.NewEncoder(conn),
	}

	go handleConnection(ctx, conn, gob.NewDecoder(conn), node)

	SendMessage(InitMessage(node.ID), peer.ConnEncoder, node.ID)

	return peer
}
