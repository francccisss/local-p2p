package test

import (
	"client/protocol"
	"fmt"
	"net"
	"testing"
)

func TestClientConnection(t *testing.T) {

	raddr, err := net.ResolveUDPAddr("udp", "localhost"+":"+"3030")
	if err != nil {
		fmt.Println("Failed to resolve UDPADDr")
		t.FailNow()
	}
	l, err := net.ListenUDP("udp", raddr)
	if err != nil {
		fmt.Println("Failed to create udp listener")
		t.FailNow()
	}
	msg := protocol.RPCMsg{
		Method:      protocol.PING,
		RPCType:     protocol.CALL,
		StatusCode:  0,
		Message:     "Ni hao",
		PayloadSize: 0,
	}
	b, err := protocol.WrapPayloadToBuffer(msg, nil)

	if err != nil {
		fmt.Println("Failed to wrap message")
		t.FailNow()
	}
	err = protocol.SendMsg(l, b, protocol.NodeAddr{Port: 5656, IP: []byte("localhost")})
	if err != nil {

		fmt.Println("Failed to send udp message")
		t.FailNow()
	}

}
