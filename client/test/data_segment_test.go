package test

import (
	"bytes"
	"client/protocol"
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"
)

func TestDataSegmentation(t *testing.T) {

	n := protocol.Node{
		FILE_LOCATION: "/files/",
	}

	testFile := "IosevkaTerm.zip"
	en, path, err := protocol.Checkfile(testFile, n.FILE_LOCATION)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	b, err := os.ReadFile(path + en.Name())
	fmt.Printf("file len: %d\n", len(b))

	if err != nil {
		fmt.Println("Error Reading file for len")
		t.FailNow()
	}

	finfo, err := en.Info()
	if err != nil {
		fmt.Println("Error Opening file through syscall")
		t.FailNow()
	}
	// sent by peer to request file
	dfinfo := protocol.FileRequest{
		Hash:      testFile,
		Size:      finfo.Size(),
		Offset:    0,
		BlockSize: int64(os.Getpagesize() * 256),
	}

	// this is all set from reading the meta file from DHT Server
	dfinfo.Segments = int64(math.Ceil(float64(dfinfo.Size) / float64(dfinfo.BlockSize)))

	fmt.Printf("Chunk size - %d: Bytes, %d: Mega Bytes\n", int(dfinfo.BlockSize), int(dfinfo.BlockSize/1024))

	fmt.Printf("Total segments to create: %d\n", dfinfo.Segments)

	var db bytes.Buffer

	for range dfinfo.Segments {
		fmt.Printf("Segment Pos: %d\n", dfinfo.Offset)
		fmt.Printf("Segment Size: %d\n", dfinfo.BlockSize)

		ds, headerLen, err := protocol.CreateDataSegment(path+en.Name(), &dfinfo)

		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
		dsBuf := ds.Bytes()
		sh, n := protocol.ParseSegmentHeader(dsBuf[:headerLen])

		fmt.Println("--------------------------------------------------")
		fmt.Printf("Cluster Name: %s\n", sh.ClusterName)
		fmt.Printf("Total Segments: %d\n", dfinfo.Segments)
		fmt.Printf("Segment Position: %d\n", sh.SegmentPosition)
		fmt.Printf("Segment Size: %d\n", sh.SegmentSize)
		fmt.Printf("Payload Size: %d\n", len(dsBuf[n:]))
		db.Write(dsBuf[n:])
		fmt.Println("--------------------------------------------------")
	}

	err = os.WriteFile("./tmp/Iozevka.zip", db.Bytes(), 0644)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
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

func TestLeeching(t *testing.T) {
	fmt.Println("[TEST]: Starting Leecher")
	conn, err := protocol.InitUDPConn(3030)
	if err != nil {
		t.FailNow()
	}
	node := protocol.NewNode(conn, protocol.NodeAddr{IP: []byte("localhost"), Port: 3030}, "Leecher", "/files/")

	// sent by peer to request file
	dfinfo := protocol.FileRequest{
		Hash:      "IosevkaTerm.zip",
		Size:      374780158,
		Offset:    0,
		BlockSize: int64(os.Getpagesize()),
	}

	protocol.CreateCluster(node, protocol.ClusterName(dfinfo.Hash))
	clt := (*node.ClusterTable)[protocol.ClusterName(dfinfo.Hash)]
	clt.NewClusterPeer(protocol.NodeAddr{IP: []byte("localhost"), Port: 5656}, "Receiver")
	clt.NewClusterPeerThread("Receiver")

	err = protocol.Leech(node, protocol.ClusterName(dfinfo.Hash), true, dfinfo)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	for {
		// TODO: Find a way to consume all bytes if exceeds buffer
		buff := make([]byte, 4096*256) // exceeding size of buffer
		// need to be able to know that there are remaining bytes in the
		// kernel buffer that needs to be read from the full
		// communication
		n, _, err := node.UDPconn.ReadFromUDP(buff)
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
		msg, err := protocol.ReadRPCMessage(buff[:n])
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
		err = protocol.RecvRPCMessage(node, msg)
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}

	}
}
