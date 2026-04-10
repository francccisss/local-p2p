package utils_test

import (
	"fmt"
	"net"
	"strconv"
)

func InitUDPConn(port int) (*net.UDPConn, error) {

	laddr, err := net.ResolveUDPAddr("udp", "localhost:"+strconv.Itoa(port))
	if err != nil {
		fmt.Println("Error Unable to resolve UDP Addr")
		return nil, err
	}
	UDPConn, err := net.ListenUDP("udp", laddr)
	if err != nil {
		fmt.Println("Error Unable to create a listener for UDP packets")
		return nil, err
	}
	return UDPConn, nil

}
