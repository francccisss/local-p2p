package main

import (
	"client/protocol"
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

	clientNode := protocol.NewNode(
		UDPConn,
		protocol.NodeAddr{
			IP:   addr.IP,
			Port: addr.Port,
		},
		"Receiver",
		FILE_LOCATION,
	)
	// #TODO: Remove this, this is only for testing
	testFileData := protocol.FileMetaData{Hash: "IosevkaTerm.zip"}
	protocol.CreateCluster(clientNode, protocol.ClusterName(testFileData.Hash))
	// #TODO: Remove this, this is only for testing

	// en, _, err := protocol.Checkfile(testFileData.Hash, FILE_LOCATION)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// fmt.Println(en.Name())
	//
	// return
	if err != nil {
		fmt.Println(err.Error())
		panic("Shutting down")
	}

	var buffer = make([]byte, 4096)
	for {
		n, _, err := UDPConn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println(err.Error())
			panic("Unable to handle incoming data")
		}

		if n < 1 {
			fmt.Println("Empty")
			break
		}

		rpcMsg, err := protocol.ReadRPCMessage(buffer[:n])
		if err != nil {
			fmt.Println(err.Error())
			panic("Unable to handle incoming data")
		}
		fmt.Printf("Recevied Data: %+v\n", rpcMsg)
		fmt.Printf("Body Contents: %s\n", rpcMsg.Payload)
		err = protocol.RecvRPCMessage(clientNode, rpcMsg)

		if err != nil {
			fmt.Println(err.Error())
			panic("Unable to handle incoming data")
		}

	}
	fmt.Println("Done")

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
