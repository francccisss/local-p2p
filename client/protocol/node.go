package protocol

import "net"

type PeerStatus int

const (
	LEECHING PeerStatus = iota
	SEEDING
	IDLE
)

type NodeID string

type NodeAddr struct {
	NodeID
	IP   []byte
	Port int
	Conn *net.TCPConn
}

type Node struct {
	NeighboringNodes []NodeAddr // used for bootstrapping
	NodeID           NodeID
	Addr             NodeAddr
	FILE_LOCATION    string
}

func NewNode(addr NodeAddr, nodeID NodeID, fileLoc string) *Node {
	return &Node{
		Addr:             addr,
		NodeID:           nodeID,
		FILE_LOCATION:    fileLoc,
		NeighboringNodes: make([]NodeAddr, 0, 10),
	}

}
