package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

func HandlePingNodeResponse(msg RPCMsgHeader, payload []byte) PingResponse {
	fmt.Printf("Message: %s\n", msg.Message)
	return PingResponse(payload)
}

func HandlePingClusterResponse(msg RPCMsgHeader, payload []byte) PingResponse {
	fmt.Printf("Ping received from %s\n", msg.NodeID)
	return PingResponse(payload)
}
func HandleLeechResponse(msg RPCMsgHeader, payload []byte, conn io.ReadWriteCloser) error {
	return nil
	// // match the clustername and then the NodeID that sent the request
	// c, ok := (*clt)[seg.ClusterName]
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

// if a node contains the peers to a cluster, then that means that the current node is also
// included in that cluster as a peer
func HandleFindClusterResponse(msg RPCMsgHeader, payload []byte, clt *ClusterTable) (*ClusterResponse, error) {
	cr := &ClusterResponse{}
	err := json.Unmarshal(payload, cr)
	if err != nil {
		return nil, err
	}
	if msg.StatusCode == ERROR {
		rpcError := RPCErrorStr{ErrorMessage: msg.Message}
		return nil, rpcError
	}

	_, ok := (*clt)[cr.ClusterName]
	// it could be empty BUT the node that the request was sent to has sent an empty
	// list of peers which should mean that cluster does exist on the table
	if !ok {
		fmt.Printf("Creating new cluster: %s\n", cr.ClusterName)
		(*clt)[cr.ClusterName] = CreateCluster(cr.ClusterName, cr.FileHash)
	}

	// appends the node that returned
	(*clt)[cr.ClusterName].AppendClusterPeer(ClusterPeer{Addr: NodeAddr{IP: msg.IP, Port: int(binary.LittleEndian.Uint32(msg.Port))}, NodeID: msg.NodeID, Conn: nil})
	for _, cpeer := range cr.Peers {
		(*clt)[cr.ClusterName].AppendClusterPeer(cpeer)
	}
	return cr, nil
}

func HandleJoinClusterResponse(msg RPCMsgHeader, payload []byte, conn io.ReadWriteCloser, clt *ClusterTable) error {
	jsonRes := &JoinResponse{}
	err := json.Unmarshal(payload, jsonRes)
	if err != nil {
		return nil
	}
	_, ok := (*clt)[jsonRes.ClusterName]
	// dont add clusters peers if node has not requested a search to a cluster
	// joining a cluster should only be done once the cluster is located between all
	// other nodes
	if !ok {
		return fmt.Errorf("Cluster %s does not exist\n", jsonRes.ClusterName)
	}
	fmt.Printf("[REPLY]: - JOIN '%s' Accepted request to join '%s cluster'\n", jsonRes.NodeID, jsonRes.ClusterName)
	// cluster.AppendClusterPeer(ClusterPeer{Addr: NodeAddr{IP: msg.IP, Port: int(binary.LittleEndian.Uint32(msg.Port))}, NodeID: msg.NodeID, Conn: nil})

	return nil
}

func HandleProbeResponse(msg RPCMsgHeader, payload []byte) error {

	var fileMetaData ProbeReponse
	err := json.Unmarshal(payload, &fileMetaData)
	if err != nil {
		fmt.Println("Unable to Unmarshal FileMetaData")
		return err
	}

	fmt.Println("File Meta Data Received")
	fmt.Printf("File Meta Data: %+v\n", fileMetaData)
	return nil
}
