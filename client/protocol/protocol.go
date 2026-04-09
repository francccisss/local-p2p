package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
	"time"
)

type Method int

const (
	PING Method = iota
	LEECH
	PROBE
	SENDFILE
)

// Node can implement client connection interface
// this interface describes the characteristics of
// a Node in a cluster that makes it able to communicate with its peers

type ClientConn interface {
	Ping() error
	Leech() error
	ProbeFile(fileKey string) (StatusCode, error) // checks for file existence also should do checksum for data integrity before transfering for security reasons
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

type BodyMsg struct {
}

// MsgType could be either reply or call
type RPCMsg struct {
	SegmentPosition int
	SegmentCount    int
	RPCType         MsgType
	NodeAddr        NodeAddr
	NodeID          NodeID
	Method          Method
	Payload         []byte
	StatusCode      StatusCode
	Comment         string
}

// buffer is the payload received from a peer
func ReadRPCMessage(buffer []byte) (RPCMsg, error) {

	var msg RPCMsg
	err := json.Unmarshal(buffer, &msg)
	if err != nil {
		return RPCMsg{}, err
	}
	return msg, nil
}

// TODO add checksum parameter passed in by caller
func ProbeFile(n *Node, fileKey string) (StatusCode, error) {

	entry, path, err := n.Checkfile(fileKey, n.FILE_LOCATION)
	if err != nil {
		return ERROR, err
	}

	file, err := entry.Info()
	if err != nil {
		return ERROR, err
	}
	// obviously need to use the absolute route to the file
	// reuse wd prefix? hmmm
	fmt.Printf("Absolute Path: %s\n", path)
	fileBuffer, err := os.ReadFile(path + file.Name())

	if err != nil {
		return ERROR, err
	}

	fmt.Printf("file length: %d\n", len(fileBuffer))

	// check data integrity of file using checksum

	return SUCCESS, nil
}

func Ping(n *Node, wg *sync.WaitGroup, cname ClusterName) error {

	var msg RPCMsg = RPCMsg{
		RPCType:    CALL,
		NodeAddr:   n.Addr,
		Method:     PING,
		Payload:    []byte(cname),
		StatusCode: SUCCESS,
	}

	for i := 0; i < len(n.Peers); i++ {
		var p Peer = n.Peers[i]
		fmt.Printf("\nPEER: %+v\n", p)
		SendMsg(n.UDPconn, msg, p.NodeAddr)
	}

	fmt.Println("\nPinging peers in cluster.")
	fmt.Println("Ping Sent")
	return nil
}

// when sending a message from a CALL rpc type, if the response takes too long, we drop and forget it.
// and consider that peer as offline
func SendMsg(conn *net.UDPConn, message RPCMsg, peerAddr NodeAddr) error {
	b, err := json.Marshal(message)
	if err != nil {
		return err
	}

	ip := string(peerAddr.IP)
	port := strconv.Itoa(peerAddr.Port)
	raddr, err := net.ResolveUDPAddr("udp", ip+":"+port)

	if err != nil {
		return err
	}

	fmt.Printf("\nSend to: %s\n", ip+":"+port)
	n, err := conn.WriteTo(b, raddr)
	if err != nil {
		return err
	}

	fmt.Printf("Marshalled: %d\nSent: %d\n", len(b), n)

	return nil
}

// func SendDataFileSegments(n *Node, datas []DataSegment, cname ClusterName) {
// 	for _, p := range n.Peers {
//
// 	}
// }

func Leech(n *Node, cname ClusterName) error {

	c, ok := n.ClusterTable[cname]
	if !ok {
		return fmt.Errorf("Cluster does not exist")
	}

	return nil
}

func RecvRPCMessage(n *Node, msg RPCMsg) error {

	var newRPCMsg RPCMsg
	switch msg.RPCType {
	case CALL: // when peers/nodes send a call RPCType
		fmt.Println("Call Message")
		// PRELOADING RPC MESSAGE
		newRPCMsg = RPCMsg{
			RPCType:  REPLY,
			Comment:  "",
			NodeID:   n.NodeID,
			NodeAddr: NodeAddr{IP: n.Addr.IP, Port: n.Addr.Port},
		}
		switch msg.Method {
		case PING:
			// sender triggers a ping on receiver(this)
			// receiver sends their NodeID in return
			// so that the sender can keep track of the receivers
			newRPCMsg.Method = PING
			newRPCMsg.Payload = []byte(n.NodeID)
			newRPCMsg.StatusCode = SUCCESS
			err := SendMsg(n.UDPconn, newRPCMsg, msg.NodeAddr)
			if err != nil {
				fmt.Println("Unable to respond to ping")
				return err
			}
		case LEECH:
		// reply to LEECH request
		case PROBE:

			fileKey := string(msg.Payload)
			_, err := ProbeFile(n, fileKey)
			if err != nil {
				fmt.Println("Error unable to probe for file")
				return err
			}
			fmt.Println("Success")
		}

	case REPLY: // when peers/nodes send a call RPCType

		if msg.StatusCode == ERROR {
			return fmt.Errorf("%s", msg.Comment)
		}
		var seg DataSegment
		err := json.Unmarshal(msg.Payload, &seg)
		if err != nil {
			return err
		}

		switch msg.Method {
		case LEECH:

			// match the clustername and then the NodeID that sent the request
			c, ok := n.ClusterTable[seg.ClusterName]
			if !ok {
				return fmt.Errorf("Cluster does not exist")
			}

			t, ok := c.PeerThreads[msg.NodeID]
			if !ok {
				return fmt.Errorf("NodeID Key does not exist for thread")
			}
			t.bytesReceived += len(msg.Payload)
			t.NodeIDChann <- msg.NodeID

		case PING:
			// when receivers of the call responds/reply back to this
			// process, create a new cluster with name and initialize
			// pear threads and assign a peer thread that corresponds
			// with the receiver's NodeID that it send from PING
			convCname := ClusterName(string(msg.Payload))
			c, ok := n.ClusterTable[convCname]
			if !ok {
				// return fmt.Errorf("NodeID Key does not exist for thread")
				n.ClusterTable[convCname] = Cluster{
					PeerThreads: make(map[NodeID]PeerThread),
					ClusterName: convCname,
				}

				c = n.ClusterTable[convCname]
			}

			// update map
			c.PeerThreads[msg.NodeID] = NewPeerThread(convCname)
			n.ClusterTable[convCname] = c

			// after setting up peers and creating go routines, create new threads

		}

		fmt.Println("Reply from Call Message")

	default:

	}

	return nil
}

// will be received every reply to LEECH is received
// Use ctx to cancel when leeching is done
func MeasurePeerTransfer(ctx *context.Context, n *Node, threadTimer *PeerThread) {

	for {
		select {
		case <-(*ctx).Done():
			// clean up thread ORR ELSEEE!!!
			return
		case nodeID := <-(*threadTimer).NodeIDChann:
			{
				fmt.Printf("Transfer by: %s\n", nodeID)

				currentTime := time.Now()
				elapsedms := currentTime.Sub(threadTimer.timeSince)
				threadTimer.averageBytes = threadTimer.bytesReceived / int(elapsedms)

				fmt.Printf("Bytes transferred per second: %dms\n", threadTimer.averageBytes)
				threadTimer.bytesReceived = 0
			}

		}
	}

}
