package main

import (
	"client/connection"
	"client/protocol"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/google/uuid"
)

const FILE_LOCATION = "/files/"

func main() {

	fmt.Println("Client")
	// args[1] port number, args[2] command, args[3] parameter for command
	args := os.Args[1:]
	print_args()
	ip := net.ParseIP("127.0.0.1") // TODO Change this and grab local IP address
	if len(args) < 1 {
		panic("No port arguments, add a port number")
	}
	port, err := strconv.Atoi(args[0])

	if err != nil {
		fmt.Println(err.Error())
		panic("Shutting down")
	}

	addr := &net.TCPAddr{IP: ip, Port: port}

	fmt.Println(addr.IP)
	l, err := net.Listen("tcp4", addr.String())
	if err != nil {
		fmt.Printf("Bind Error: %s\n", err.Error())
		panic("Shutting down")
	}
	defer l.Close()
	nodeID, err := uuid.NewUUID()
	if err != nil {
		panic(err)
	}

	node := protocol.NewNode(protocol.NodeAddr{IP: addr.IP, Port: addr.Port}, protocol.NodeID(nodeID.String()), FILE_LOCATION)
	newClusterTable := protocol.CreateclusterTable()

	// blocking handler
	// TODO: create context for this function handler so that we can cancel through process interrupts
	err = connection.HandleConn(&l, node, newClusterTable)
	if err != nil {
		fmt.Println(err)

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
