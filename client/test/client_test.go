package test

import (
	"client/protocol"
	"encoding/json"
	"fmt"
	"testing"
)

func TestClientConnection(t *testing.T) {

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
		NodeID:      protocol.NodeID("Sender"),
		Method:      protocol.PING,
		RPCType:     protocol.CALL,
		StatusCode:  0,
		Message:     "Ni hao",
		PayloadSize: uint32(len(pngBuf)),
	}

	fmt.Printf("Payload size: %d\n", len(pngBuf))
	buf, err := protocol.WrapPayloadToBuffer(msg, pngBuf)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	err = protocol.SendMsg(buf, protocol.NodeAddr{Port: 5656, IP: []byte("localhost")})
	if err != nil {
		fmt.Println("Failed to send udp message")
		t.FailNow()
	}

}
