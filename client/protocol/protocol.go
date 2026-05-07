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

type Method uint32
type MethodString string

const (
	PING_NODE Method = iota
	PING_CLUSTER
	LEECH
	PROBE
	JOIN
	FIND_CLUSTER
)

var MethodStringMap = map[Method]MethodString{
	0: "PING NODE",
	1: "PING CLUSTER",
	2: "LEECH",
	3: "PROBE",
	4: "JOIN",
	5: "FIND CLUSTER",
}

const PREFIX_HEADER_SIZE = 4

// Node can implement client connection interface
// this interface describes the characteristics of
// a Node in a cluster that makes it able to communicate with its peers

type DialConn struct {
}

type ClientConn interface {
	FindCluster(cname ClusterName)
	Join()
	Ping(cname ClusterName) error
	Leech(cname ClusterName, spawnThreads bool, fr FileRequest) error
	ProbeFile(fileKey string) (StatusCode, error)
	RecvRPCMessage(msg RPCMsg) error
}

type MsgType uint32

const (
	CALL MsgType = iota
	REPLY
)

type StatusCode int

const (
	SUCCESS StatusCode = iota
	ERROR
)

// MsgType could be either reply or call
type RPCMsg struct {
	RPCType     MsgType
	IP          []byte
	Port        []byte
	NodeID      NodeID
	Method      Method
	PayloadSize uint32
	StatusCode  StatusCode
	Message     string
}

type ClusterRequest string

type ClusterResponse struct {
	ClusterName
	Peers []ClusterPeer
}

type NodeStatus string
type NodeStatusEnum int

const (
	ACTIVE NodeStatusEnum = iota
)

var NodeStatusMap = map[NodeStatusEnum]NodeStatus{
	0: "ACTIVE",
}

type PingResponse NodeStatus
type PingRequest string

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
	return nil
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

func (n *Node) RecvRPCMessage(msg RPCMsg, payload []byte, conn *net.Conn) error {

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
			return n.HandleFindClusterRequest(newRPCMsg, msg, payload, conn)
		case PING_CLUSTER:
			return HandlePingClusterRequest(newRPCMsg, msg, payload, conn)
		case PING_NODE:
			return HandlePingNodeRequest(newRPCMsg, msg, payload, conn)
		case LEECH:
			return n.HandleLeechRequest(newRPCMsg, msg, payload, conn)
		case PROBE:
			return n.HandleProbeRequest(newRPCMsg, msg, payload, conn)
		}
	case REPLY: // when peers/nodes send a call RPCType

		if msg.StatusCode == ERROR {
			return ProtocolErrorWrap(msg.Message, REPLY, msg.Method)
		}

		switch msg.Method {

		case FIND_CLUSTER:
			return n.HandleFindClusterResponse(msg, payload, conn)

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
func (n *Node) PingCluster(cname ClusterName) error {

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
	c, ok := (*n.ClusterTable)[cname]
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

func (n *Node) Probe(cname ClusterName) error {
	c, ok := (*n.ClusterTable)[cname]
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

func (n *Node) Leech(cname ClusterName, spawnThreads bool, fr FileRequest) error {

	c, ok := (*n.ClusterTable)[cname]
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

// -----------------------------------------
// METHODS FOR HANDLING RPC REQUESTS
// -----------------------------------------

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

// -----------------------------------------
// METHODS FOR HANDLING RPC RESPONSE
// -----------------------------------------
