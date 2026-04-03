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
	msg := RPCMsg{
		RPCType:    CALL,
		StatusCode: SUCCESS,
		Method:     PING,
		NodeAddr:   clientNode.Addr,
		Payload:    []byte("Ping"),
	}
	raddr := NodeAddr{
		IP:   []byte("localhost"),
		Port: 5656,
	}

	err = SendMsg(UDPConn, msg, raddr)
	if err != nil {
		panic(err.Error())
	}

}
