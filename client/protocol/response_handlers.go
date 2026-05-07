package protocol

import (
	"encoding/json"
	"fmt"
	"net"
)

func HandlePingNodeResponse(msg RPCMsg, payload []byte, conn *net.Conn) error {

	var pingMsg PingResponse = PingResponse(payload)
	fmt.Printf("Payload PingResponse-%s\n", pingMsg)
	fmt.Printf("Message: %s\n", msg.Message)
	(*conn).Close()
	return nil
}

func HandlePingClusterResponse(msg RPCMsg, payload []byte) error {
	// When receivers of the call responds/reply back to this
	// process, create a new cluster with name and initialize
	// pear threads and assign a peer thread that corresponds
	// with the receiver's NodeID that it send from PING

	fmt.Printf("Ping received from %s\n", msg.NodeID)
	var pingMsg PingResponse = PingResponse(payload)
	fmt.Println(pingMsg)
	fmt.Println("Pong")
	return nil
}
func HandleLeechResponse(msg RPCMsg, payload []byte, conn *net.Conn) error {
	return nil
	// // match the clustername and then the NodeID that sent the request
	// c, ok := (*n.ClusterTable)[seg.ClusterName]
	// if !ok {
	// 	return fmt.Errorf("Cluster does not exist")
	// }
	//
	// t, ok := c.ClusterPeerThreads[msg.NodeID]
	// if !ok {
	// 	return fmt.Errorf("NodeID Key does not exist for thread")
	// }
	// var ds SegmentHeader
	// err := json.Unmarshal(payload, &ds)
	// if err != nil {
	// 	return err
	// }
	// // t.BytesReceived += len(ds.DataChunk)
	// t.NodeIDChann <- msg.NodeID
	// // calculate segments received currently
	// if ds.SegmentPosition == ds.TotalSegments {
	// 	fmt.Println("WE ARE DONE BUCKO")
	// 	return nil
	// }
	// // read the file to create a FileRequest
	// // the file contains the meta data of the file that is to
	// // be leeched from other peers
	// // TODO: make sure that we're calling openfile once
	// // store it in memory so we dont have to use syscall
	// // everytime a leech reply is received which would take potentially take
	//
	// // TODO: For testing only remove this after testing the file transfer if it works
	// dfinfo := FileRequest{
	// 	Hash:      "IosevkaTerm.zip",
	// 	Size:      374780158,
	// 	Offset:    ds.SegmentPosition,
	// 	BlockSize: int64(os.Getpagesize()),
	// }
	// // TODO: For testing only remove this after testing the file transfer if it works
	//
	// err = n.Leech(ds.ClusterName, false, dfinfo)
	// if err != nil {
	// 	return err
	// }
	// return nil
}

func (n *Node) HandleFindClusterResponse(msg RPCMsg, payload []byte, conn *net.Conn) error {
	gcr := &ClusterResponse{}
	err := json.Unmarshal(payload, gcr)
	if err != nil {
		return err
	}
	if msg.StatusCode == ERROR {
		return fmt.Errorf("[REPLY]: FIND CLUSTER ERROR - %s", msg.Message)
	}

	cl, ok := (*n.ClusterTable)[gcr.ClusterName]
	if !ok {
		fmt.Printf("Cluster oes not exist creating local cluster for %s\n", gcr.ClusterName)
		(*n.ClusterTable)[gcr.ClusterName] = CreateCluster(gcr.ClusterName)
		cl = (*n.ClusterTable)[gcr.ClusterName]
	}
	for _, p := range gcr.Peers {
		cl.NewClusterPeer(p.Addr, p.NodeID)
	}
	fmt.Printf("[REPLY]: FIND CLUSTER - %d new peers added to %s cluster\n", len(cl.ClusterPeers), cl.ClusterName)
	return nil
}
func HandleProbeResponse(msg RPCMsg, payload []byte) error {

	var fileMetaData FileMetaData
	err := json.Unmarshal(payload, &fileMetaData)
	if err != nil {
		fmt.Println("Unable to Unmarshal FileMetaData")
		return err
	}

	fmt.Println("File Meta Data Received")
	fmt.Printf("File Meta Data: %+v\n", fileMetaData)
	return nil
}
