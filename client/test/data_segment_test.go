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

	finfo, err := en.Info()
	if err != nil {
		fmt.Println("Error Opening file through syscall")
		t.FailNow()
	}
	// sent by peer to request file
	dfinfo := protocol.FileRequest{
		Hash:      "IosevkaTerm.zip",
		Size:      finfo.Size(),
		Offset:    0,
		BlockSize: int64(os.Getpagesize() * 256),
	}

	// this is all set from reading the meta file from DHT Server
	dfinfo.Segments = int64(math.Ceil(float64(dfinfo.Size) / float64(dfinfo.BlockSize)))

	fmt.Printf("Chunk size - %d: Bytes, %d: Mega Bytes\n", int(dfinfo.BlockSize), int(dfinfo.BlockSize/1024))

	fmt.Printf("Total segments to create: %d\n", dfinfo.Segments)

	// segmentsize will be the pzgesize that can be returned by mmap()
	// simulates node retrieving and incrementing segment Position
	dsComb := make([][]byte, dfinfo.Segments)

	ds := protocol.DataSegment{}

	for range dfinfo.Segments {
		fmt.Printf("Segment Pos: %d\n", dfinfo.Offset)
		fmt.Printf("Segment Size: %d\n", dfinfo.BlockSize)

		ds, err = protocol.CreateDataSegment(path+en.Name(), &dfinfo)

		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}

		dsComb = append(dsComb, ds.DataChunk)
		fmt.Println("--------------------------------------------------")
		fmt.Printf("Segment Number: %d\n", ds.SegmentPosition)
		fmt.Println("--------------------------------------------------")
	}

	tmp := slices.Concat(dsComb...)

	fmt.Printf("Number of Chunks from DS: %d\n", dfinfo.Segments)
	fmt.Printf("original file len: %d\n", len(b))
	fmt.Printf("transported file len: %d\n", len(tmp))
	err = os.WriteFile("./tmp/Iozevka.zip", tmp, 0644)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	fmt.Printf("DS.Hash: %s\n", ds.ClusterName)
	fmt.Printf("DS.TotalSegments: %d\n", ds.TotalSegments)
	fmt.Printf("DS.SegmentPosition: %d\n", ds.SegmentPosition)

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
