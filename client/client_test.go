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
	msg := RPCMsg{
		RPCType:    CALL,
		StatusCode: SUCCESS,
		Method:     PING,
		NodeAddr:   NodeAddr{IP: "localhost", Port: "3030"},
		Payload:    []byte("Ping"),
	}

	err = SendMsg(UDPConn, msg, NodeAddr{IP: "localhost", Port: "5656"})
	if err != nil {
		panic(err.Error())
	}

}
