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
	RPCType    MsgType
	NodeAddr   NodeAddr
	NodeID     NodeID
	Method     Method
	Payload    []byte
	StatusCode StatusCode
	Comment    string
}

type PingMessage struct {
	Status PeerStatus
	ClusterName
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

func RecvRPCMessage(n *Node, msg RPCMsg) error {

	switch msg.RPCType {
	case CALL: // when peers/nodes send a call RPCType

		var newRPCMsg RPCMsg
		fmt.Println("Call Message")
		// PRELOADING RPC MESSAGE
		newRPCMsg = RPCMsg{
			RPCType:    REPLY,
			NodeID:     n.NodeID,
			NodeAddr:   NodeAddr{IP: n.Addr.IP, Port: n.Addr.Port},
			Comment:    "",
			StatusCode: 0,
		}
		switch msg.Method {
		case PING:
			// sender triggers a ping on receiver(this)
			// receiver sends their NodeID in return
			// so that the sender can keep track of the receivers
			newRPCMsg.Method = PING
			newRPCMsg.StatusCode = SUCCESS

			var incomingPingMsg PingMessage
			err := json.Unmarshal(msg.Payload, &incomingPingMsg)
			if err != nil {
				return err
			}

			fmt.Printf("Checking cluster table for '%s' cluster\n", incomingPingMsg.ClusterName)
			// it is always assumed that people that have the existing file should have an entry for cluster
			cl, ok := (*n.ClusterTable)[incomingPingMsg.ClusterName]
			// dont need to respond if does not exist anyways
			if !ok {
				fmt.Println("Cluster does not exist")
				newRPCMsg.StatusCode = ERROR
				newRPCMsg.Comment = fmt.Sprintf("Cluster %s does not exist", incomingPingMsg.ClusterName)
				err = SendMsg(n.UDPconn, newRPCMsg, msg.NodeAddr)
				if err != nil {
					return err
				}
				return fmt.Errorf("Unable to deliver reply from PING CALL")
			}

			fmt.Printf("Cluster '%s' exists\n", incomingPingMsg.ClusterName)

			fmt.Printf("Adding '%s' to the cluster\n", msg.NodeID)

			cl.NewClusterPeer(msg.NodeAddr, msg.NodeID)

			for _, cp := range cl.ClusterPeers {
				fmt.Printf("Peers in cluster: '%s'\n", cp.NodeID)
			}

			fmt.Printf("Sending reply back to '%s'\n", msg.NodeID)
			newPingMsg := PingMessage{
				Status:      cl.Peer.Status,
				ClusterName: cl.ClusterName,
			}
			b, err := json.Marshal(newPingMsg)
			if err != nil {
				return err
			}
			newRPCMsg.Payload = b
			err = SendMsg(n.UDPconn, newRPCMsg, msg.NodeAddr)

			if err != nil {
				fmt.Println("Unable to respond to ping")
				return err
			}

			fmt.Printf("Reply sent\n")
		case LEECH:
		//  What TODO

		case PROBE:
			newRPCMsg.Method = PROBE
			cname := string(msg.Payload)
			status, meta, err := ProbeFile(n.FILE_LOCATION, ClusterName(cname))

			if err != nil {
				fmt.Printf("[PROBE REQUEST]: %s\n", err)
				newRPCMsg.Comment = err.Error()
				newRPCMsg.StatusCode = status
				err := SendMsg(n.UDPconn, newRPCMsg, msg.NodeAddr)
				if err != nil {
					return fmt.Errorf("Unable to send message")
				}
			}
			b, err := json.Marshal(meta)

			if err != nil {
				fmt.Println("Unable to Marshal FileMetaData")
				return err
			}

			newRPCMsg.Payload = b
			newRPCMsg.StatusCode = SUCCESS
			err = SendMsg(n.UDPconn, newRPCMsg, msg.NodeAddr)
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
			c, ok := (*n.ClusterTable)[seg.ClusterName]
			if !ok {
				return fmt.Errorf("Cluster does not exist")
			}

			t, ok := c.ClusterPeerThreads[msg.NodeID]
			if !ok {
				return fmt.Errorf("NodeID Key does not exist for thread")
			}
			t.BytesReceived += len(msg.Payload)
			t.NodeIDChann <- msg.NodeID

		case PING:
			// When receivers of the call responds/reply back to this
			// process, create a new cluster with name and initialize
			// pear threads and assign a peer thread that corresponds
			// with the receiver's NodeID that it send from PING

			if msg.StatusCode == ERROR {
				return fmt.Errorf("%s", msg.Comment)
			}
			fmt.Printf("Ping received from %s\n", msg.NodeID)
			var pingMsg PingMessage
			err := json.Unmarshal(msg.Payload, &pingMsg)
			if err != nil {
				return err
			}
			// Verified that the cluster does exist on other peers
			// so create a cluster entry in the cluster table

			// State of new node is set to IDLE on default
			convCname := pingMsg.ClusterName
			cl, ok := (*n.ClusterTable)[convCname]
			if !ok {
				CreateCluster(n, convCname)
				cl = (*n.ClusterTable)[convCname]
			}
			// update map

			cl.NewClusterPeerThread(msg.NodeID)
			cl.NewClusterPeer(msg.NodeAddr, msg.NodeID)
			fmt.Printf("Peer thread created in cluster '%s'\n", convCname)
		case PROBE:
			var fileMetaData FileMetaData
			err := json.Unmarshal(msg.Payload, &fileMetaData)
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

// n asks what other peers if they have this p.cname
// in their table, if so, they add this node and set the status to idle.
// This function can be used within a cluster, if passed in the peers of that cluster
// or the neighboring nodes for creating a cluster table for p.cname in the sender process
func Ping(n *Node, cname ClusterName) error {

	var msg RPCMsg = RPCMsg{
		RPCType:    CALL,
		NodeAddr:   n.Addr,
		Method:     PING,
		StatusCode: SUCCESS,
		NodeID:     n.NodeID,
	}

	// for bootstrapped nodes
	c, ok := (*n.ClusterTable)[cname]
	if !ok {
		return fmt.Errorf("ERROR: Cluster not found")
	}
	fmt.Println(len(c.ClusterPeers))
	for _, p := range c.ClusterPeers {
		fmt.Printf("\nPEER: %+v\n", p)
		newPingMsg := PingMessage{ClusterName: cname, Status: IDLE}

		b, err := json.Marshal(newPingMsg)
		if err != nil {
			return err
		}

		msg.Payload = b
		err = SendMsg(n.UDPconn, msg, p.Addr)
		if err != nil {
			fmt.Printf("%s", err)
			continue
		}
	}

	fmt.Println("\nPinging peers in cluster.")
	fmt.Println("Ping Sent")
	return nil
}

func Probe(n *Node, cname ClusterName) error {
	c, ok := (*n.ClusterTable)[cname]
	if !ok {
		return fmt.Errorf("Cluster does not exist")
	}
	newMsg := RPCMsg{
		RPCType:  CALL,
		Method:   PROBE,
		NodeID:   n.NodeID,
		NodeAddr: n.Addr,
		Payload:  []byte(cname),
	}

	for _, cp := range c.ClusterPeers {
		err := SendMsg(n.UDPconn, newMsg, cp.Addr)
		if err != nil {
			continue
		}
	}

	return nil

}

func Leech(n *Node, cname ClusterName) error {

	c, ok := (*n.ClusterTable)[cname]
	if !ok {
		return fmt.Errorf("Cluster does not exist")
	}

	c.Peer.Status = LEECHING

	var wg sync.WaitGroup

	// TODO: Figure out when to cancel and how
	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	fmt.Println("[LEECH REQUEST]: Preparing Cluster Peer Threads for Byte Measurement")

	for _, p := range c.ClusterPeers {
		wg.Go(func() {
			c.SpawnPeerThreads(&ctx, c.ClusterPeerThreads[p.NodeID])
		})
	}
	fmt.Printf("[LEECH REQUEST]: Waiting for Cluster Peer Threads to deploy...")
	wg.Wait()

	fmt.Printf("[LEECH REQUEST]: Done!")
	fmt.Printf("[LEECH REQUEST]: Now Sending request to peers in cluster: %s\n", cname)
	newMsg := RPCMsg{}
	for _, p := range c.ClusterPeers {
		err := SendMsg(n.UDPconn, newMsg, p.Addr)
		if err != nil {
			continue
		}
	}

	fmt.Printf("[LEECH REQUEST]: Request sent to peers in cluster: %s\n", cname)

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

	fmt.Printf("Sending to: %s\n", ip+":"+port)
	n, err := conn.WriteTo(b, raddr)
	if err != nil {
		return err
	}

	fmt.Printf("Marshalled: %d\nSent: %d\n", len(b), n)

	return nil
}
