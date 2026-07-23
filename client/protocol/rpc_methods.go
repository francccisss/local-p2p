package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
)

// ----------------------
// METHODS FOR RPC CALLS
// ----------------------

func (n *Node) FindCluster(cname string) error {

	var wg sync.WaitGroup

	payload := ClusterRequest(cname)
	newMsg := RPCMsgHeader{
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

	var newMsg RPCMsgHeader = RPCMsgHeader{
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
func (n *Node) PingCluster(cname string, clt *ClusterTable) error {

	var newMsg RPCMsgHeader = RPCMsgHeader{
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

func (n *Node) ProbeFile(cname string, clt *ClusterTable) error {
	c, ok := (*clt)[cname]
	if !ok {
		return fmt.Errorf("[CALLING]: PROBE[ERROR] - 'Cluster does not exist'")
	}
	newMsg := RPCMsgHeader{
		RPCType: CALL,
		Method:  PROBE,
		NodeID:  n.NodeID,
		IP:      n.Addr.IP,
		Message: "Probe",
		Port:    make([]byte, 4),
	}

	// TODO: update cname to actual file hash
	newProbeReq := ProbeRequest{
		ClusterHash: cname,
		FileHash:    c.ClusterHash,
	}
	b, err := json.Marshal(newProbeReq)
	if err != nil {
		return fmt.Errorf("[CALLING]: PROBE[ERROR] - %s", err)
	}

	binary.LittleEndian.PutUint32(newMsg.Port, uint32(n.Addr.Port))

	nSent := 0
	for _, cp := range c.ClusterPeers {
		buf, err := WrapPayloadToBuffer(newMsg, b)
		if err != nil {
			fmt.Printf("[CALLING]: PROBE[ERROR] - %s", err)
			continue
		}
		err = SendViaExistingConn(buf, cp.Conn)
		if err != nil {
			fmt.Printf("[CALLING]: PROBE[ERROR] - %s", err)
			continue
		}
		nSent++
	}
	fmt.Printf("[CALLING]: PROBE - 'Sent Probe request to %d Peers in Cluster %s'\n", nSent, cname)

	return nil

}

// Be called AFTER creating a cluster and appending bootstrapped nodes on cluster
func (n *Node) JoinCluster(fmd FileMetaData, clt *ClusterTable) error {

	newMsg := RPCMsgHeader{
		NodeID:  n.NodeID,
		IP:      n.Addr.IP,
		Port:    make([]byte, 4),
		Message: "Joining cluster",
		Method:  JOIN,
		RPCType: CALL,
	}

	binary.LittleEndian.PutUint32(newMsg.Port, uint32(n.Addr.Port))

	cluster := (*clt)[fmd.Hash]

	b, err := WrapPayloadToBuffer(newMsg, []byte(fmd.Hash))

	if err != nil {
		return fmt.Errorf("[CALLING]: JOIN_CLUSTER[ERROR] - '%s'\n", err.Error())
	}

	// TODO: Need to dial instead of using tcp write
	// TODO: Change this afterwards
	for _, nc := range cluster.ClusterPeers {
		_, err := nc.Conn.Write(b)
		if err != nil {
			continue
		}
	}

	return nil
}

func (n *Node) Leech(fr FileRequest, cpeer ClusterPeer, isConnectionExist bool) error {

	mds, err := json.Marshal(fr)
	if err != nil {
		return err
	}
	newMsg := RPCMsgHeader{
		RPCType: CALL,
		IP:      n.Addr.IP,
		NodeID:  n.NodeID,
		Method:  LEECH,
		Message: "",
		Port:    make([]byte, 4),
	}

	binary.LittleEndian.PutUint32(newMsg.Port, uint32(n.Addr.Port))

	buf, err := WrapPayloadToBuffer(newMsg, mds)
	if err != nil {
		return err
	}
	if !isConnectionExist {
		err = SendMsg(buf, cpeer.Addr)
		if err != nil {
			return err
		}
	}
	err = SendViaExistingConn(buf, cpeer.Conn)
	if err != nil {
		return err
	}

	fmt.Printf("[LEECH REQUEST]: Request sent to peers in cluster: %s\n", cpeer.NodeID)

	return nil
}

// Wraps the RPCMsgHeader and the payload into a buffer and returns it
func WrapPayloadToBuffer(msg RPCMsgHeader, payload []byte) ([]byte, error) {
	msg.PayloadSize = uint32(len(payload))
	b, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	// where header append?
	buf := make([]byte, PREFIX_HEADER_SIZE, len(b)+PREFIX_HEADER_SIZE)

	fmt.Printf("[ PAYLOAD WRAPPER ] HEADER SIZE: %d\n", PREFIX_HEADER_SIZE)
	// creates a header for th json metadata
	binary.LittleEndian.PutUint32(buf, uint32(len(b)))
	fmt.Printf("[ PAYLOAD WRAPPER ]: META JSON SIZE: %d\n", len(b))

	fmt.Printf("[ PAYLOAD WRAPPER ]: PAYLOAD SIZE: %d\n", len(payload))

	// appends marshaled json after getting len of the msg header
	buf = append(buf, b...)
	if payload != nil {
		// appends the payload
		buf = append(buf, payload...)
	}
	fmt.Printf("[ PAYLOAD WRAPPER ]: Total Message Size: %d\n", len(buf))

	return buf, nil
}

func SendViaExistingConn(message []byte, conn io.ReadWriteCloser) error {

	_, err := conn.Write(message)
	if err != nil {
		return err
	}
	fmt.Printf("[ SENDING ]: Marshalled Size: %d-bytes including payload, Prefix Header Size: %d-bytes, Total Size: %d-bytes\n", len(message[PREFIX_HEADER_SIZE:]), PREFIX_HEADER_SIZE, len(message))
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

	fmt.Printf("[ SENDING ]: Marshalled Size: %d including payload, Prefix Header Size: %d\nTotal Size: %d\n", len(message[PREFIX_HEADER_SIZE:]), PREFIX_HEADER_SIZE, len(message))

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
