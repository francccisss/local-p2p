package protocol

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
)

type Method int

const (
	PING Method = iota
	LEECH
	PROBE
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
	Method          Method
	Payload         []byte
	StatusCode      StatusCode
	Comment         string
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

func RecvRPCMessage(n *Node, msg RPCMsg) error {

	var newRPCMsg RPCMsg
	switch msg.RPCType {
	case CALL:
		fmt.Println("Call Message")
		newRPCMsg = RPCMsg{
			RPCType:  CALL,
			Comment:  "",
			NodeAddr: NodeAddr{IP: n.Addr.IP, Port: n.Addr.Port},
		}
		switch msg.Method {
		case PING:
			newRPCMsg.Method = PING
			newRPCMsg.Payload = []byte("Pong")
			newRPCMsg.StatusCode = SUCCESS
			err := SendMsg(n.UDPconn, newRPCMsg, msg.NodeAddr)
			if err != nil {
				fmt.Println("Unable to respond to ping")
				return err
			}
		case LEECH:

		case PROBE:

			fileKey := string(msg.Payload)
			_, err := ProbeFile(n, fileKey)
			if err != nil {
				fmt.Println("Error unable to probe for file")
				return err
			}
			fmt.Println("Success")
		}

	case REPLY:
		fmt.Println("Reply from Call Message")
	default:

	}

	return nil
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

func Ping(n *Node, wg *sync.WaitGroup) error {

	var msg RPCMsg = RPCMsg{
		RPCType:    CALL,
		NodeAddr:   n.Addr,
		Method:     PING,
		Payload:    []byte("Ping"),
		StatusCode: SUCCESS,
	}

	for i := 0; i < len(n.PeerTable); i++ {
		var p Peer = n.PeerTable[i]
		fmt.Printf("\nPEER: %+v\n", p)
		SendMsg(n.UDPconn, msg, p.NodeAddr)
	}

	fmt.Println("\nPinging peers in cluster.")
	fmt.Println("Ping Sent")
	return nil
}
