package protocol

import (
	"client/utils/bitfield"
	"fmt"
	"io"
	"math"
	"sync"
)

// calling Join() sets the TCP connection for the peers in the cluster
type ClusterPeer struct {
	Addr         NodeAddr
	NodeID       NodeID
	Conn         io.ReadWriteCloser
	Status       PeerStatus
	PeerBitField bitfield.BitField
}

type Cluster struct {
	ClusterHash  string
	ClusterPeers []ClusterPeer
	Node         CurrentNode // Represents the CurrentNode in this process
	// DHT already provides all that information
	FileMetaData FileMetaData // no need to send File Request == to File Meta Data
	BitField     bitfield.BitField
	WaitBuffer   []int // creates half of total pieces
	Window       SlidingWindow
	Mux          *sync.Mutex
}

type CurrentNode struct {
	Status      PeerStatus
	ClusterName string
}

type ClusterTable map[string]*Cluster

func CreateClusterTable() *ClusterTable {
	clt := make(ClusterTable)
	return &clt

}

func CreateCluster(fmd FileMetaData) *Cluster {
	newCluster := Cluster{
		ClusterPeers: []ClusterPeer{},
		Node:         CurrentNode{Status: IDLE, ClusterName: fmd.Hash},
		ClusterHash:  fmd.Hash,
		BitField:     bitfield.NewBitField(fmd.Pieces),
		FileMetaData: fmd,
		WaitBuffer:   make([]int, 0, int(math.Ceil(float64(fmd.Pieces)/float64(2)))),
		Mux:          new(sync.Mutex),
	}

	return &newCluster

}

func (cl *Cluster) NewClusterPeer(addr NodeAddr, nodeID NodeID, conn io.ReadWriteCloser, status PeerStatus) *ClusterPeer {

	return &ClusterPeer{
		Addr:   addr,
		NodeID: nodeID,
		Conn:   conn,
		Status: status,
	}
}

func (cl *Cluster) AppendClusterPeer(cpeer ClusterPeer) error {
	for _, cp := range cl.ClusterPeers {
		if cp.NodeID != cpeer.NodeID {
			continue
		}
		return fmt.Errorf("%s - Already exists\n", cpeer.NodeID)
	}
	cl.ClusterPeers = append(cl.ClusterPeers, cpeer)
	return nil
}
