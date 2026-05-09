package test

import (
	"client/connection"
	"client/protocol"
	"fmt"
	"net"
	"testing"
)

var node *protocol.Node

func TestClientConnection(t *testing.T) {

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
	buf []byte
}

//	type clusterRW interface {
//		Read(p []byte) (n int, err error)
//		Write(p []byte) (n int, err error)
//		Close() error
//	}
func (clb *clusterBuff) Read(p []byte) (n int, err error) {
	copy(p, clb.buf)
	clb.buf = clb.buf[len(p):]
	return len(p), nil
}

func (clb *clusterBuff) Write(p []byte) (n int, err error) {
	copy(clb.buf, p)
	return len(p), nil
}
func (clb *clusterBuff) Close() error {
	return nil
}

func TestClusterSearch(t *testing.T) {
	var clusterName protocol.ClusterName = "vim-boys"

	addr := net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 3030}
	clb := clusterBuff{buf: make([]byte, 4096)}

	// different go routine that sends request to receiver node
	clientNode := protocol.NewNode(protocol.NodeAddr{IP: addr.IP, Port: 3030}, "requester", "")
	clientNode.NeighboringNodes = append(clientNode.NeighboringNodes, protocol.NodeAddr{IP: addr.IP, Port: 5656, Conn: &clb})
	clientNode.FindCluster(clusterName)

	receiverNode := protocol.NewNode(protocol.NodeAddr{IP: addr.IP, Port: 5656}, "receiver", "")
	recvclt := protocol.CreateClusterTable()
	(*recvclt)[clusterName] = protocol.CreateCluster(clusterName)

	for {
		connection.MessageHandler(&clb, receiverNode, recvclt)
	}

}
