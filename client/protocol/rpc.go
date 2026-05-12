package protocol

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// buffer is the payload received from a peer
func ReadRPCMessage(buffer []byte) (RPCMsg, error) {

	var msg RPCMsg
	err := json.Unmarshal(buffer, &msg)
	if err != nil {
		fmt.Println("Unable to Unmarshal")
		return RPCMsg{}, err
	}
	return msg, nil
}

func (n *Node) RecvRPCMessage(msg RPCMsg, payload []byte, conn io.ReadWriteCloser, clt *ClusterTable) error {
	if msg.StatusCode == ERROR {
		return ProtocolErrorWrap(msg.Message, msg.RPCType, msg.Method)
	}

	switch msg.RPCType {

	case CALL: // when peers/nodes send a call RPCType

		var newRPCMsg RPCMsg
		fmt.Println("Call Message")

		newRPCMsg = RPCMsg{
			Method:      msg.Method,
			RPCType:     REPLY,
			NodeID:      n.NodeID,
			IP:          n.Addr.IP,
			Message:     "",
			StatusCode:  SUCCESS,
			PayloadSize: 0,
			Port:        make([]byte, 4),
		}
		// setting requesting node's port to host byte ordering
		binary.LittleEndian.PutUint32(newRPCMsg.Port, uint32(n.Addr.Port))
		switch msg.Method {

		case PING_NODE:
			requstNodePort := binary.LittleEndian.Uint32(msg.Port)
			requestNodeAddr := NodeAddr{IP: msg.IP, Port: int(requstNodePort)}
			b, err := HandlePingNodeRequest(newRPCMsg, msg, payload)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), CALL, PING_NODE)
			}
			err = SendMsg(b, requestNodeAddr)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), CALL, PING_NODE)
			}

		case PING_CLUSTER:
			b, err := HandlePingClusterRequest(newRPCMsg, msg, payload)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), CALL, PING_CLUSTER)
			}
			err = SendViaExistingConn(b, conn)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), CALL, PING_CLUSTER)
			}

		case FIND_CLUSTER:
			b, err := HandleFindClusterRequest(newRPCMsg, msg, payload, clt)
			// Sends an error that a cluster does not exist
			if err != nil {
				newRPCMsg.StatusCode = ERROR
				newRPCMsg.Message = err.Error()
				b, err := WrapPayloadToBuffer(newRPCMsg, nil)
				if err != nil {
					return ProtocolErrorWrap(err.Error(), CALL, FIND_CLUSTER)
				}
				err = SendViaExistingConn(b, conn)
				if err != nil {
					return ProtocolErrorWrap(err.Error(), CALL, FIND_CLUSTER)
				}
				return nil
			}

			err = SendViaExistingConn(b, conn)
			if err != nil {
				fmt.Println("Unable to respond to ping")
				return ProtocolErrorWrap(err.Error(), CALL, FIND_CLUSTER)
			}
		case JOIN:
			b, err := HandleJoinClusterRequest(newRPCMsg, msg, payload, conn, clt)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), CALL, JOIN)
			}
			err = SendViaExistingConn(b, conn)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), CALL, JOIN)
			}
			return nil

		case PROBE:
			return HandleProbeRequest(newRPCMsg, msg, payload, conn, n.FILE_LOCATION)

		case LEECH:
			return HandleLeechRequest(newRPCMsg, msg, payload, conn, n.FILE_LOCATION)
		}

	case REPLY: // when peers/nodes send a call RPCType

		if msg.StatusCode == ERROR {
			return ProtocolErrorWrap(msg.Message, REPLY, msg.Method)
		}

		switch msg.Method {

		case PING_NODE:
			pingResponse := HandlePingNodeResponse(msg, payload)
			fmt.Printf("Retrived Ping reponse %+v", pingResponse)
			conn.Close()
		case PING_CLUSTER:
			pingResponse := HandlePingClusterResponse(msg, payload)
			fmt.Printf("Retrived Ping reponse %+v", pingResponse)
		case FIND_CLUSTER:
			cr, err := HandleFindClusterResponse(msg, payload, clt)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), REPLY, FIND_CLUSTER)
			}
			cl, ok := (*clt)[cr.ClusterName]
			if !ok {
				fmt.Printf("Cluster does not exist creating local cluster for %s\n", cr.ClusterName)
				(*clt)[cr.ClusterName] = CreateCluster(cr.ClusterName)
				cl = (*clt)[cr.ClusterName]
			}
			for _, p := range cr.Peers {
				// populate the conn ONLY WHEN JOIN is called and accepted for each peer in cluster
				cl.NewClusterPeer(p.Addr, p.NodeID, nil, p.Status)
			}
			fmt.Printf("[REPLY]: FIND CLUSTER - %d new peers added to %s cluster\n", len(cl.ClusterPeers), cl.ClusterName)
		case JOIN:
			fmt.Println("HANDLING")
			err := HandleJoinClusterResponse(msg, payload, conn, clt)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), CALL, JOIN)
			}

		case LEECH:
			return HandleLeechResponse(msg, payload, conn)

		case PROBE:
			return HandleProbeResponse(msg, payload)

		}
	}
	return nil
}

// ----------------------
// METHODS FOR RPC CALLS
// ----------------------

func (n *Node) FindCluster(cname ClusterName) error {

	var wg sync.WaitGroup

	payload := ClusterRequest(cname)
	newMsg := RPCMsg{
		NodeID:  n.NodeID,
		Message: "where cluster?",
		Method:  FIND_CLUSTER,
		RPCType: CALL,
		IP:      n.Addr.IP,
		Port:    make([]byte, 4),
	}
	binary.LittleEndian.PutUint32(newMsg.Port, uint32(n.Addr.Port))

	b, err := WrapPayloadToBuffer(newMsg, []byte(payload))
	if err != nil {
		return err
	}

	fmt.Printf("DOUBLE CHECK BUFFER")
	for _, n := range n.NeighboringNodes {
		wg.Go(func() {
			// TODO: need to dial NeighboringNodes instead of communicating in an open channel
			// BRUUUHHH
			// conn, err := net.Dial("tcp", string(n.IP)+":"+strconv.Itoa(n.Port))
			// if err != nil {
			// 	fmt.Printf("FIND CLUSTER ERROR: %s", n.NodeID)
			// 	fmt.Println(err)
			// 	return
			// }
			_, err := n.Conn.Write(b)
			if err != nil {
				fmt.Printf("FIND CLUSTER ERROR: %s", n.NodeID)
				fmt.Println(err)
			}
		})
	}
	wg.Wait()
	fmt.Println("Sent FIND CLUSTER request to neighboring nodes")

	return nil
}

// The difference between PingCluster & PingNodes is pretty self explanatory
// instead of pinging a cluster and also the function uses Dial instead of
// the existing TCP connection
func (n *Node) PingNodes() error {

	var newMsg RPCMsg = RPCMsg{
		RPCType:    CALL,
		IP:         n.Addr.IP,
		Method:     PING_NODE,
		StatusCode: SUCCESS,
		NodeID:     n.NodeID,
		Port:       make([]byte, 4),
	}
	binary.LittleEndian.PutUint32(newMsg.Port, uint32(n.Addr.Port))
	pingRequest := []byte("Ping")
	newMsg.PayloadSize = uint32(len(pingRequest))
	buf, err := WrapPayloadToBuffer(newMsg, pingRequest)
	for _, neighbor := range n.NeighboringNodes {
		fmt.Printf("\nNEIGHBOR: %+v\n", neighbor)

		if err != nil {
			fmt.Printf("%s", err)
			continue
		}
		err = SendMsg(buf, neighbor)
		if err != nil {
			fmt.Printf("%s", err)
			continue
		}
	}
	return nil
}

// Pinging cluster uses the existing TCP connections for communication
func (n *Node) PingCluster(cname ClusterName, clt *ClusterTable) error {

	var newMsg RPCMsg = RPCMsg{
		RPCType:    CALL,
		Method:     PING_CLUSTER,
		StatusCode: SUCCESS,
		NodeID:     n.NodeID,
		IP:         n.Addr.IP,
		Port:       make([]byte, 4),
	}

	binary.LittleEndian.PutUint32(newMsg.Port, uint32(n.Addr.Port))

	// for bootstrapped nodes
	c, ok := (*clt)[cname]
	if !ok {
		return fmt.Errorf("Cluster not found")
	}
	pingRequest := []byte("Ping")
	for _, p := range c.ClusterPeers {
		fmt.Printf("\nPEER: %+v\n", p)

		buf, err := WrapPayloadToBuffer(newMsg, pingRequest)
		if err != nil {
			fmt.Printf("%s", err)
			continue
		}
		_, err = p.Conn.Write(buf)
		if err != nil {
			fmt.Printf("%s", err)
			continue
		}
	}
	return nil
}

func (n *Node) Probe(cname ClusterName, clt *ClusterTable) error {
	c, ok := (*clt)[cname]
	if !ok {
		return fmt.Errorf("Cluster does not exist")
	}
	newMsg := RPCMsg{
		RPCType: CALL,
		Method:  PROBE,
		NodeID:  n.NodeID,
		IP:      n.Addr.IP,
	}
	binary.LittleEndian.PutUint32(newMsg.Port, uint32(n.Addr.Port))

	for _, cp := range c.ClusterPeers {

		buf, err := WrapPayloadToBuffer(newMsg, []byte(cname))
		if err != nil {
			continue
		}
		err = SendMsg(buf, cp.Addr)
		if err != nil {
			continue
		}
	}

	return nil

}

func (n *Node) JoinCluster(cname ClusterName, clt *ClusterTable) error {

	newMsg := RPCMsg{
		NodeID:  n.NodeID,
		IP:      n.Addr.IP,
		Port:    make([]byte, 4),
		Message: "Joining cluster",
		Method:  JOIN,
		RPCType: CALL,
	}

	binary.LittleEndian.PutUint32(newMsg.Port, uint32(n.Addr.Port))

	ct, ok := (*clt)[cname]

	if !ok {
		return fmt.Errorf("[INVOKE]: - JOIN CLUSTER 'Cluster %s does not exist'\n", cname)
	}

	joinReq := ct.ClusterName

	b, err := WrapPayloadToBuffer(newMsg, []byte(joinReq))

	if err != nil {
		return err
	}

	// TODO: Need to dial instead of using tcp write
	// TODO: Change this afterwards
	for _, nc := range ct.ClusterPeers {
		_, err := nc.Conn.Write(b)
		if err != nil {
			continue
		}
	}

	return nil
}

func (n *Node) Leech(cname ClusterName, spawnThreads bool, fr FileRequest, clt *ClusterTable) error {

	c, ok := (*clt)[cname]
	if !ok {
		return fmt.Errorf("Cluster does not exist")
	}

	c.CurrentNode.Status = LEECHING

	if spawnThreads {
		var wg sync.WaitGroup
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
		fmt.Println("[LEECH REQUEST]: Preparing Cluster Peer Threads for Byte Measurement")
		defer cancel()

		for _, p := range c.ClusterPeers {
			wg.Go(func() {
				go c.SpawnPeerThreads(&ctx, c.ClusterPeerThreads[p.NodeID])
			})
		}
		fmt.Println("[LEECH REQUEST]: Waiting for Cluster Peer Threads to deploy...")
		wg.Wait()
		fmt.Println("[LEECH REQUEST]: Done!")
	}
	fmt.Printf("[LEECH REQUEST]: Sending request to peers in cluster: %s\n", cname)
	mds, err := json.Marshal(fr)
	if err != nil {
		return err
	}
	newMsg := RPCMsg{
		RPCType: CALL,
		IP:      n.Addr.IP,
		NodeID:  n.NodeID,
		Method:  LEECH,
		Message: "",
		Port:    make([]byte, 4),
	}

	binary.LittleEndian.PutUint32(newMsg.Port, uint32(n.Addr.Port))
	for _, p := range c.ClusterPeers {

		buf, err := WrapPayloadToBuffer(newMsg, mds)
		if err != nil {
			continue
		}
		err = SendMsg(buf, p.Addr)
		if err != nil {
			continue
		}
	}

	fmt.Printf("[LEECH REQUEST]: Request sent to peers in cluster: %s\n", cname)

	return nil
}

// Wraps the RPCMsg and the payload into a buffer and returns it
func WrapPayloadToBuffer(msg RPCMsg, payload []byte) ([]byte, error) {
	msg.PayloadSize = uint32(len(payload))
	b, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	// where header append?
	buf := make([]byte, PREFIX_HEADER_SIZE, len(b)+PREFIX_HEADER_SIZE)

	fmt.Printf("HEADER SIZE: %d\n", PREFIX_HEADER_SIZE)
	// creates a header for th json metadata
	binary.LittleEndian.PutUint32(buf, uint32(len(b)))
	fmt.Printf("META JSON SIZE: %d\n", len(b))

	fmt.Printf("PAYLOAD SIZE: %d\n", len(payload))

	// appends marshaled json after getting len of the msg header
	buf = append(buf, b...)
	if payload != nil {
		// appends the payload
		buf = append(buf, payload...)
	}
	fmt.Printf("Total Message Size: %d\n", len(buf))

	return buf, nil
}

func SendViaExistingConn(message []byte, conn io.ReadWriteCloser) error {

	_, err := conn.Write(message)
	if err != nil {
		return err
	}
	fmt.Printf("Marshalled Size: %d-bytes including payload, Prefix Header Size: %d-bytes, Total Size: %d-bytes\n", len(message[PREFIX_HEADER_SIZE:]), PREFIX_HEADER_SIZE, len(message))
	return nil

}

// when sending a message from a CALL rpc type, if the response takes too long, we drop and forget it.
// and consider that peer as offline
func SendMsg(message []byte, peerAddr NodeAddr) error {

	ad := net.TCPAddr{IP: peerAddr.IP, Port: peerAddr.Port}
	fmt.Println(ad.String())
	conn, err := net.Dial("tcp4", ad.String())

	if err != nil {
		return err
	}
	fmt.Println("SENDING")

	_, err = conn.Write(message)
	if err != nil {
		return err
	}

	fmt.Printf("Marshalled Size: %d including payload, Prefix Header Size: %d\nTotal Size: %d\n", len(message[PREFIX_HEADER_SIZE:]), PREFIX_HEADER_SIZE, len(message))

	return nil
}

func ProtocolErrorWrap(errStr string, msgType MsgType, methodType Method) error {
	switch msgType {
	case CALL:
		return fmt.Errorf("[CALL]: ERROR - METHOD: %s - Message: '%s'", MethodStringMap[methodType], errStr)
	case REPLY:
		return fmt.Errorf("[REPLY]: ERROR - METHOD: %s - Message: '%s'", MethodStringMap[methodType], errStr)
	default:
		panic("MsgType not either CALL | REPLY types")
	}
}
