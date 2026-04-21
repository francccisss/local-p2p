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

	connReader := bufio.NewReader(UDPConn)

	// read prefix header first
	var prefixedBuf []byte = make([]byte, 4)
	n, err := connReader.Read(prefixedBuf)
	if err != nil {
		fmt.Println("Failed to read data to prefixedBuf")
		panic(err)
	}
	fmt.Printf("What is n: %d\n", n)
	fmt.Printf("What is prefix value: %d\n", prefixedBuf[:n])
	var metaJsonLen uint32 = binary.LittleEndian.Uint32(prefixedBuf[:n])
	fmt.Printf("Length of meta json received: %d\n", metaJsonLen)
	fmt.Println("Discarded 4 bytes from prefix header")
	// read header first

	bodyBuf := make([]byte, metaJsonLen)
	jsonBuf := make([]byte, 0, metaJsonLen)
	total := 0
	for {
		n, err := connReader.Read(bodyBuf)
		if err != nil {
			fmt.Println("Failed to read data to bodybuf")
			panic(err)
		}

		jsonBuf = append(jsonBuf, bodyBuf[:n]...)
		if len(jsonBuf) < int(metaJsonLen) {
			continue
		}
		break
	}

	fmt.Printf("meta json bytes total received: %d, currently read: %d\n", total, n)
	fmt.Printf("meta json buff len: %d\n", len(jsonBuf))
	rpc := protocol.RPCMsg{}
	err = json.Unmarshal(jsonBuf, &rpc)
	if err != nil {
		fmt.Println("Failed to Unmarshal json")
		panic(err)
	}
	fmt.Printf("Received rpc meta data: %+v\n", rpc)
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
