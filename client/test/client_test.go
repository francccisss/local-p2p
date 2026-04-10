package test

import (
	pro "client/protocol"
	utils_test "client/test/utils"
	"fmt"
	"testing"
)

type fileData struct {
	location string
	hash     pro.ClusterName
	name     string
}

func TestClientConn(t *testing.T) {
	testPort := 3030
	testFileData := fileData{"somewhere", "this is the hash of the file", "file.txt"}
	NeighborBootstrap := []pro.Peer{
		pro.Peer{NodeID: "localhost:5656", Status: pro.LEECHING, NodeAddr: pro.NodeAddr{IP: []byte("localhost"), Port: 5656}},
		pro.Peer{NodeID: "localhost:6952", Status: pro.IDLE, NodeAddr: pro.NodeAddr{IP: []byte("localhost"), Port: 6952}},
	}
	UDPConn, err := utils_test.InitUDPConn(testPort)
	if err != nil {
		fmt.Printf("[TEST ERROR]: %s", err)
		t.FailNow()
	}

	client := pro.NewNode(UDPConn, pro.NodeAddr{IP: []byte("localhost"), Port: testPort}, "Pinger", "/files/")

	// bootstrap neighbors retrieved from DHT server
	for _, n := range NeighborBootstrap {
		client.NeighboringNodes = append(client.NeighboringNodes, n)
	}

	err = pro.Ping(client, client.NeighboringNodes, testFileData.hash)
	if err != nil {
		fmt.Printf("[TEST ERROR]: %s", err)
		t.FailNow()
	}

	for {
		buf := make([]byte, 2048)
		n, _, err := client.UDPconn.ReadFromUDP(buf)
		if err != nil {
			fmt.Printf("[TEST ERROR]: %s", err)
			t.FailNow()
		}

		msg, err := pro.ReadRPCMessage(buf[:n])

		err = pro.RecvRPCMessage(client, msg)

		if err != nil {
			fmt.Printf("[TEST ERROR]: %s", err)
			t.FailNow()
		}

	}

}
