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

type Method int
type MethodString string

const (
	PING Method = iota
	LEECH
	PROBE
	JOIN
	FIND_CLUSTER
)

var MethodStringMap = map[Method]MethodString{
	0: "PING",
	1: "LEECH",
	2: "PROBE",
	3: "JOIN",
	4: "FIND CLUSTER",
}

const PREFIX_HEADER_SIZE = 4

// Node can implement client connection interface
// this interface describes the characteristics of
// a Node in a cluster that makes it able to communicate with its peers

type ClientConn interface {
	FindCluster(cname ClusterName)
	Join()
	Ping(cname ClusterName) error
	Leech(cname ClusterName, spawnThreads bool, fr FileRequest) error
	ProbeFile(fileKey string) (StatusCode, error)
	RecvRPCMessage(msg RPCMsg) error
}

type MsgType int

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
	NodeAddr    NodeAddr
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

// -----------------------------------------
// METHODS FOR HANDLING A `CALL` RPC MESSAGE
// -----------------------------------------

// TODO add checksum parameter passed in by caller
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
// METHODS FOR CREATING A `CALL` RPC MESSAGE
// -----------------------------------------

func (n *Node) RecvRPCMessage(msg RPCMsg, payload []byte) error {

	switch msg.RPCType {
	case CALL: // when peers/nodes send a call RPCType

		var newRPCMsg RPCMsg
		fmt.Println("Call Message")
		// PRELOADING RPC MESSAGE
		newRPCMsg = RPCMsg{
			Method:      msg.Method,
			RPCType:     REPLY,
			NodeID:      n.NodeID,
			NodeAddr:    NodeAddr{IP: n.Addr.IP, Port: n.Addr.Port},
			Message:     "",
			StatusCode:  0,
			PayloadSize: 0,
		}
		switch msg.Method {

		case FIND_CLUSTER:
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
				err = n.SendMsg(b, msg.NodeAddr)
				if err != nil {
					return err
				}
				return fmt.Errorf("Unable to deliver reply from PING CALL")
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
			err = n.SendMsg(buff, msg.NodeAddr)

			if err != nil {
				fmt.Println("Unable to respond to ping")
				return err
			}

		case PING:
			// sender triggers a ping on receiver(this)
			// receiver sends their NodeID in return
			// so that the sender can keep track of the receivers
			newRPCMsg.StatusCode = SUCCESS
			newRPCMsg.Message = "Pong"
			pr := PingRequest(payload)
			fmt.Printf("[CALL]: PING - pinged by neighboring node %s\n", pr)
			var newPingResponse PingResponse = PingResponse(NodeStatusMap[ACTIVE]) // interesting unnessary type assertion

			b, err := WrapPayloadToBuffer(newRPCMsg, []byte(newPingResponse))
			if err != nil {
				return protocolErrorWrap(err.Error(), CALL, PING)
			}
			err = n.SendMsg(b, msg.NodeAddr)
			if err != nil {
				return protocolErrorWrap(err.Error(), CALL, PING)
			}
			fmt.Println("Ping")
		case LEECH:
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
				msgErr := n.SendMsg(buff, msg.NodeAddr)
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
			err = n.SendMsg(b, msg.NodeAddr)
			if err != nil {
				return err
			}

		case PROBE:
			newRPCMsg.Method = PROBE
			cname := string(payload)
			status, meta, err := ProbeFile(n.FILE_LOCATION, ClusterName(cname))

			if err != nil {
				fmt.Printf("[PROBE REQUEST]: %s\n", err)
				newRPCMsg.Message = err.Error()
				newRPCMsg.StatusCode = status
				buf, err := WrapPayloadToBuffer(newRPCMsg, nil)
				if err != nil {
					return fmt.Errorf("Unable to send message")
				}
				err = n.SendMsg(buf, msg.NodeAddr)
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
			err = n.SendMsg(buf, msg.NodeAddr)
			if err != nil {
				return err
			}
		}
	case REPLY: // when peers/nodes send a call RPCType

		if msg.StatusCode == ERROR {
			return fmt.Errorf("%s", msg.Message)
		}
		var seg SegmentHeader
		err := json.Unmarshal(payload, &seg)
		if err != nil {
			return err
		}

		switch msg.Method {

		case FIND_CLUSTER:
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

		case LEECH:

			// match the clustername and then the NodeID that sent the request
			c, ok := (*n.ClusterTable)[seg.ClusterName]
			if !ok {
				return fmt.Errorf("Cluster does not exist")
			}

			t, ok := c.ClusterPeerThreads[msg.NodeID]
			if !ok {
				return fmt.Errorf("NodeID Key does not exist for thread")
			}
			var ds SegmentHeader
			err := json.Unmarshal(payload, &ds)
			if err != nil {
				return err
			}
			// t.BytesReceived += len(ds.DataChunk)
			t.NodeIDChann <- msg.NodeID
			// calculate segments received currently
			if ds.SegmentPosition == ds.TotalSegments {
				fmt.Println("WE ARE DONE BUCKO")
				return nil
			}
			// read the file to create a FileRequest
			// the file contains the meta data of the file that is to
			// be leeched from other peers
			// TODO: make sure that we're calling openfile once
			// store it in memory so we dont have to use syscall
			// everytime a leech reply is received which would take potentially take

			// TODO: For testing only remove this after testing the file transfer if it works
			dfinfo := FileRequest{
				Hash:      "IosevkaTerm.zip",
				Size:      374780158,
				Offset:    ds.SegmentPosition,
				BlockSize: int64(os.Getpagesize()),
			}
			// TODO: For testing only remove this after testing the file transfer if it works

			err = n.Leech(ds.ClusterName, false, dfinfo)
			if err != nil {
				return err
			}
		case PING:
			// When receivers of the call responds/reply back to this
			// process, create a new cluster with name and initialize
			// pear threads and assign a peer thread that corresponds
			// with the receiver's NodeID that it send from PING

			if msg.StatusCode == ERROR {
				return protocolErrorWrap(msg.Message, REPLY, PING)
			}
			fmt.Printf("Ping received from %s\n", msg.NodeID)
			var pingMsg PingResponse = PingResponse(payload)
			fmt.Println(pingMsg)
			fmt.Println("Pong")
		case PROBE:
			var fileMetaData FileMetaData
			err := json.Unmarshal(payload, &fileMetaData)
			if err != nil {
				fmt.Println("Unable to Unmarshal FileMetaData")
				return err
			}

			fmt.Println("File Meta Data Received")
			fmt.Printf("File Meta Data: %+v\n", fileMetaData)

		}
		fmt.Println("Reply from Call Message")
	default:
	}
	return nil
}

func (n *Node) FindCluster(cname ClusterName) {

	var wg sync.WaitGroup

	payload := []byte(cname)
	newMsg := RPCMsg{
		NodeID:      n.NodeID,
		NodeAddr:    n.Addr,
		Message:     "where cluster?",
		Method:      FIND_CLUSTER,
		RPCType:     CALL,
		PayloadSize: uint32(len(payload)),
	}

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

func (n *Node) PingNodes() error {

	var newMsg RPCMsg = RPCMsg{
		RPCType:    CALL,
		NodeAddr:   n.Addr,
		Method:     PING,
		StatusCode: SUCCESS,
		NodeID:     n.NodeID,
	}

	for _, neighbor := range n.NeighboringNodes {
		fmt.Printf("\nNEIGHBOR: %+v\n", neighbor)

		buf, err := WrapPayloadToBuffer(newMsg, []byte("Ping"))
		if err != nil {
			fmt.Printf("%s", err)
			continue
		}
		err = n.SendMsg(buf, neighbor.Addr)
		if err != nil {
			fmt.Printf("%s", err)
			continue
		}
	}

	fmt.Println("\nPinging peers in cluster.")
	fmt.Println("Ping Sent")
	return nil
}

func (n *Node) PingCluster(cname ClusterName) error {

	var newMsg RPCMsg = RPCMsg{
		RPCType:    CALL,
		NodeAddr:   n.Addr,
		Method:     PING,
		StatusCode: SUCCESS,
		NodeID:     n.NodeID,
	}

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
		err = n.SendMsg(buf, p.Addr)
		if err != nil {
			fmt.Printf("%s", err)
			continue
		}
	}

	fmt.Println("\nPinging peers in cluster.")
	fmt.Println("Ping Sent")
	return nil
}

func (n *Node) Probe(cname ClusterName) error {
	c, ok := (*n.ClusterTable)[cname]
	if !ok {
		return fmt.Errorf("Cluster does not exist")
	}
	newMsg := RPCMsg{
		RPCType:  CALL,
		Method:   PROBE,
		NodeID:   n.NodeID,
		NodeAddr: n.Addr,
	}

	for _, cp := range c.ClusterPeers {

		buf, err := WrapPayloadToBuffer(newMsg, []byte(cname))
		if err != nil {
			continue
		}
		err = n.SendMsg(buf, cp.Addr)
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
		RPCType:  CALL,
		NodeAddr: n.Addr,
		NodeID:   n.NodeID,
		Method:   LEECH,
		Message:  "",
	}
	for _, p := range c.ClusterPeers {

		buf, err := WrapPayloadToBuffer(newMsg, mds)
		if err != nil {
			continue
		}
		err = n.SendMsg(buf, p.Addr)
		if err != nil {
			continue
		}
	}

	fmt.Printf("[LEECH REQUEST]: Request sent to peers in cluster: %s\n", cname)

	return nil
}

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

// when sending a message from a CALL rpc type, if the response takes too long, we drop and forget it.
// and consider that peer as offline
func (n *Node) SendMsg(message []byte, peerAddr NodeAddr) error {
	ip := string(peerAddr.IP)
	port := strconv.Itoa(peerAddr.Port)
	// first byte size of json meta data

	conn, err := net.Dial("tcp", string(peerAddr.IP)+":"+strconv.Itoa(peerAddr.Port))

	if err != nil {
		return err
	}
	fmt.Printf("Sending to: %s\n", ip+":"+port)

	_, err = conn.Write(message)
	if err != nil {
		return err
	}

	fmt.Printf("Marshalled Size: %d including payload, Prefix Header Size: %d\nTotal Size: %d", len(message[PREFIX_HEADER_SIZE:]), PREFIX_HEADER_SIZE, len(message))

	return nil
}

func protocolErrorWrap(errStr string, msgType MsgType, methodType Method) error {
	switch msgType {
	case CALL:
		return fmt.Errorf("[REPLY]: ERROR - METHOD: %s - Message: '%s'", MethodStringMap[methodType], errStr)
	case REPLY:
		return fmt.Errorf("[CALL]: ERROR - METHOD: %s - Message: '%s'", MethodStringMap[methodType], errStr)
	default:
		panic("MsgType not either CALL | REPLY types")
	}
}
