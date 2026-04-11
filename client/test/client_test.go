package test

import (
	pro "client/protocol"
	"fmt"
	"testing"
)

type fileData struct {
	location string
	hash     pro.ClusterName
	name     string
}

func TestClientPing(t *testing.T) {
	testPort := 3030
	testFileData := fileData{hash: "this is the hash of the file"}
	NeighborBootstrap := []pro.ClusterPeer{
		pro.ClusterPeer{NodeID: "localhost:5656", Addr: pro.NodeAddr{IP: []byte("localhost"), Port: 5656}},
		pro.ClusterPeer{NodeID: "localhost:4500", Addr: pro.NodeAddr{IP: []byte("localhost"), Port: 4500}},
		pro.ClusterPeer{NodeID: "localhost:4269", Addr: pro.NodeAddr{IP: []byte("localhost"), Port: 4269}},
	}
	UDPConn, err := pro.InitUDPConn(testPort)
	if err != nil {
		fmt.Printf("[TEST ERROR]: %s", err)
		t.FailNow()
	}

	client := pro.NewNode(
		UDPConn,
		pro.NodeAddr{
			IP:   []byte("localhost"),
			Port: testPort},
		"Pinger",
		"/files/")

	// bootstrap neighbors retrieved from DHT server
	pro.CreateCluster(client, testFileData.hash)

	c, ok := client.ClusterTable[testFileData.hash]
	if !ok {
		fmt.Printf("[TEST ERROR]: unable to find cluster\n")
		t.FailNow()
	}
	for _, n := range NeighborBootstrap {
		c.ClusterPeers = append(c.ClusterPeers, n)
	}
	client.ClusterTable[testFileData.hash] = c

	err = pro.Ping(client, testFileData.hash)
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
