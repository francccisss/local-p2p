package main

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

func TestClientConn(t *testing.T) {

	fmt.Printf("Client send udp")
	laddr, err := net.ResolveUDPAddr("udp", "localhost:3030")

	if err != nil {
		panic(err.Error())
	}
	UDPConn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		panic(err.Error())
	}
	defer UDPConn.Close()
	var clientNode Node = Node{
		UDPconn: UDPConn,
		Addr: NodeAddr{
			IP:   []byte("localhost"),
			Port: 3030,
		},
	}
	clientNode.PeerTable = []Peer{
		{
			LStatus: IDLE,
			PStatus: SEEDING,

			NodeAddr: NodeAddr{
				IP:   []byte("localhost"),
				Port: 5656,
			},
		},
		{
			LStatus: IDLE,
			PStatus: SEEDING,

			NodeAddr: NodeAddr{
				IP:   []byte("localhost"),
				Port: 4209,
			},
		},
	}

	var wg sync.WaitGroup
	err = Ping(&clientNode, &wg)
	if err != nil {
		panic(err.Error())
	}

	time.Sleep(time.Second * 3)
}

func TestFiles(t *testing.T) {

	fmt.Printf("Client send udp")
	laddr, err := net.ResolveUDPAddr("udp", "localhost:3030")

	if err != nil {
		panic(err.Error())
	}
	UDPConn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		panic(err.Error())
	}
	defer UDPConn.Close()
	var clientNode Node = Node{
		UDPconn: UDPConn,
		Addr: NodeAddr{
			IP:   []byte("localhost"),
			Port: 3030,
		},
	}
	clientNode.PeerTable = []Peer{
		{
			LStatus: IDLE,
			PStatus: SEEDING,

			NodeAddr: NodeAddr{
				IP:   []byte("localhost"),
				Port: 5656,
			},
		},
		{
			LStatus: IDLE,
			PStatus: SEEDING,

			NodeAddr: NodeAddr{
				IP:   []byte("localhost"),
				Port: 4209,
			},
		},
	}
	msg := RPCMsg{
		NodeAddr: clientNode.Addr,
		Payload:  []byte("pdd2zwopm2sg1.webp"),
		Method:   PROBE,
		RPCType:  CALL,
	}
	err = SendMsg(clientNode.UDPconn, msg, clientNode.PeerTable[0].NodeAddr)
	if err != nil {
		t.Fatalf("Failed from: %s", err.Error())
	}

}
