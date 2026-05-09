package test

import (
	"client/connection"
	"client/protocol"
	"fmt"
	"net"
	"sync"
	"testing"
)

func TestClientConnection(t *testing.T) {

	var node *protocol.Node
	addr := net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 3030}

	node = protocol.NewNode(protocol.NodeAddr{IP: addr.IP, Port: 3030}, "pinger", "")
	node.NeighboringNodes = append(node.NeighboringNodes, protocol.NodeAddr{IP: net.ParseIP("127.0.0.1"), Port: 5656})
	newClusterTable := protocol.CreateClusterTable()
	fmt.Println("Pinging Nodes")
	err := node.PingNodes()
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	fmt.Println(addr.String())
	listener, err := net.Listen("tcp4", addr.String())
	if err != nil {
		fmt.Println("[TEST]: ERROR Listener")
		fmt.Println(err)
		t.FailNow()
	}
	for {
		if err := connection.HandleConn(&listener, node, newClusterTable); err != nil {
			fmt.Println("[TEST]: ERROR Listener")
			fmt.Println(err)
			t.FailNow()
		}
	}
}

type clusterBuff struct {
	buf chan []byte
	n   int // current byte index to buffer
}

func (clb *clusterBuff) Read(p []byte) (n int, err error) {
	// it reads all of clb.buf into
	rb := <-clb.buf
	fmt.Printf("READ FROM BUF TO P %d\n", len(rb))
	readBytes := copy(p, rb[:clb.n])
	return readBytes, nil
}

func (clb *clusterBuff) Write(p []byte) (n int, err error) {
	fmt.Printf("WROTE TO BUFF %d\n", len(p))
	clb.buf <- p
	clb.n = len(p)
	return len(p), nil
}
func (clb *clusterBuff) Close() error {
	return nil
}

func TestClusterSearch(t *testing.T) {
	var clusterName protocol.ClusterName = "vim-boys"

	addr := net.TCPAddr{IP: net.ParseIP("127.0.0.1")}
	rclb := clusterBuff{buf: make(chan []byte, 1), n: 0}

	receiverNode := protocol.NewNode(protocol.NodeAddr{IP: addr.IP, Port: 5656}, "receiver", "")
	recvclt := protocol.CreateClusterTable()
	(*recvclt)[clusterName] = protocol.CreateCluster(clusterName)

	// different go routine that sends request to receiver node
	clientNode := protocol.NewNode(protocol.NodeAddr{IP: addr.IP, Port: 3030}, "requester", "")
	clientNode.NeighboringNodes = append(clientNode.NeighboringNodes, protocol.NodeAddr{IP: addr.IP, Port: 5656, Conn: &rclb})
	// clientclt := protocol.CreateClusterTable()

	// sends message rclb using json encoder
	clientNode.FindCluster(clusterName)

	// will output from the same node since its using the same buffer from where it wrote to
	// so it doesnt matter and i dont care

	// receives data from clientNode
	connection.MessageHandler(&rclb, receiverNode, recvclt)
	// send back response to find cluster method via writing into recvclit
}

func TestJoinCluster(t *testing.T) {
	var clusterName protocol.ClusterName = "vim-boys"

	addr := net.TCPAddr{IP: net.ParseIP("127.0.0.1")}
	rclb := clusterBuff{buf: make(chan []byte, 1), n: 0}

	// different go routine that sends request to receiver node
	clientNode := protocol.NewNode(protocol.NodeAddr{IP: addr.IP, Port: 3030}, "requester", "")
	clientNode.NeighboringNodes = append(clientNode.NeighboringNodes, protocol.NodeAddr{IP: addr.IP, Port: 5656, Conn: &rclb})
	clientclt := protocol.CreateClusterTable()
	newCluster := protocol.CreateCluster(clusterName)
	(*clientclt)[clusterName] = newCluster
	newCluster.ClusterPeers = append(newCluster.ClusterPeers, protocol.ClusterPeer{
		Addr:   protocol.NodeAddr{IP: addr.IP, Port: 5656},
		Conn:   &rclb,
		NodeID: "receiver",
		Status: protocol.IDLE,
	})

	// invokes RCP method
	err := clientNode.JoinCluster(*newCluster)
	if err != nil {
		protocol.ProtocolErrorWrap(err.Error(), protocol.CALL, protocol.JOIN)
		t.FailNow()
	}

	var wg sync.WaitGroup
	wg.Go(func() { testByteReader(&rclb, clusterName, addr) })

	wg.Wait()
	bodyBuf := make([]byte, 4096)

	var mr connection.MessageReader = connection.MessageReader{
		PayloadBuffer: make([]byte, 0, 4096),
		JsonBuffer:    make([]byte, 0, 4096),
		State:         connection.PROCESSING,
	}

	// Reads reply from receiver
	rclb.Read(bodyBuf)
	// receives data from clientNode
	msg, pl, err := mr.ExtractMessage(bodyBuf)
	if err != nil {
		protocol.ProtocolErrorWrap(err.Error(), protocol.CALL, protocol.JOIN)
		t.FailNow()
	}

	// read and sends back response to find cluster method via writing into recvclt
	err = clientNode.RecvRPCMessage(msg, pl, &rclb, clientclt)

	if err != nil {
		protocol.ProtocolErrorWrap(err.Error(), protocol.CALL, protocol.JOIN)
		t.FailNow()
	}

}

// Writes reply
func testByteReader(rclb *clusterBuff, clusterName protocol.ClusterName, nadd net.TCPAddr) {
	receiverNode := protocol.NewNode(protocol.NodeAddr{IP: nadd.IP, Port: 5656}, "receiver", "")
	recvclt := protocol.CreateClusterTable()
	(*recvclt)[clusterName] = protocol.CreateCluster(clusterName)
	bodyBuf := make([]byte, 4096)

	var mr connection.MessageReader = connection.MessageReader{
		PayloadBuffer: make([]byte, 0, 4096),
		JsonBuffer:    make([]byte, 0, 4096),
		State:         connection.PROCESSING,
	}

	rclb.Read(bodyBuf)
	// receives data from clientNode
	msg, pl, err := mr.ExtractMessage(bodyBuf)
	if err != nil {
		protocol.ProtocolErrorWrap(err.Error(), protocol.CALL, protocol.JOIN)
		return
	}

	// read and sends back response to find cluster method via writing into recvclt
	err = receiverNode.RecvRPCMessage(msg, pl, rclb, recvclt)

	if err != nil {
		protocol.ProtocolErrorWrap(err.Error(), protocol.CALL, protocol.JOIN)
		return
	}
}
