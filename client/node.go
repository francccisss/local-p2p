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

func (n *Node) Checkfile(fileKey string) (os.DirEntry, error) {

	programwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error Unable to Read Get Current Working Directory\n")
		fmt.Printf("Reason: %s\n", err)
		return nil, err
	}
	wd := []string{programwd}
	wd = append(wd, FILE_LOCATION)

	entries, err := os.ReadDir(ConcatStr(&wd))

	entry, err := recursiveFileSearch(fileKey, entries, &wd)
	if err != nil {
		return nil, fmt.Errorf("No entries matching the fileKey.")
	}

	return entry, nil
}

func recursiveFileSearch(fileKey string, entries []os.DirEntry, wd *[]string) (os.DirEntry, error) {
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
			*wd = append(*wd, currentDirectory)
			curDirEntries, err := os.ReadDir(ConcatStr(&*wd))
			if err != nil {
				*wd = (*wd)[:len(*wd)-1]
				fmt.Printf("Error Unable to Read from Directory: %s\n", fileName)
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
		if fileName == fileKey {
			return entry, nil
		}
	}
	return nil, fmt.Errorf("No entries matching the fileKey.")
}

// Node's internal function not including reponding to rpc requests from other peers
