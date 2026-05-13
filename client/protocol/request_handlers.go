package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func HandleFindClusterRequest(newRPCMsg RPCMsg, msg RPCMsg, payload []byte, clt *ClusterTable) ([]byte, error) {
	plcname := ClusterRequest(payload)
	// will always send a string of cluster name to peer
	fmt.Printf("Checking cluster table for '%s' cluster\n", plcname)
	// it is always assumed that people that have the existing file should have an entry for cluster
	cl, ok := (*clt)[ClusterName(plcname)]
	// dont need to respond if does not exist anyways
	if !ok {
		return nil, fmt.Errorf("Cluster - %s does not exist", plcname)
	}

	fmt.Printf("Cluster '%s' exists\n", plcname)

	for _, cp := range cl.ClusterPeers {
		fmt.Printf("Peers in cluster: '%s'\n", cp.NodeID)
	}

	fmt.Printf("Sending reply back to '%s'\n", msg.NodeID)
	newResponse := ClusterResponse{
		ClusterName: cl.ClusterName,
		Peers:       cl.ClusterPeers,
	}
	b, err := json.Marshal(newResponse)
	if err != nil {
		return nil, err

	}
	buff, err := WrapPayloadToBuffer(newRPCMsg, b)
	if err != nil {
		return nil, err
	}
	return buff, nil
}

func HandlePingClusterRequest(newRPCMsg RPCMsg, msg RPCMsg, payload []byte) ([]byte, error) {

	newRPCMsg.Message = "Pong"
	pr := PingRequest(payload)
	fmt.Printf("[CALL]: PING - pinged by neighboring node %s\n", pr)

	var newPingResponse PingResponse = PingResponse(NodeStatusMap[ACTIVE]) // interesting unnessary type assertion
	b, err := WrapPayloadToBuffer(newRPCMsg, []byte(newPingResponse))
	if err != nil {
		return nil, err
	}
	return b, nil
}

func HandlePingNodeRequest(newRPCMsg RPCMsg, msg RPCMsg, payload []byte) ([]byte, error) {
	var newPingResponse PingResponse = PingResponse(NodeStatusMap[ACTIVE])
	newRPCMsg.Message = "Pong"
	b, err := WrapPayloadToBuffer(newRPCMsg, []byte(newPingResponse))
	if err != nil {
		return nil, err
	}
	return b, nil
}

func HandleLeechRequest(newRPCMsg RPCMsg, msg RPCMsg, payload []byte, conn io.ReadWriteCloser, FILE_LOCATION string) error {

	// a call to leech received a single SegmentHeader
	// a reply to leech receives an array of SegmentHeaders
	var fr FileRequest
	err := json.Unmarshal(payload, &fr)
	if err != nil {
		fmt.Println("Unable to unmarshal data segment")
		return err
	}
	en, path, err := Checkfile(fr.Hash, FILE_LOCATION)
	if err != nil {
		fmt.Println("Unable to unmarshal data segment")
		newRPCMsg.Message = err.Error()
		newRPCMsg.StatusCode = ERROR
		buff, err := WrapPayloadToBuffer(newRPCMsg, nil)
		if err != nil {
			fmt.Println("[ERROR]: Unable to send message")
		}
		msgErr := SendViaExistingConn(buff, conn)
		if msgErr != nil {
			fmt.Println("[ERROR]: Unable to send message")
		}
		return err
	}
	buf, _, err := CreateDataPiece(path+en.Name(), &fr)
	if err != nil {
		return err
	}
	b, err := WrapPayloadToBuffer(newRPCMsg, buf.Bytes())
	if err != nil {
		return err
	}
	err = SendViaExistingConn(b, conn)
	if err != nil {
		return err
	}
	return nil
}

func HandleJoinClusterRequest(newRPCMsg RPCMsg, msg RPCMsg, payload []byte, conn io.ReadWriteCloser, clt *ClusterTable) ([]byte, error) {
	clusterName := JoinRequest(payload)

	cl, ok := (*clt)[clusterName]
	if !ok {
		return nil, fmt.Errorf("'Cluster %s does not exist'\n", clusterName)
	}

	fmt.Printf("[CALL]: JOIN - 'Adding new peer to %s cluster\n", cl.ClusterName)
	ncp := cl.NewClusterPeer(NodeAddr{IP: msg.IP, Port: int(binary.LittleEndian.Uint32(msg.Port))}, msg.NodeID, conn, IDLE)
	err := cl.AppendClusterPeer(*ncp)
	if err != nil {
		return nil, err
	}

	newRPCMsg.StatusCode = SUCCESS

	joinResponse := JoinResponse{
		NodeID:      newRPCMsg.NodeID,
		Status:      cl.CurrentNode.Status,
		ClusterName: cl.ClusterName,
	}
	b, err := json.Marshal(joinResponse)
	if err != nil {
		return nil, err
	}

	buf, err := WrapPayloadToBuffer(newRPCMsg, b)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func HandleProbeRequest(newRPCMsg RPCMsg, msg RPCMsg, payload []byte, FILE_LOCATION string) ([]byte, error) {

	var preq ProbeRequest
	err := json.Unmarshal(payload, &preq)
	if err != nil {
		return nil, err
	}

	meta, err := ProbeFile(FILE_LOCATION, preq.FileHash)

	b, err := json.Marshal(meta)

	if err != nil {
		fmt.Println("Unable to Marshal FileMetaData")
		return nil, err
	}

	return b, nil
}

// TODO: add checksum parameter passed in by caller
// each file corresponds to a cluster name

func ProbeFile(FILE_LOCATION string, fileHash FileHash) (FileMetaData, error) {

	entry, path, err := Checkfile(fileHash, FILE_LOCATION)
	if err != nil {
		return FileMetaData{}, err
	}

	file, err := entry.Info()
	if err != nil {
		return FileMetaData{}, err
	}
	// obviously need to use the absolute route to the file
	// reuse wd prefix? hmmm
	fmt.Printf("Absolute Path: %s\n", path)
	fileBuffer, err := os.ReadFile(path + file.Name())

	if err != nil {
		return FileMetaData{}, err
	}

	fmt.Printf("file length: %d\n", len(fileBuffer))

	// check data integrity of file using checksum
	return FileMetaData{Name: file.Name(), Hash: fileHash, Size: uint64(file.Size())}, nil
}
