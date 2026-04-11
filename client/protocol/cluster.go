package protocol

import (
	"context"
	"fmt"
	"time"
)

type ClusterName string

type PeerThread struct {
	timeSince     time.Time
	NodeIDChann   chan NodeID
	ClusterName   ClusterName
	averageBytes  int
	bytesReceived int
}

type Cluster struct {
	PeerThreads map[NodeID]PeerThread // keep track of peers
	ClusterName ClusterName
	Peers       []Peer
}

// ClusterTable is used locally for handling threads spawned for different
// peers
type ClusterTable map[ClusterName]Cluster

func CreateCluster(clTable ClusterTable, cname ClusterName) {

	newCluster := Cluster{
		PeerThreads: make(map[NodeID]PeerThread, 10),
		Peers:       make([]Peer, 10),
		ClusterName: cname,
	}

	_, ok := clTable[cname]
	if !ok {
		clTable[cname] = newCluster

		fmt.Printf("New Cluster created for %s\n", cname)
		return
	}
	fmt.Printf("Cluster %s already exists\n", cname)

}

func NewPeerThread(cname ClusterName) PeerThread {
	return PeerThread{
		ClusterName: cname,
		NodeIDChann: make(chan NodeID),
		timeSince:   time.Now(),
	}
}

// will be received every reply to LEECH is received
// Use ctx to cancel when leeching is done
func MeasurePeerTransfer(ctx *context.Context, n *Node, threadTimer *PeerThread) {

	for {
		select {
		case <-(*ctx).Done():
			// clean up thread ORR ELSEEE!!!
			return
		case nodeID := <-(*threadTimer).NodeIDChann:
			{
				fmt.Printf("Transfer by: %s\n", nodeID)

				currentTime := time.Now()
				elapsedms := currentTime.Sub(threadTimer.timeSince)
				threadTimer.averageBytes = threadTimer.bytesReceived / int(elapsedms)

				fmt.Printf("Bytes transferred per second: %dms\n", threadTimer.averageBytes)
				threadTimer.bytesReceived = 0
			}

		}
	}

}
