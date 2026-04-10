package protocol

import (
	"client/utils"
	"fmt"
	"net"
	"os"
	"time"
)

type PeerStatus int

const (
	LEECHING PeerStatus = iota
	SEEDING
	IDLE
)

type NodeID string

type Peer struct {
	Status   PeerStatus
	NodeAddr NodeAddr
	NodeID   NodeID
}

type ClusterName string

type PeerThread struct {
	timeSince     time.Time
	NodeIDChann   chan NodeID
	ClusterName   ClusterName
	averageBytes  int
	bytesReceived int
}

type Cluster struct {
	Status      PeerStatus // status for current cluster of current peer
	PeerThreads map[NodeID]PeerThread
	ClusterName
	Peers []Peer
}

// ClusterTable is used locally for handling threads spawned for different
// peers
type ClusterTable map[ClusterName]Cluster

type NodeAddr struct {
	IP   []byte
	Port int
}

type Node struct {
	UDPconn          *net.UDPConn
	NeighboringNodes []Peer
	NodeID           NodeID
	Addr             NodeAddr
	FILE_LOCATION    string
	ClusterTable     ClusterTable
}

func NewNode(conn *net.UDPConn, addr NodeAddr, nodeID NodeID, fileLoc string) *Node {
	cl := make(ClusterTable)
	return &Node{
		UDPconn:          conn,
		Addr:             addr,
		NodeID:           nodeID,
		FILE_LOCATION:    fileLoc,
		NeighboringNodes: make([]Peer, 10),
		ClusterTable:     cl,
	}

}

// key for peer table to set status for current cluster
func (n *Node) SetStatus(key string) {

}

// Node's internal function not including reponding to rpc requests from other peers

func (n *Node) Checkfile(fileKey string, FILE_LOCATION string) (os.DirEntry, string, error) {

	programwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error Unable to Read Get Current Working Directory\n")
		fmt.Printf("Reason: %s\n", err)
		return nil, "", err
	}
	wd := []string{programwd}
	wd = append(wd, FILE_LOCATION)

	entries, err := os.ReadDir(utils.ConcatStr(&wd))

	entry, err := recursiveFileSearch(fileKey, entries, &wd)

	if err != nil {
		return nil, "", fmt.Errorf("No entries matching the fileKey.")
	}

	return entry, utils.ConcatStr(&wd), nil
}

// initializes `wd` with the current working directory of the program
// appended with the file location of the user and as `entries` array is iterated
// and if the current entry is a Directory the `wd` is appended with the current name
// of the directory, and if not then continue.
// If the current file is not a directory and matches the `fileKey` the return the entry of that file
func recursiveFileSearch(fileKey string, entries []os.DirEntry, wd *[]string) (os.DirEntry, error) {
	for _, entry := range entries {
		info, err := entry.Info()
		entryName := info.Name()
		fmt.Printf("entry: %s\n", entryName)
		if err != nil {
			fmt.Printf("Error Unable to get info for file: %s\n", entryName)
			fmt.Printf("Reason: %s\n", err)
			continue
		}
		if !info.IsDir() {
			if entryName == fileKey {
				return entry, nil
			}
			continue
		}
		currentDirectory := entryName + "/"
		*wd = append(*wd, currentDirectory)
		curDirEntries, err := os.ReadDir(utils.ConcatStr(wd))
		if err != nil {
			*wd = (*wd)[:len(*wd)-1]
			fmt.Printf("Error Unable to Read from Directory: %s\n", entryName)
			fmt.Printf("Reason: %s\n", err)
			continue
		}
		foundEntry, err := recursiveFileSearch(fileKey, curDirEntries, wd)
		if err != nil {
			*wd = (*wd)[:len(*wd)-1]
			continue
		}
		return foundEntry, nil
	}
	return nil, fmt.Errorf("No entries matching the fileKey.")
}

func NewPeerThread(cname ClusterName) PeerThread {
	return PeerThread{
		ClusterName: cname,
		NodeIDChann: make(chan NodeID),
		timeSince:   time.Now(),
	}
}
