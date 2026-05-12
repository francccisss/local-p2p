package test

import (
	"client/connection"
	"client/protocol"
	"fmt"
	"net"
	"testing"
)

type dummyCommChannel struct {
	buf chan []byte
	n   int // current byte index to buffer
}

func (clb *dummyCommChannel) Read(p []byte) (n int, err error) {
	// it reads all of clb.buf into
	rb := <-clb.buf
	fmt.Printf("READ FROM BUF TO P %d\n", len(rb))
	readBytes := copy(p, rb[:clb.n])
	return readBytes, nil
}

func (clb *dummyCommChannel) Write(p []byte) (n int, err error) {
	fmt.Printf("WROTE TO BUFF %d\n", len(p))
	clb.buf <- p
	clb.n = len(p)
	return len(p), nil
}
func (clb *dummyCommChannel) Close() error {
	return nil
}

func TestClusterSearch(t *testing.T) {
	fmt.Println("[Begin Test]: Test Cluster Search")
	var clusterName protocol.ClusterName = "vim-boys"

	addr := net.TCPAddr{IP: net.ParseIP("127.0.0.1")}
	dchl := dummyCommChannel{buf: make(chan []byte, 1), n: 0}

	// different go routine that sends request to receiver node
	clientNode := protocol.NewNode(protocol.NodeAddr{IP: addr.IP, Port: 3030}, "requester", "")
	clientNode.NeighboringNodes = append(clientNode.NeighboringNodes, protocol.NodeAddr{IP: addr.IP, Port: 5656, Conn: &dchl})
	clientclt := protocol.CreateClusterTable()

	//  writes to dchl
	clientNode.FindCluster(clusterName)

	consumerNode := protocol.NewNode(protocol.NodeAddr{IP: net.ParseIP("127.0.0.1")}, "receiver", "")
	ConsumerNodeReaderWriterToChannel(*consumerNode, &dchl, clusterName)

	bodyBuf := make([]byte, 4096)

	var mr connection.MessageReader = connection.MessageReader{
		PayloadBuffer: make([]byte, 0, 4096),
		JsonBuffer:    make([]byte, 0, 4096),
		State:         connection.PROCESSING,
	}

	n, err := dchl.Read(bodyBuf)
	if err != nil {
		t.Fatalf("TEST ERROR: %s\n", err.Error())
	}
	fmt.Printf("len of bodyBuf: %d\n", len(bodyBuf[:n]))

	msg, pl, err := mr.ExtractMessage(bodyBuf)
	if err != nil {
		protocol.ProtocolErrorWrap(err.Error(), protocol.CALL, protocol.JOIN)
		t.FailNow()
	}

	// read and sends back response to find cluster method via writing into recvclt
	err = clientNode.RecvRPCMessage(msg, pl, &dchl, clientclt)

	if err != nil {
		protocol.ProtocolErrorWrap(err.Error(), protocol.CALL, protocol.JOIN)
		t.FailNow()
	}

}

func TestJoinCluster(t *testing.T) {
	var clusterName protocol.ClusterName = "vim-boys"

	addr := net.TCPAddr{IP: net.ParseIP("127.0.0.1")}
	dchl := dummyCommChannel{buf: make(chan []byte, 1), n: 0}

	// different go routine that sends request to receiver node
	clientNode := protocol.NewNode(protocol.NodeAddr{IP: addr.IP, Port: 3030}, "requester", "")

	// after FIND_CLUSTR
	clientclt := protocol.CreateClusterTable()

	newCluster := protocol.CreateCluster(clusterName)

	(*clientclt)[clusterName] = newCluster

	newCluster.ClusterPeers = append(newCluster.ClusterPeers, protocol.ClusterPeer{
		Addr:   protocol.NodeAddr{IP: addr.IP, Port: 5656},
		Conn:   &dchl,
		NodeID: "receiver",
		Status: protocol.IDLE,
	})

	err := clientNode.JoinCluster(clusterName, clientclt)

	recvNode := protocol.NewNode(protocol.NodeAddr{IP: addr.IP, Port: 5656}, "receiver", "")
	ConsumerNodeReaderWriterToChannel(*recvNode, &dchl, clusterName)

	// invokes RCP method
	if err != nil {
		protocol.ProtocolErrorWrap(err.Error(), protocol.CALL, protocol.JOIN)
		t.FailNow()
	}

	bodyBuf := make([]byte, 4096)

	var mr connection.MessageReader = connection.MessageReader{
		PayloadBuffer: make([]byte, 0, 4096),
		JsonBuffer:    make([]byte, 0, 4096),
		State:         connection.PROCESSING,
	}

	// Reads reply from receiver
	dchl.Read(bodyBuf)
	// receives data from clientNode
	msg, pl, err := mr.ExtractMessage(bodyBuf)
	if err != nil {
		fmt.Println(protocol.ProtocolErrorWrap(err.Error(), protocol.CALL, protocol.JOIN))
		t.FailNow()
	}

	// read and sends back response to find cluster method via writing into recvclt
	err = clientNode.RecvRPCMessage(msg, pl, &dchl, clientclt)

	if err != nil {
		fmt.Println(protocol.ProtocolErrorWrap(err.Error(), protocol.CALL, protocol.JOIN))
		t.FailNow()
	}
	cl := (*clientclt)[clusterName]

	if len(cl.ClusterPeers) == 0 {
		t.Fatalf("Cluster was not added")
	}
	for _, cp := range cl.ClusterPeers {
		fmt.Printf("Cluster Peer: %+v\n", cp)
	}
	fmt.Printf("Cluster added to %s\n", clusterName)
}

// Writes reply to a dummy channel
func ConsumerNodeReaderWriterToChannel(n protocol.Node, dchl *dummyCommChannel, clusterName protocol.ClusterName) {
	receiverNode := protocol.NewNode(n.Addr, "receiver", "")
	recvclt := protocol.CreateClusterTable()
	(*recvclt)[clusterName] = protocol.CreateCluster(clusterName)
	bodyBuf := make([]byte, 4096)

	var mr connection.MessageReader = connection.MessageReader{
		PayloadBuffer: make([]byte, 0, 4096),
		JsonBuffer:    make([]byte, 0, 4096),
		State:         connection.PROCESSING,
	}

	nyte, _ := dchl.Read(bodyBuf)
	// receives data from clientNode
	msg, pl, err := mr.ExtractMessage(bodyBuf[:nyte])
	if err != nil {
		protocol.ProtocolErrorWrap(err.Error(), protocol.CALL, protocol.JOIN)
		return
	}

	// read and sends back response to find cluster method via writing into recvclt
	err = receiverNode.RecvRPCMessage(msg, pl, dchl, recvclt)

	if err != nil {
		protocol.ProtocolErrorWrap(err.Error(), protocol.CALL, protocol.JOIN)
		return
	}
}
