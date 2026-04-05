package main

import (
	"encoding/json"
	"fmt"
	"net"
)

type PeerStatus int

const (
	Leeching PeerStatus = iota
	Seeding
	Idle
)

type Peer struct {
	Status PeerStatus
	IP     string
	Port   int
}
type NodeAddr struct {
	IP   string
	Port string
}

type Node struct {
	UDPconn   *net.UDPConn
	PeerTable []Peer
	Id        string // 16bit len
	Addr      NodeAddr
}

type ClientConn interface {
	Ping() error
	SetStatus() error
	Leech() error
	ReceiveSegment() (int, error)
}

type MsgType int

const (
	CALL MsgType = iota
	REPLY
)

// MsgType could be either reply or call
type RPCMsg struct {
	SegmentPosition int
	SegmentCount    int
	RPCType         MsgType
	NodeAddr        NodeAddr
	Body            []byte
}

func SendMsg(conn *net.UDPConn, message RPCMsg, peer *net.UDPAddr) error {
	b, err := json.Marshal(message)
	if err != nil {
		return err
	}
	n, err := conn.Write(b)
	if err != nil {
		return err
	}

	var msg RPCMsg
	err = json.Unmarshal(b, &msg)
	if err != nil {
		return err
	}
	fmt.Printf("\nRPC Message: %d\n", b)

	fmt.Printf("\nMarshalled: %d\nSent: %d\n", len(b), n)

	return nil
}

func (n *Node) Ping() error {

	var msg RPCMsg = RPCMsg{RPCType: CALL, NodeAddr: n.Addr}

	for i := 0; i < len(n.PeerTable); i++ {
		var p Peer = n.PeerTable[i]
		raddr := net.UDPAddr{
			Port: p.Port,
			IP:   []byte(p.IP),
		}
		SendMsg(n.UDPconn, msg, &raddr)
		if i == len(n.PeerTable) {
			break
		}
	}

	return nil
}

// buffer is the payload received from a peer
func ReadRPCMessage(buffer []byte) error {

	var msg RPCMsg
	err := json.Unmarshal(buffer, &msg)
	if err != nil {
		return err
	}
	fmt.Printf("Decoded RPC Message: %+v", msg)

	return nil
}

func (n *Node) CheckFile
