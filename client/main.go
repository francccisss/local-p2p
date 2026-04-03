package main

import (
	"fmt"
	"net"
	"os"
	"strconv"
)

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

	var clientNode Node = Node{
		Addr: NodeAddr{
			IP:   ip,
			Port: port,
		},
	}

	if err != nil {
		fmt.Println(err.Error())
		panic("Shutting down")
	}

	addr := &net.UDPAddr{IP: clientNode.Addr.IP, Port: clientNode.Addr.Port}

	UDPConn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println(err.Error())
		panic("Shutting down")
	}

	var buffer = make([]byte, 4096)
	for {
		// does this block the thread
		n, _, err := UDPConn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println(err.Error())
			panic("Unable to handle incoming data")
		}
		if n < 1 {
			fmt.Println("Empty")
			break

		}

		rpcMsg, err := ReadRPCMessage(buffer[:n])
		if err != nil {
			fmt.Println(err.Error())
			panic("Unable to handle incoming data")
		}
		fmt.Printf("Recevied Data: %+v\n", rpcMsg)
		fmt.Printf("Bodyt Contents: %s\n", rpcMsg.Payload)

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
