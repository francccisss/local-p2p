package main

import (
	"fmt"
	"net"
	"os"
)

type PeerStatus int

const (
	LEECHING PeerStatus = iota
	SEEDING
	IDLE
)

type Peer struct {
	PStatus  PeerStatus
	LStatus  PeerStatus
	NodeAddr NodeAddr
}
type NodeAddr struct {
	IP   []byte
	Port int
}

type Node struct {
	UDPconn   *net.UDPConn
	PeerTable []Peer // TODO Change to map with array of Peer, key is the hash value of the file that is being transffered in the cluster
	Id        string // 16bit len
	Addr      NodeAddr
}

// key for peer table to set status for current cluster
func (n *Node) SetStatus(key string) {

}
func ConcatStr(str *[]string) string {

	tmp := ""
	for _, s := range *str {
		tmp += s
	}
	return tmp

}

func (n *Node) Checkfile(fileKey string, entries []os.DirEntry, programWD *[]string) (os.DirEntry, error) {

	for _, entry := range entries {
		info, err := entry.Info()
		fileName := info.Name()
		fmt.Printf("entry: %s\n", fileName)
		if err != nil {
			fmt.Printf("Error Unable to get info for file: %s\n", fileName)
			fmt.Printf("Reason: %s\n", err)
			continue
		}
		if info.IsDir() {
			currentDirectory := fileName + "/"
			*programWD = append(*programWD, currentDirectory)
			curDirEntries, err := os.ReadDir(ConcatStr(programWD))
			if err != nil {
				*programWD = (*programWD)[:len(*programWD)-1]
				fmt.Printf("Error Unable to Read from Directory: %s\n", fileName)
				fmt.Printf("Reason: %s\n", err)
				continue
			}

			foundEntry, err := n.Checkfile(fileKey, curDirEntries, programWD)
			if err != nil {
				*programWD = (*programWD)[:len(*programWD)-1]
				continue
			}
			return foundEntry, nil
		}
		if fileName == fileKey {
			return entry, nil
		}
	}
	return nil, fmt.Errorf("No entries matching the fileKey.")
}

// Node's internal function not including reponding to rpc requests from other peers
