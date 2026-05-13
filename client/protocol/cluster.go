package protocol

import (
	"context"
	"fmt"
	"io"
	"math"
	"time"
)

type ClusterName string

type ClusterPeerThread struct {
	TimeSince     time.Time
	NodeIDChann   chan NodeID
	ClusterName   ClusterName
	AverageBytes  int
	BytesReceived int
}

// calling Join() sets the TCP connection for the peers in the cluster
type ClusterPeer struct {
	Addr   NodeAddr
	NodeID NodeID
	Conn   io.ReadWriteCloser
	Status PeerStatus
}

type Cluster struct {
	ClusterPeerThreads map[NodeID]*ClusterPeerThread // keep track of peers
	ClusterName        ClusterName
	FileHash           string
	ClusterPeers       []ClusterPeer
	CurrentNode        Peer // created when a cluster is created
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

func CreateCluster(cname ClusterName) *Cluster {
	cpt := make(map[NodeID]*ClusterPeerThread)
	newCluster := Cluster{
		ClusterPeerThreads: cpt,
		ClusterPeers:       []ClusterPeer{},
		CurrentNode:        Peer{Status: IDLE, ClusterName: cname},
		ClusterName:        cname,
	}

	return &newCluster

}

func (cl *Cluster) NewClusterPeer(addr NodeAddr, nodeID NodeID, conn io.ReadWriteCloser, status PeerStatus) *ClusterPeer {

	return &ClusterPeer{
		Addr:   addr,
		NodeID: nodeID,
		Conn:   nil,
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

func (cl *Cluster) NewClusterPeerThread(nodeID NodeID) *ClusterPeerThread {
	newClusterPeerThread := &ClusterPeerThread{
		ClusterName: cl.ClusterName,
		NodeIDChann: make(chan NodeID),
		TimeSince:   time.Now(),
	}
	cl.ClusterPeerThreads[nodeID] = newClusterPeerThread
	return newClusterPeerThread
}

// will be received every reply to LEECH is received
// Use ctx to cancel when leeching is done
func (cl *Cluster) SpawnPeerThreads(ctx *context.Context, cpt *ClusterPeerThread) {
	// handle nil of cpt
	// if cpt == nil {
	// 	return fmt.Errorf("Peer Thread is nil")
	// }

	for {
		select {
		case <-(*ctx).Done():
			fmt.Println("TIMED OUT")
			fmt.Println("Cleaning Thread")
			return
			// clean up thread ORR ELSEEE!!!
		case nodeID := <-(*cpt).NodeIDChann:
			{
				fmt.Printf("Transfer by: %s\n", nodeID)

				currentTime := time.Now()
				elapsedms := currentTime.Sub(cpt.TimeSince)
				cpt.AverageBytes = cpt.BytesReceived / int(math.Max(1, float64(elapsedms.Seconds())))

				fmt.Printf("Time Elapsed: %fs\n", elapsedms.Seconds())
				fmt.Printf("Bytes Received: %d\n", cpt.BytesReceived)
				fmt.Printf("Average Bytes: %d\n", cpt.AverageBytes)
			}

		}
	}

}
