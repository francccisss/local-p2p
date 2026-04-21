package test

import (
	"client/protocol"
	"encoding/json"
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
	pngMsg := protocol.PingMessage{
		Status:      protocol.IDLE,
		ClusterName: "cluster",
	}
	pngBuf, err := json.Marshal(pngMsg)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	msg := protocol.RPCMsg{
		Method:      protocol.PING,
		RPCType:     protocol.CALL,
		StatusCode:  0,
		Message:     "Ni hao",
		PayloadSize: uint32(len(pngBuf)),
	}
	buf, err := protocol.WrapPayloadToBuffer(msg, pngBuf)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	err = protocol.SendMsg(l, buf, protocol.NodeAddr{Port: 5656, IP: []byte("localhost")})
	if err != nil {
		fmt.Println("Failed to send udp message")
		t.FailNow()
	}

}
