package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type ClusterName string

type ClusterPeerThread struct {
	timeSince     time.Time
	NodeIDChann   chan NodeID
	ClusterName   ClusterName
	averageBytes  int
	bytesReceived int
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

func (cl *Cluster) NewClusterPeer(addr NodeAddr, nodeID NodeID) *ClusterPeer {
	newClusterPeer := &ClusterPeer{
		Addr:   addr,
		NodeID: nodeID,
	}

	cl.ClusterPeers = append(cl.ClusterPeers, *newClusterPeer)
	return newClusterPeer
}

type ClusterTable map[ClusterName]*Cluster

func CreateCluster(n *Node, cname ClusterName) {

	newCluster := Cluster{
		ClusterPeerThreads: make(map[NodeID]*ClusterPeerThread),
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

func (cl *Cluster) NewClusterPeerThread(nodeID NodeID) *ClusterPeerThread {

	newClusterPeerThread := &ClusterPeerThread{
		ClusterName: cl.ClusterName,
		NodeIDChann: make(chan NodeID),
		timeSince:   time.Now(),
	}
	cl.ClusterPeerThreads[nodeID] = newClusterPeerThread
	return newClusterPeerThread
}

// will be received every reply to LEECH is received
// Use ctx to cancel when leeching is done
func (cl *Cluster) MeasurePeerTransfer(ctx *context.Context, n *Node, clPeerThread *ClusterPeerThread) {

	for {
		select {
		case <-(*ctx).Done():
			fmt.Println("TIMED OUT")
			// clean up thread ORR ELSEEE!!!
			return
		case nodeID := <-(*clPeerThread).NodeIDChann:
			{
				fmt.Printf("Transfer by: %s\n", nodeID)

				currentTime := time.Now()
				elapsedms := currentTime.Sub(clPeerThread.timeSince)
				clPeerThread.averageBytes = clPeerThread.bytesReceived / int(elapsedms)

				fmt.Printf("Bytes transferred per second: %dms\n", clPeerThread.averageBytes)
				clPeerThread.bytesReceived = 0
			}

		}
	}

}

// -----------------------------------------
// METHODS FOR HANDLING A `CALL` RPC MESSAGE
// -----------------------------------------

// TODO add checksum parameter passed in by caller
// each file corresponds to a cluster name

func ProbeFile(FILE_LOCATION string, cname ClusterName) (StatusCode, FileMetaData, error) {

	entry, path, err := Checkfile(string(cname), FILE_LOCATION)
	if err != nil {
		return ERROR, FileMetaData{}, err
	}

	file, err := entry.Info()
	if err != nil {
		return ERROR, FileMetaData{}, err
	}
	// obviously need to use the absolute route to the file
	// reuse wd prefix? hmmm
	fmt.Printf("Absolute Path: %s\n", path)
	fileBuffer, err := os.ReadFile(path + file.Name())

	if err != nil {
		return ERROR, FileMetaData{}, err
	}

	fmt.Printf("file length: %d\n", len(fileBuffer))

	// check data integrity of file using checksum
	return SUCCESS, FileMetaData{Name: file.Name(), Hash: string(cname), Size: uint64(file.Size())}, nil
}

// -----------------------------------------
// METHODS FOR CREATING A `CALL` RPC MESSAGE
// -----------------------------------------

// n asks what other peers if they have this p.cname
// in their table, if so, they add this node and set the status to idle.
// This function can be used within a cluster, if passed in the peers of that cluster
// or the neighboring nodes for creating a cluster table for p.cname in the sender process
func Ping(n *Node, cname ClusterName) error {

	var msg RPCMsg = RPCMsg{
		RPCType:    CALL,
		NodeAddr:   n.Addr,
		Method:     PING,
		StatusCode: SUCCESS,
		NodeID:     n.NodeID,
	}

	// for bootstrapped nodes
	c, ok := (*n.ClusterTable)[cname]
	if !ok {
		return fmt.Errorf("ERROR: Cluster not found")
	}
	fmt.Println(len(c.ClusterPeers))
	for _, p := range c.ClusterPeers {
		fmt.Printf("\nPEER: %+v\n", p)
		newPingMsg := PingMessage{ClusterName: cname, Status: IDLE}

		b, err := json.Marshal(newPingMsg)
		if err != nil {
			return err
		}

		msg.Payload = b
		err = SendMsg(n.UDPconn, msg, p.Addr)
		if err != nil {
			fmt.Printf("%s", err)
			continue
		}
	}

	fmt.Println("\nPinging peers in cluster.")
	fmt.Println("Ping Sent")
	return nil
}

func Probe(n *Node, cname ClusterName) error {
	c, ok := (*n.ClusterTable)[cname]
	if !ok {
		return fmt.Errorf("Cluster does not exist")
	}
	newMsg := RPCMsg{
		RPCType:  CALL,
		Method:   PROBE,
		NodeID:   n.NodeID,
		NodeAddr: n.Addr,
		Payload:  []byte(cname),
	}

	for _, cp := range c.ClusterPeers {
		err := SendMsg(n.UDPconn, newMsg, cp.Addr)
		if err != nil {
			continue
		}
	}

	return nil

}

func Leech(n *Node, cname ClusterName) error {

	c, ok := (*n.ClusterTable)[cname]
	if !ok {
		return fmt.Errorf("Cluster does not exist")
	}

	c.Peer.Status = LEECHING

	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)

	for _, p := range c.ClusterPeers {
		go c.MeasurePeerTransfer(&ctx, n, c.ClusterPeerThreads[p.NodeID])
	}

	return nil
}
