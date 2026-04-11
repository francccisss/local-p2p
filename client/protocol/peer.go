package protocol

import (
	"encoding/json"
	"fmt"
	"os"
)

type Peer struct {
	Status      PeerStatus
	ClusterName ClusterName
}

func NewPeer(cname ClusterName) *Peer {

	return &Peer{
		Status:      IDLE,
		ClusterName: cname,
	}

}

// TODO add checksum parameter passed in by caller
func (p *Peer) ProbeFile(n *Node) (StatusCode, error) {

	entry, path, err := Checkfile(string(p.ClusterName), n.FILE_LOCATION)
	if err != nil {
		return ERROR, err
	}

	file, err := entry.Info()
	if err != nil {
		return ERROR, err
	}
	// obviously need to use the absolute route to the file
	// reuse wd prefix? hmmm
	fmt.Printf("Absolute Path: %s\n", path)
	fileBuffer, err := os.ReadFile(path + file.Name())

	if err != nil {
		return ERROR, err
	}

	fmt.Printf("file length: %d\n", len(fileBuffer))

	// check data integrity of file using checksum

	return SUCCESS, nil
}

// METHODS FOR CREATING A `CALL` RPC MESSAGE

// n asks what other peers if they have this p.cname
// in their table, if so, they add this node and set the status to idle.
// This function can be used within a cluster, if passed in the peers of that cluster
// or the neighboring nodes for creating a cluster table for p.cname in the sender process
func (p *Peer) Ping(n *Node, clTable ClusterTable) error {

	var msg RPCMsg = RPCMsg{
		RPCType:    CALL,
		NodeAddr:   n.Addr,
		Method:     PING,
		StatusCode: SUCCESS,
	}

	c, ok := clTable[p.ClusterName]
	if !ok {
		return fmt.Errorf("ERROR: Cluster not found")
	}
	peers := c.Peers
	fmt.Println(len(peers))
	for _, p := range peers {
		fmt.Printf("\nPEER: %+v\n", p)
		newPingMsg := PingMessage{ClusterName: p.ClusterName, Status: IDLE}

		b, err := json.Marshal(newPingMsg)
		if err != nil {
			return err
		}

		msg.Payload = b
		err = SendMsg(n.UDPconn, msg, n.Addr)
		if err != nil {
			fmt.Printf("%s", err)
			continue
		}
	}

	fmt.Println("\nPinging peers in cluster.")
	fmt.Println("Ping Sent")
	return nil
}

func (p *Peer) Leech(n *Node, clTable ClusterTable) error {

	c, ok := clTable[p.ClusterName]
	if !ok {
		return fmt.Errorf("Cluster does not exist")
	}

	p.Status = LEECHING
	clTable[p.ClusterName] = c
	return nil
}
