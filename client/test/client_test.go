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
	newClusterTable := protocol.CreateclusterTable()
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

func TestClusterSearch(t *testing.T) {

}
