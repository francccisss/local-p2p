package main

import (
	"fmt"
	"net"
	"testing"
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

	err = clientNode.Ping()
	if err != nil {
		panic(err.Error())
	}

}
