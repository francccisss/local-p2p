package main

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync"
)

type Method int

const (
	PING Method = iota
	LEECH
)

// Node can implement client connection interface
// this interface describes the characteristics of
// a Node in a cluster that makes it able to communicate with its peers

type ClientConn interface {
	Ping() error
	Leech() error
	CheckFile() (StatusCode, error) // checks for file existence also should do checksum for data integrity before transfering for security reasons
	RecvRPCMessage(msg RPCMsg) error
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

func RecvRPCMessage(n *Node, msg RPCMsg) error {

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

	return fmt.Errorf("Unable to receive rpc requests.")
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

func ProbeFile(n *Node, fileKey string) (StatusCode, error) {

	return SUCCESS, nil
}

func Ping(n *Node, wg *sync.WaitGroup) error {

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
