package protocol

import (
	"fmt"
	"io"
)

type ClusterName string

// calling Join() sets the TCP connection for the peers in the cluster
type ClusterPeer struct {
	Addr   NodeAddr
	NodeID NodeID
	Conn   io.ReadWriteCloser
	Status PeerStatus
}

type Cluster struct {
	ClusterName  ClusterName
	FileHash     string
	ClusterPeers []ClusterPeer
	CurrentNode  Peer // created when a cluster is created
}

type Peer struct {
	Status      PeerStatus
	ClusterName ClusterName
}

type ClusterTable map[ClusterName]*Cluster

func CreateClusterTable() *ClusterTable {
	clt := make(ClusterTable)
	return &clt

}

func CreateCluster(cname ClusterName, fileHash FileHash) *Cluster {
	newCluster := Cluster{
		ClusterPeers: []ClusterPeer{},
		CurrentNode:  Peer{Status: IDLE, ClusterName: cname},
		ClusterName:  cname,
		FileHash:     fileHash,
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
