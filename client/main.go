package main

import (
	"fmt"
	"net"
	"os"
)

func main() {

	fmt.Println("Client")
	// args[1] port number, args[2] command, args[3] parameter for command
	print_args()
	ip := []byte("")
	addr := &net.UDPAddr{IP: ip, Port: 5656}

	// addr, err := net.ResolveUDPAddr("udp", "localhost:5656")
	//
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	panic("Shutting down")
	// }

	UDPConn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println(err.Error())
		panic("Shutting down")
	}

	var buffer []byte
	for {
		// does this block the thread
		len, _, err := UDPConn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println(err.Error())
			panic("Unable to handle incoming data")
		}
		fmt.Printf("Bytes read %d\n", len)

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
