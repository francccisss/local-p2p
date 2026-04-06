package test

import (
	protoClient "client/protocol"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestClientConn(t *testing.T) {

	fmt.Printf("Client send udp")
	UDPConn, err := InitUDPConn("3030")
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}

	defer UDPConn.Close()
	var clientNode protoClient.Node = protoClient.Node{
		UDPconn: UDPConn,
		Addr: protoClient.NodeAddr{
			IP:   []byte("localhost"),
			Port: 3030,
		},
	}
	clientNode.PeerTable = []protoClient.Peer{
		{
			LStatus: protoClient.IDLE,
			PStatus: protoClient.SEEDING,

			NodeAddr: protoClient.NodeAddr{
				IP:   []byte("localhost"),
				Port: 5656,
			},
		},
		{
			LStatus: protoClient.IDLE,
			PStatus: protoClient.SEEDING,

			NodeAddr: protoClient.NodeAddr{
				IP:   []byte("localhost"),
				Port: 4209,
			},
		},
	}

	var wg sync.WaitGroup
	err = protoClient.Ping(&clientNode, &wg)
	if err != nil {
		panic(err.Error())
	}

	time.Sleep(time.Second * 3)
}

func TestFiles(t *testing.T) {

	fmt.Printf("Client send udp")
	UDPConn, err := InitUDPConn("3030")
	if err != nil {
		fmt.Println(err)
		t.Fail()
	}
	defer UDPConn.Close()
	var clientNode protoClient.Node = protoClient.Node{
		UDPconn: UDPConn,
		Addr: protoClient.NodeAddr{
			IP:   []byte("localhost"),
			Port: 3030,
		},
	}
	clientNode.PeerTable = []protoClient.Peer{
		{
			LStatus: protoClient.IDLE,
			PStatus: protoClient.SEEDING,

			NodeAddr: protoClient.NodeAddr{
				IP:   []byte("localhost"),
				Port: 5656,
			},
		},
		{
			LStatus: protoClient.IDLE,
			PStatus: protoClient.SEEDING,

			NodeAddr: protoClient.NodeAddr{
				IP:   []byte("localhost"),
				Port: 4209,
			},
		},
	}
	msg := protoClient.RPCMsg{
		NodeAddr: clientNode.Addr,
		Payload:  []byte("pdd2zwopm2sg1.webp"),
		Method:   protoClient.PROBE,
		RPCType:  protoClient.CALL,
	}
	err = protoClient.SendMsg(clientNode.UDPconn, msg, clientNode.PeerTable[0].NodeAddr)
	if err != nil {
		t.Fatalf("Failed from: %s", err.Error())
	}

}
