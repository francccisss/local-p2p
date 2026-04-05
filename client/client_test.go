package main

import (
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

func TestClientConn(t *testing.T) {

	fmt.Printf("Client send udp")
	laddr, err := net.ResolveUDPAddr("udp", "localhost:3030")

	if err != nil {
		panic(err.Error())
	}
	UDPConn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		panic(err.Error())
	}
	defer UDPConn.Close()
	var clientNode Node = Node{
		UDPconn: UDPConn,
		Addr: NodeAddr{
			IP:   []byte("localhost"),
			Port: 3030,
		},
	}
	clientNode.PeerTable = []Peer{
		{
			LStatus: IDLE,
			PStatus: SEEDING,

			NodeAddr: NodeAddr{
				IP:   []byte("localhost"),
				Port: 5656,
			},
		},
		{
			LStatus: IDLE,
			PStatus: SEEDING,

			NodeAddr: NodeAddr{
				IP:   []byte("localhost"),
				Port: 4209,
			},
		},
	}

	var wg sync.WaitGroup
	err = Ping(&clientNode, &wg)
	if err != nil {
		panic(err.Error())
	}

	time.Sleep(time.Second * 3)
}

func TestFiles(t *testing.T) {

	var clientNode Node
	entries, err := os.ReadDir("./files")

	programWD, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error Unable to Read Get Current Working Directory\n")
		fmt.Printf("Reason: %s\n", err)
		t.FailNow()
	}
	wd := []string{programWD}
	wd = append(wd, FILE_LOCATION)
	entry, err := clientNode.Checkfile("pdd2zwopm2sg1.webp", entries, &wd)
	entryNumber := len(entries)
	fmt.Printf("Entry numbers: %d\n", entryNumber)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	fmt.Printf("Recieved Entry: %s\n", entry.Name())

}
