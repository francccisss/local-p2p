package test

import (
	"client/protocol"
	clientProtocol "client/protocol"
	"fmt"
	"io/fs"
	"os"
	_ "os"
	"slices"
	"testing"
)

func TestDataSegmentation(t *testing.T) {

	n := clientProtocol.Node{
		FILE_LOCATION: "/files/",
	}
	en, path, err := clientProtocol.Checkfile("newfile.webp", n.FILE_LOCATION)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	b, err := os.ReadFile(path + en.Name())
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	ds, err := clientProtocol.DataSegmentation(b, 10)
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
	UDPConn, err := clientProtocol.InitUDPConn(testPort)
	if err != nil {
		fmt.Printf("[TEST ERROR]: %s", err)
		t.FailNow()
	}

	client := clientProtocol.NewNode(
		UDPConn,
		clientProtocol.NodeAddr{
			IP:   []byte("localhost"),
			Port: testPort},
		"Snicker",
		"/files/")

	// bootstrap neighbors retrieved from DHT server
	clientProtocol.CreateCluster(client, testFileData.hash)

	c, ok := (*client.ClusterTable)[testFileData.hash]
	if !ok {
		fmt.Printf("[TEST ERROR]: unable to find cluster\n")
		t.FailNow()
	}
	c.ClusterPeers = append(c.ClusterPeers, clientProtocol.ClusterPeer{NodeID: "localhost:5656", Addr: clientProtocol.NodeAddr{IP: []byte("localhost"), Port: 5656}})
	clientProtocol.Probe(client, c.ClusterName)
	buff := make([]byte, 4096)
	for {

		n, _, err := client.UDPconn.ReadFromUDP(buff)
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
		msg, err := clientProtocol.ReadRPCMessage(buff[:n])
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
		err = clientProtocol.RecvRPCMessage(client, msg)

		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
	}

}

func TestMeasureArrivingBytes(t *testing.T) {
	port := 3030
	UDPConn, err := clientProtocol.InitUDPConn(port)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	clientNode := clientProtocol.NewNode(UDPConn, clientProtocol.NodeAddr{IP: []byte("localhost"), Port: port}, "LeechingNode", "/files/")

	buff := make([]byte, 4096)

	for {
		n, _, err := UDPConn.ReadFromUDP(buff)
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}
		msg, err := protocol.ReadRPCMessage(buff[:n])

		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}

		err = protocol.RecvRPCMessage(clientNode, msg)
		if err != nil {
			fmt.Println(err)
			t.FailNow()
		}

	}

}
