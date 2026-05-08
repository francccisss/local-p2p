package protocol

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
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

func (n *Node) RecvRPCMessage(msg RPCMsg, payload []byte, conn *net.Conn, clt *ClusterTable) error {

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

		case FIND_CLUSTER:
			return HandleFindClusterRequest(newRPCMsg, msg, payload, conn, clt)
		case PING_CLUSTER:
			return HandlePingClusterRequest(newRPCMsg, msg, payload, conn)
		case PING_NODE:
			return HandlePingNodeRequest(newRPCMsg, msg, payload, conn)
		case LEECH:
			return HandleLeechRequest(newRPCMsg, msg, payload, conn, n.FILE_LOCATION)
		case PROBE:
			return HandleProbeRequest(newRPCMsg, msg, payload, conn, n.FILE_LOCATION)
		}
	case REPLY: // when peers/nodes send a call RPCType

		if msg.StatusCode == ERROR {
			return ProtocolErrorWrap(msg.Message, REPLY, msg.Method)
		}

		switch msg.Method {

		case FIND_CLUSTER:
			return HandleFindClusterResponse(msg, payload, conn, clt)

		case LEECH:
			return HandleLeechResponse(msg, payload, conn)

		case PING_NODE:
			return HandlePingNodeResponse(msg, payload, conn)
		case PING_CLUSTER:

			return HandlePingClusterResponse(msg, payload)
		case PROBE:
			return HandleProbeResponse(msg, payload)

		}
		fmt.Println("Reply from Call Message")
	default:
	}
	return nil
}

// ----------------------
// METHODS FOR RPC CALLS
// ----------------------

func (n *Node) FindCluster(cname ClusterName) {

	var wg sync.WaitGroup

	payload := []byte(cname)
	newMsg := RPCMsg{
		NodeID:      n.NodeID,
		Message:     "where cluster?",
		Method:      FIND_CLUSTER,
		RPCType:     CALL,
		IP:          n.Addr.IP,
		PayloadSize: uint32(len(payload)),
	}
	binary.LittleEndian.PutUint32(newMsg.Port, uint32(n.Addr.Port))

	for _, n := range n.NeighboringNodes {
		wg.Go(func() {
			conn, err := net.Dial("tcp", string(n.IP)+":"+strconv.Itoa(n.Port))
			if err != nil {
				fmt.Printf("FIND CLUSTER ERROR: %s", n.NodeID)
				fmt.Println(err)
				return
			}
			err = json.NewEncoder(conn).Encode(newMsg)
			if err != nil {
				fmt.Printf("FIND CLUSTER ERROR: %s", n.NodeID)
				fmt.Println(err)
				return

			}
		})
	}

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
	ping := []byte("Ping")
	newMsg.PayloadSize = uint32(len(ping))
	buf, err := WrapPayloadToBuffer(newMsg, ping)
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
	for _, p := range c.ClusterPeers {
		fmt.Printf("\nPEER: %+v\n", p)

		buf, err := WrapPayloadToBuffer(newMsg, []byte("Ping"))
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

// Need to set little endian of ip and port due to int values
func WrapPayloadToBuffer(msg RPCMsg, payload []byte) ([]byte, error) {
	b, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, PREFIX_HEADER_SIZE, len(b)+PREFIX_HEADER_SIZE)
	fmt.Printf("meta json size: %d\n", len(b))

	// creates a header for th json metadata
	binary.LittleEndian.PutUint32(buf, uint32(len(b)))

	// appends marshaled json after getting len
	buf = append(buf, b...)
	if payload != nil {
		// appends the payload
		buf = append(buf, payload...)
		fmt.Printf("payload size: %d\n", len(payload))
	}

	return buf, nil
}

func SendViaExistingConn(message []byte, conn *net.Conn) error {

	_, err := (*conn).Write(message)
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

// TODO: add checksum parameter passed in by caller
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
