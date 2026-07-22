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
	// it reads all of clb.buf into rb
	rb := <-clb.buf
	fmt.Printf("[ TEST ]: READ FROM BUF TO P %d\n", len(rb))
	readBytes := copy(p, rb[:clb.n])
	return readBytes, nil
}

func (clb *dummyCommChannel) Write(p []byte) (n int, err error) {
	fmt.Printf("[ TEST ]: WROTE TO BUFF %d\n", len(p))
	clb.buf <- p
	clb.n = len(p)
	return len(p), nil
}
func (clb *dummyCommChannel) Close() error {
	return nil
}

var receiverNode = protocol.NewNode(protocol.NodeAddr{IP: net.ParseIP("127.0.0.1"), Port: 5656}, "receiver", "/files/")

const fileHash_test = "what.txt"

func TestClusterSearch(t *testing.T) {
	fmt.Println("[Begin Test]: Test Cluster Search")
	var clusterName = "vim-boys"

	addr := net.TCPAddr{IP: net.ParseIP("127.0.0.1")}
	dchl := dummyCommChannel{buf: make(chan []byte, 1), n: 0}

	// different go routine that sends request to receiver node
	clientNode := protocol.NewNode(protocol.NodeAddr{IP: addr.IP, Port: 3030}, "requester", "")
	clientNode.NeighboringNodes = append(clientNode.NeighboringNodes, protocol.NodeAddr{IP: addr.IP, Port: 5656, Conn: &dchl})
	clientclt := protocol.CreateClusterTable()

	//  writes to dchl
	clientNode.FindCluster(clusterName)
	ConsumerNodeReaderWriterToChannel(*receiverNode, &dchl, protocol.FileMetaData{Hash: clusterName})

	bodyBuf := make([]byte, 4096)

	var mr connection.MessageReader = connection.MessageReader{
		PayloadBuffer: make([]byte, 0, 4096),
		JSONBuffer:    make([]byte, 0, 4096),
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
	var clusterName = "vim-boys"

	addr := net.TCPAddr{IP: net.ParseIP("127.0.0.1")}
	dchl := dummyCommChannel{buf: make(chan []byte, 1), n: 0}

	// different go routine that sends request to receiver node
	clientNode := protocol.NewNode(protocol.NodeAddr{IP: addr.IP, Port: 3030}, "requester", "")

	// after FIND_CLUSTR
	clientclt := protocol.CreateClusterTable()

	newCluster := protocol.CreateCluster(protocol.FileMetaData{Hash: clusterName})

	(*clientclt)[clusterName] = newCluster

	newCluster.ClusterPeers = append(newCluster.ClusterPeers, protocol.ClusterPeer{
		Addr:   protocol.NodeAddr{IP: addr.IP, Port: 5656},
		Conn:   &dchl,
		NodeID: "receiver",
		Status: protocol.IDLE,
	})

	err := clientNode.JoinCluster(protocol.FileMetaData{Hash: clusterName}, clientclt)
	// invokes RCP method
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	ConsumerNodeReaderWriterToChannel(*receiverNode, &dchl, protocol.FileMetaData{Hash: clusterName})

	bodyBuf := make([]byte, 4096)

	var mr connection.MessageReader = connection.MessageReader{
		PayloadBuffer: make([]byte, 0, 4096),
		JSONBuffer:    make([]byte, 0, 4096),
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
		fmt.Println(err)
		t.FailNow()
	}
	cl := (*clientclt)[clusterName]

	if len(cl.ClusterPeers) == 0 {
		t.Fatalf("Cluster was not added")
	}
	for _, cp := range cl.ClusterPeers {
		fmt.Printf("Cluster Peer: %+v\n", cp)
	}
}

func TestProbeFile(t *testing.T) {
	fmt.Println("[Begin Test]: Test Cluster Search")
	var clusterName = "vim-boys"

	addr := net.TCPAddr{IP: net.ParseIP("127.0.0.1")}
	dchl := dummyCommChannel{buf: make(chan []byte, 1), n: 0}

	// different go routine that sends request to receiver node
	clientNode := protocol.NewNode(protocol.NodeAddr{IP: addr.IP, Port: 3030}, "requester", "")
	clientclt := protocol.CreateClusterTable()
	newCluster := protocol.CreateCluster(protocol.FileMetaData{Hash: clusterName})

	(*clientclt)[clusterName] = newCluster

	receiverPeer := newCluster.NewClusterPeer(receiverNode.Addr, receiverNode.NodeID, &dchl, protocol.IDLE)
	newCluster.AppendClusterPeer(*receiverPeer)
	fmt.Printf("%+v\n", (*clientclt)[clusterName])

	if err := clientNode.ProbeFile(clusterName, clientclt); err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	ConsumerNodeReaderWriterToChannel(*receiverNode, &dchl, protocol.FileMetaData{Hash: clusterName})

	bodyBuf := make([]byte, 4096)

	var mr connection.MessageReader = connection.MessageReader{
		PayloadBuffer: make([]byte, 0, 4096),
		JSONBuffer:    make([]byte, 0, 4096),
		State:         connection.PROCESSING,
	}

	// Reads reply from receiver
	fmt.Println("Waiting for message from Consumer")
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
		fmt.Println(err)
		t.FailNow()
	}

}

func TestDataDelivery(t *testing.T) {

	addr := protocol.NodeAddr{IP: net.ParseIP("127.0.0.1"), Port: 5656}

	dhcl := dummyCommChannel{
		buf: make(chan []byte, 1),
		n:   0,
	}

	// receives meta data from the DHT server
	metaData, err := protocol.NewFileMetaData("./files/new_dir/text.txt")
	if err != nil {
		t.Fatal(err.Error())
	}

	requestingNode := protocol.NewNode(addr, "requester", "/files/")
	clt := protocol.CreateClusterTable()
	(*clt)[(metaData.Hash)] = protocol.CreateCluster(metaData)
	cluster := (*clt)[(metaData.Hash)]

	cpeer := cluster.NewClusterPeer(receiverNode.Addr, receiverNode.NodeID, &dhcl, protocol.SEEDING)
	cluster.AppendClusterPeer(*cpeer)

	cluster.BitField.FillBits()

	requestingNode.Leech(protocol.FileRequest{
		Hash:     metaData.Hash,
		Interest: 0,
	}, clt, true)

	connection.MessageHandler(&dhcl, requestingNode, clt)
}

// Writes reply to a dummy channel

// Writes reply to a dummy channel
func ConsumerNodeReaderWriterToChannel(n protocol.Node, dchl *dummyCommChannel, fmd protocol.FileMetaData) {
	recvclt := protocol.CreateClusterTable()
	(*recvclt)[fmd.Hash] = protocol.CreateCluster(fmd)
	bodyBuf := make([]byte, 4096)

	var mr connection.MessageReader = connection.MessageReader{
		PayloadBuffer: make([]byte, 0, 4096),
		JSONBuffer:    make([]byte, 0, 4096),
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
	err = n.RecvRPCMessage(msg, pl, dchl, recvclt)

	if err != nil {
		protocol.ProtocolErrorWrap(err.Error(), protocol.CALL, protocol.JOIN)
		return
	}
}
