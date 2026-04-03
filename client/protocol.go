package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync"
)

type PeerStatus int

const (
	LEECHING PeerStatus = iota
	SEEDING
	IDLE
)

type Method int

const (
	PING Method = iota
	LEECH
)

type Peer struct {
	PStatus  PeerStatus
	LStatus  PeerStatus
	NodeAddr NodeAddr
}
type NodeAddr struct {
	IP   []byte
	Port int
}

type Node struct {
	UDPconn   *net.UDPConn
	PeerTable []Peer // TODO Change to map with array of Peer, key is the hash value of the file that is being transffered in the cluster
	Id        string // 16bit len
	Addr      NodeAddr
}

type ClientConn interface {
	Ping() error
	SetStatus() error // setting internal status
	Leech() error
}

type MsgType int

const (
	CALL MsgType = iota
	REPLY
)

type StatusCode int

const (
	SUCCESS StatusCode = iota
	ERROR
)

type BodyMsg struct {
}

// MsgType could be either reply or call
type RPCMsg struct {
	SegmentPosition int
	SegmentCount    int
	RPCType         MsgType
	NodeAddr        NodeAddr
	Method          Method
	Payload         []byte
	StatusCode      StatusCode
	Comment         string
}

func SendMsg(conn *net.UDPConn, message RPCMsg, peerAddr NodeAddr) error {
	b, err := json.Marshal(message)
	if err != nil {
		return err
	}

	ip := string(peerAddr.IP)
	port := strconv.Itoa(peerAddr.Port)
	raddr, err := net.ResolveUDPAddr("udp", ip+":"+port)

	if err != nil {
		return err
	}

	fmt.Printf("\nSend to: %s\n", ip+":"+port)
	n, err := conn.WriteTo(b, raddr)
	if err != nil {
		return err
	}

	fmt.Printf("Marshalled: %d\nSent: %d\n", len(b), n)

	return nil
}

func (n *Node) Ping(wg *sync.WaitGroup) error {

	var msg RPCMsg = RPCMsg{
		RPCType:    CALL,
		NodeAddr:   n.Addr,
		Method:     PING,
		Payload:    []byte("Ping"),
		StatusCode: SUCCESS,
	}

	for i := 0; i < len(n.PeerTable); i++ {
		var p Peer = n.PeerTable[i]
		fmt.Printf("\nPEER: %+v\n", p)
		SendMsg(n.UDPconn, msg, p.NodeAddr)
	}

	fmt.Println("\nPinging peers in cluster.")
	fmt.Println("Ping Sent")
	return nil
}

func (n *Node) RecvRPCMessage(msg RPCMsg) {

	var newRPCMsg RPCMsg
	switch msg.RPCType {
	case CALL:
		fmt.Println("Call Message")
		switch msg.Method {
		case PING:
			// respond to ping
			newRPCMsg = RPCMsg{
				Method:     PING,
				RPCType:    CALL,
				Comment:    "",
				StatusCode: SUCCESS,
				NodeAddr:   NodeAddr{IP: n.Addr.IP, Port: n.Addr.Port},
				Payload:    []byte("Pong"),
			}
			SendMsg(n.UDPconn, newRPCMsg, msg.NodeAddr)
		case LEECH:

		}

	case REPLY:
		fmt.Println("Reply from Call Message")
	default:

	}

}

// buffer is the payload received from a peer
func ReadRPCMessage(buffer []byte) (RPCMsg, error) {

	var msg RPCMsg
	err := json.Unmarshal(buffer, &msg)
	if err != nil {
		return RPCMsg{}, err
	}
	return msg, nil
}
