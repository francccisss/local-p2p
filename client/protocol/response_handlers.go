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

func HandleLeechResponse(msg RPCMsgHeader, payload []byte, conn io.ReadWriteCloser, clt *ClusterTable) (ph PieceHeader, Error error) {

	h, err := ParsePieceHeader(&payload)
	if err != nil {
		fmt.Println("Error while parsing piece header")
		return PieceHeader{}, err
	}

	return h, nil
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

	// appends the node that returned
	(*clt)[cr.ClusterHash].AppendClusterPeer(ClusterPeer{Addr: NodeAddr{IP: msg.IP, Port: int(binary.LittleEndian.Uint32(msg.Port))}, NodeID: msg.NodeID, Conn: nil})
	for _, cpeer := range cr.Peers {
		(*clt)[cr.ClusterHash].AppendClusterPeer(cpeer)
	}
	return cr, nil
}

func HandleJoinClusterResponse(msg RPCMsgHeader, payload []byte, conn io.ReadWriteCloser, clt *ClusterTable) error {
	jsonRes := &JoinResponse{}
	err := json.Unmarshal(payload, jsonRes)
	if err != nil {
		return nil
	}
	_, ok := (*clt)[jsonRes.ClusterHash]
	// dont add clusters peers if node has not requested a search to a cluster
	// joining a cluster should only be done once the cluster is located between all
	// other nodes
	if !ok {
		return fmt.Errorf("Cluster %s does not exist\n", jsonRes.ClusterHash)
	}
	fmt.Printf("[REPLY]: - JOIN '%s' Accepted request to join '%s cluster'\n", jsonRes.NodeID, jsonRes.ClusterHash)
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
