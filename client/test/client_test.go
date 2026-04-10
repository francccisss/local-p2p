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
		pro.Peer{Status: pro.LEECHING, NodeAddr: pro.NodeAddr{IP: []byte("localhost"), Port: 5656}},
		pro.Peer{Status: pro.IDLE, NodeAddr: pro.NodeAddr{IP: []byte("localhost"), Port: 6952}},
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
	pro.Ping(client, testFileData.hash)

}
