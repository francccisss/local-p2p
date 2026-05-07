package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
)

// TODO: Maybe do a global cluster table instead of embedding the table in the Node itself
func (n *Node) HandleFindClusterRequest(newRPCMsg RPCMsg, msg RPCMsg, payload []byte, conn *net.Conn) error {
	plcname := ClusterRequest(payload)
	// will always send a string of cluster name to peer
	fmt.Printf("Checking cluster table for '%s' cluster\n", plcname)
	// it is always assumed that people that have the existing file should have an entry for cluster
	cl, ok := (*n.ClusterTable)[ClusterName(plcname)]
	// dont need to respond if does not exist anyways
	if !ok {
		fmt.Println("Cluster does not exist")
		newRPCMsg.StatusCode = ERROR
		newRPCMsg.Message = fmt.Sprintf("Cluster %s does not exist", plcname)
		b, err := WrapPayloadToBuffer(newRPCMsg, nil)
		err = SendViaExistingConn(b, conn)
		if err != nil {
			return err
		}
		return fmt.Errorf("Unable to deliver reply from FIND CLUSTER CALL")
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
		return err
	}
	buff, err := WrapPayloadToBuffer(newRPCMsg, b)
	if err != nil {
		return err
	}
	err = SendViaExistingConn(buff, conn)

	if err != nil {
		fmt.Println("Unable to respond to ping")
		return err
	}
	return nil
}

func HandlePingClusterRequest(newRPCMsg RPCMsg, msg RPCMsg, payload []byte, conn *net.Conn) error {

	// sender triggers a ping on receiver(this)
	// receiver sends their NodeID in return
	// so that the sender can keep track of the receivers
	newRPCMsg.Message = "Pong"
	pr := PingRequest(payload)
	fmt.Printf("[CALL]: PING - pinged by neighboring node %s\n", pr)

	var newPingResponse PingResponse = PingResponse(NodeStatusMap[ACTIVE]) // interesting unnessary type assertion

	b, err := WrapPayloadToBuffer(newRPCMsg, []byte(newPingResponse))
	if err != nil {
		return ProtocolErrorWrap(err.Error(), CALL, PING_CLUSTER)
	}
	err = SendViaExistingConn(b, conn)
	if err != nil {
		return ProtocolErrorWrap(err.Error(), CALL, PING_CLUSTER)
	}
	fmt.Println("Ping")
	return nil
}

func HandlePingNodeRequest(newRPCMsg RPCMsg, msg RPCMsg, payload []byte, conn *net.Conn) error {
	var newPingResponse PingResponse = PingResponse(NodeStatusMap[ACTIVE])
	newRPCMsg.Message = "Pong"
	fmt.Printf("Sending Ping reponse %s, in bytes %b\n", newPingResponse, []byte(newPingResponse))
	newRPCMsg.PayloadSize = uint32(len([]byte(newPingResponse)))
	b, err := WrapPayloadToBuffer(newRPCMsg, []byte(newPingResponse))
	if err != nil {
		return ProtocolErrorWrap(err.Error(), CALL, PING_NODE)
	}
	requstNodePort := binary.LittleEndian.Uint32(msg.Port)
	requestNodeAddr := NodeAddr{IP: msg.IP, Port: int(requstNodePort)}
	err = SendMsg(b, requestNodeAddr)
	if err != nil {
		return ProtocolErrorWrap(err.Error(), CALL, PING_NODE)
	}
	return nil
}

func (n *Node) HandleLeechRequest(newRPCMsg RPCMsg, msg RPCMsg, payload []byte, conn *net.Conn) error {

	// a call to leech received a single SegmentHeader
	// a reply to leech receives an array of SegmentHeaders
	var fr FileRequest
	err := json.Unmarshal(payload, &fr)
	if err != nil {
		fmt.Println("Unable to unmarshal data segment")
		return err
	}
	en, path, err := Checkfile(fr.Hash, n.FILE_LOCATION)
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
	buf, _, err := CreateDataSegment(path+en.Name(), &fr)
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

func (n *Node) HandleProbeRequest(newRPCMsg RPCMsg, msg RPCMsg, payload []byte, conn *net.Conn) error {

	cname := string(payload)
	status, meta, err := ProbeFile(n.FILE_LOCATION, ClusterName(cname))

	if err != nil {
		fmt.Printf("[PROBE REQUEST]: %s\n", err)
		newRPCMsg.Message = err.Error()
		newRPCMsg.StatusCode = status
		b, err := WrapPayloadToBuffer(newRPCMsg, nil)
		if err != nil {
			return fmt.Errorf("Unable to send message")
		}
		err = SendViaExistingConn(b, conn)
		if err != nil {
			return fmt.Errorf("Unable to send message")
		}
	}
	b, err := json.Marshal(meta)

	if err != nil {
		fmt.Println("Unable to Marshal FileMetaData")
		return err
	}

	newRPCMsg.StatusCode = SUCCESS
	buf, err := WrapPayloadToBuffer(newRPCMsg, b)
	if err != nil {
		return err
	}
	err = SendViaExistingConn(buf, conn)
	if err != nil {
		return err
	}
	return nil
}
