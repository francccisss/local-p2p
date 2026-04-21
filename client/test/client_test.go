package test

import (
	"client/protocol"
	"fmt"
	"net"
	"testing"
)

func TestClientConnection(t *testing.T) {

	addr := net.UDPAddr{IP: []byte("localhost"), Port: 3030}
	l, err := net.ListenUDP("udp", &addr)
	if err != nil {
		fmt.Println("Failed to create udp listener")
		t.FailNow()
	}
	msg := protocol.RPCMsg{
		Method: protocol.PING,
		RPCType: protocol.CALL,
		StatusCode: 0,
		Message: "Ni hao",
		PayloadSize: 0,
	}
	b,err:= protocol.WrapPayloadToBuffer(msg,nil)

	if err != nil {
		fmt.Println("Failed to create udp listener")
		t.FailNow()}
	protocol.SendMsg(l,,protocol.NodeAddr{})

}
