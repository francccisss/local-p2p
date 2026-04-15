package test

import (
	"client/protocol"
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"slices"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestDataSegmentation(t *testing.T) {

	n := protocol.Node{
		FILE_LOCATION: "/files/",
	}

	en, path, err := protocol.Checkfile("IosevkaTerm.zip", n.FILE_LOCATION)
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
	var fileLen float64 = float64(len(b))

	// given the length of a file and the pagesize for mmap as 4kb
	// if fileLen > page_size = total segment > 1
	blockSize := int64(os.Getpagesize() * 256)

	fmt.Printf("Chunk size - %d: Bytes, %d: Mega Bytes\n", int(blockSize), int(blockSize/1024))

	ceild := math.Ceil(fileLen / float64(blockSize))

	dataInfo := protocol.DataSegment{
		TotalSegments:   int(ceild),
		SegmentPosition: 0,
	}

	fmt.Printf("Total segments to create: %d\n", dataInfo.TotalSegments)

	// segmentsize will be the pzgesize that can be returned by mmap()
	dsComb := make([][]byte, dataInfo.TotalSegments)

	fd, err := syscall.Open(path+en.Name(), syscall.O_RDONLY, 0644)

	if err != nil {
		fmt.Println("Error Opening file through syscall")
		t.FailNow()
	}
	for range dataInfo.TotalSegments {
		tmpChunk := make([]byte, blockSize)
		fmt.Printf("Segment Pos: %d\n", dataInfo.SegmentPosition)
		fmt.Printf("Segment Size: %d\n", blockSize)

		buf, err := protocol.DataSegmentation(fd, path+en.Name(), int64(dataInfo.SegmentPosition), int64(blockSize))

		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}

		// simulates node retrieving and incrementing segment Position
		rem := min(blockSize, int64(fileLen)-int64(dataInfo.SegmentPosition))
		copy(tmpChunk, buf[:rem])
		dsComb = append(dsComb, tmpChunk)

		err = syscall.Munmap(buf)
		if err != nil {
			fmt.Printf("[TEST ERROR]: %s", err)
			t.Fatalf("[Munmap ERROR]: %s", err)
		}

		dataInfo.SegmentPosition += int(rem)
	}
	//return

	tmp := slices.Concat(dsComb...)

	fmt.Printf("Number of Chunks from DS: %d\n", dataInfo.TotalSegments)
	fmt.Printf("original file len: %d\n", len(b))
	fmt.Printf("transported file len: %d\n", len(tmp))
	err = os.WriteFile("Iozevka.zip", tmp, 0644)
	if err != nil {
		fmt.Println(err.Error())
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
