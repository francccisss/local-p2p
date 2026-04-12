package protocol

import (
	"context"
	"fmt"
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

type ClusterPeer struct {
	Addr   NodeAddr
	NodeID NodeID
}
type Cluster struct {
	ClusterPeerThreads map[NodeID]*ClusterPeerThread // keep track of peers
	ClusterName        ClusterName
	ClusterPeers       []ClusterPeer
	Peer               Peer // created when a cluster is created
}

type Peer struct {
	Status      PeerStatus
	ClusterName ClusterName
}

type ClusterTable map[ClusterName]*Cluster

func CreateCluster(n *Node, cname ClusterName) {
	cpt := make(map[NodeID]*ClusterPeerThread)
	newCluster := Cluster{
		ClusterPeerThreads: cpt,
		ClusterPeers:       []ClusterPeer{},
		Peer:               Peer{Status: IDLE, ClusterName: cname},
		ClusterName:        cname,
	}

	_, ok := (*n.ClusterTable)[cname]
	if !ok {
		(*n.ClusterTable)[cname] = &newCluster
		fmt.Printf("New Cluster created for '%s'\n", cname)
		return
	}
	fmt.Printf("Cluster '%s' already exists\n", cname)

}

func (cl *Cluster) NewClusterPeer(addr NodeAddr, nodeID NodeID) *ClusterPeer {
	newClusterPeer := &ClusterPeer{
		Addr:   addr,
		NodeID: nodeID,
	}

	cl.ClusterPeers = append(cl.ClusterPeers, *newClusterPeer)
	return newClusterPeer
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

	for {
		select {
		case <-(*ctx).Done():
			fmt.Println("TIMED OUT")
			fmt.Println("Cleaning Thread")
			// clean up thread ORR ELSEEE!!!
			return
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
