package main

import (
	"fmt"
	"net"
)

func main() {

	fmt.Printf("Client send udp")
	laddr, err := net.ResolveUDPAddr("udp", "localhost:3030")

	raddr, err := net.ResolveUDPAddr("udp", "localhost:5656")
	if err != nil {
		panic(err.Error())
	}
	UDPConn, err := net.DialUDP("udp", laddr, raddr)
	if err != nil {
		panic(err.Error())
	}
	defer UDPConn.Close()
	msg := RPCMsg{
		RPCType:  CALL,
		NodeAddr: NodeAddr{IP: "localhost", Port: "3030"},
		Body:     []byte("Hello BIIIITACH"),
	}

	err = SendMsg(UDPConn, msg, raddr)
	if err != nil {
		panic(err.Error())
	}

}
