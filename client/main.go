package main

import (
	"bufio"
	"client/protocol"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"
)

const FILE_LOCATION = "/files/"

func main() {

	fmt.Println("Client")
	// args[1] port number, args[2] command, args[3] parameter for command
	args := os.Args[1:]
	print_args()
	ip := []byte("") // TODO Change this and grab local IP address
	if len(args) < 1 {
		panic("No port arguments, add a port number")
	}
	port, err := strconv.Atoi(args[0])

	if err != nil {
		fmt.Println(err.Error())
		panic("Shutting down")
	}

	addr := &net.UDPAddr{IP: ip, Port: port}

	UDPConn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println(err.Error())
		panic("Shutting down")
	}
	defer UDPConn.Close()

	headerLen := 4
	var metaJsonLen uint32
	connReader := bufio.NewReader(UDPConn)
	prefixedBuf := make([]byte, headerLen)
	for {
		_, err := connReader.Read(prefixedBuf)
		if err != nil {
			fmt.Println("Failed to read data to prefixedBuf")
			panic(err)
		}
		metaJsonLen = binary.LittleEndian.Uint32(prefixedBuf)
		fmt.Printf("Length of meta json received: %d\n", metaJsonLen)
		fmt.Println("Discarded 4 bytes from prefix header")
		bodyBuf := make([]byte, 10)
		jsonBuf := make([]byte, 0)
		total := 0
		for {
			n, err := connReader.Read(bodyBuf)
			if err != nil {
				fmt.Println("Failed to read data to bodybuf")
				panic(err)
			}

			// using [:n] because old data might stil be present
			jsonBuf = append(jsonBuf, bodyBuf[:n]...)
			total += n
			fmt.Printf("meta json bytes total received: %d, currently read: %d\n", total, n)
			fmt.Printf("meta json buff len: %d\n", len(jsonBuf))
			if total < int(metaJsonLen) {
				continue
			}
			rpc := protocol.RPCMsg{}
			err = json.Unmarshal(jsonBuf, &rpc)
			if err != nil {
				fmt.Println("Failed to Unmarshal json")
				panic(err)
			}
			fmt.Printf("Received rpc meta data: %+v\n", rpc)
			return
		}

	}
}

func print_args() {

	args := os.Args[1:]
	i := 0
	for {
		if i == len(args) {
			break
		}
		fmt.Println(args[i])
		i++
	}
}
