package protocol

import (
	"fmt"
	"net"
	"strconv"
)

type PeerStatus int

const (
	LEECHING PeerStatus = iota
	SEEDING
	IDLE
)

type NodeID string

type NodeAddr struct {
	IP   []byte
	Port int
}

type Node struct {
	UDPconn          *net.UDPConn
	NeighboringNodes []NodeAddr // used for bootstrapping
	NodeID           NodeID
	Addr             NodeAddr
	FILE_LOCATION    string
}

func NewNode(conn *net.UDPConn, addr NodeAddr, nodeID NodeID, fileLoc string) *Node {
	return &Node{
		UDPconn:          conn,
		Addr:             addr,
		NodeID:           nodeID,
		FILE_LOCATION:    fileLoc,
		NeighboringNodes: make([]NodeAddr, 10),
	}

}

func InitUDPConn(port int) (*net.UDPConn, error) {

	laddr, err := net.ResolveUDPAddr("udp", "localhost:"+strconv.Itoa(port))
	if err != nil {
		fmt.Println("Error Unable to resolve UDP Addr")
		return nil, err
	}
	UDPConn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		fmt.Println("Error Unable to create a listener for UDP packets")
		return nil, err
	}
	return UDPConn, nil

}
