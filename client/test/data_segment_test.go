package test

import (
	"client/protocol"
	"context"
	"fmt"
	"io/fs"
	"math/rand"
	"os"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestDataSegmentation(t *testing.T) {

	n := protocol.Node{
		FILE_LOCATION: "/files/",
	}
	en, path, err := protocol.Checkfile("newfile.webp", n.FILE_LOCATION)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	b, err := os.ReadFile(path + en.Name())
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	ds, err := protocol.DataSegmentation(b, 10)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	var tmp [][]byte
	for _, d := range ds {
		fmt.Printf("Segment #%d\nData: %+v\n", d.SegmentNum, d)
		tmp = append(tmp, d.DataChunk)
	}
	conctData := slices.Concat(tmp...)
	fmt.Printf("Data len from segment: %d\n", len(conctData))

	var fm fs.FileMode

	fm |= fs.ModePerm

	err = os.WriteFile("prettychill.webp", conctData, fm)

	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
}

func TestFileProbe(t *testing.T) {
	testPort := 3030
	testFileData := fileData{hash: "newfile.webp"}
	UDPConn, err := protocol.InitUDPConn(testPort)
	if err != nil {
		fmt.Printf("[TEST ERROR]: %s", err)
		t.FailNow()
	}

	client := protocol.NewNode(
		UDPConn,
		protocol.NodeAddr{
			IP:   []byte("localhost"),
			Port: testPort},
		"Snicker",
		"/files/")

	// bootstrap neighbors retrieved from DHT server
	protocol.CreateCluster(client, testFileData.hash)

	c, ok := (*client.ClusterTable)[testFileData.hash]
	if !ok {
		fmt.Printf("[TEST ERROR]: unable to find cluster\n")
		t.FailNow()
	}
	c.ClusterPeers = append(c.ClusterPeers, protocol.ClusterPeer{NodeID: "localhost:5656", Addr: protocol.NodeAddr{IP: []byte("localhost"), Port: 5656}})
	protocol.Probe(client, c.ClusterName)
	buff := make([]byte, 4096)
	for {

		n, _, err := client.UDPconn.ReadFromUDP(buff)
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
		msg, err := protocol.ReadRPCMessage(buff[:n])
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
		err = protocol.RecvRPCMessage(client, msg)

		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
	}

}

func TestClusterThreads(t *testing.T) {
	var cname protocol.ClusterName = "newfile.webp"
	initClt := make(protocol.ClusterTable)
	node := protocol.Node{
		ClusterTable: &initClt,
	}
	protocol.CreateCluster(&node, cname)
	clt, _ := (*node.ClusterTable)[cname]

	peerArr := []protocol.ClusterPeer{{NodeID: "peer1"}, {NodeID: "peer2"}, {NodeID: "peer3"}}

	for _, p := range peerArr {
		clt.NewClusterPeerThread(p.NodeID)
		clt.NewClusterPeer(protocol.NodeAddr{}, p.NodeID)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	for _, p := range peerArr {
		cpt := clt.ClusterPeerThreads[p.NodeID]
		func() {
			go clt.SpawnPeerThreads(&ctx, cpt)
		}()
	}
	fmt.Println("[TEST]: Spawning Peer Threads")
	wg.Wait()
	fmt.Println("[TEST]: Sending request to peers")

	// if timed out wont return test
	for _, p := range peerArr {
		go func() {

			cpt := clt.ClusterPeerThreads[p.NodeID]
			cpt.TimeSince = time.Now()
			cpt.BytesReceived = 450
			time.Sleep(time.Second * time.Duration(rand.Intn(10)))
			cpt.NodeIDChann <- p.NodeID
		}()
	}

	fmt.Println("[TEST]: Waiting for peers to finish")
	time.Sleep(time.Second * 20)
	for _, p := range peerArr {
		cpt := clt.ClusterPeerThreads[p.NodeID]
		fmt.Printf("[TEST]: %s | bytes received-%d | avg bytes-%d\n", p.NodeID, cpt.BytesReceived, cpt.AverageBytes)
	}
}
