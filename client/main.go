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
	bodyBuf := make([]byte, 4096)
	var mr MessageReader
	mr.readBuffer = make([]byte, 4096)
	mr.payloadBuffer = make([]byte, 4096)
	for {
		n, err := connReader.Read(bodyBuf)
		if err != nil {
			fmt.Println("Failed to read data to bodybuf")
			panic(err)
		}
		_, pl, err := mr.ExtractMessage(bodyBuf[:n])
		if err != nil {
			fmt.Println("Failed to extract message")
			panic(err)
		}
		if mr.state == DONE {
			fmt.Println("Process rpc message and payload")
			fmt.Printf("Payload contains: %s", pl)

			// #TODO: Handle payload and rpc message for methods
			// go protocol.RecvRPCMessage()
		}
	}

}

type PHASE int

const (
	PREFIX PHASE = iota
	META_JSON
	PAYLOAD
)

type MessageReaderState int

const (
	DONE MessageReaderState = iota
	PROCESSING
)

type MessageReader struct {
	payloadBuffer      []byte
	readBuffer         []byte
	metaJson           protocol.RPCMsg
	currentPayloadSize int
	phaseDelim         int // is the previous block of bytes used for previous phase
	metaJsonSize       int
	phase              PHASE
	state              MessageReaderState
}
type Payload []byte

func (mr *MessageReader) ExtractMessage(buffer []byte) (protocol.RPCMsg, Payload, error) {
	// Payload could exceed the buffer size set outside the function
	// so if the remaining buffer < payload size needed from the first arrival of the bytes message
	// in the socket (which is already handled in case PAYLOAD)
	// this conditional phasement continues the loop by accumulating the incoming bytes
	// from the buffer of the current message
	fmt.Printf("Total buffer size read: %d\n", len(buffer))
	if mr.phase == PAYLOAD {
		mr.currentPayloadSize += len(buffer)
		fmt.Printf("Extracted: %d bytes, Needed: %d", mr.currentPayloadSize, mr.metaJson.PayloadSize)
		if mr.currentPayloadSize < int(mr.metaJson.PayloadSize) {
			return mr.metaJson, nil, nil
		}
		mr.phase = PREFIX
		mr.state = DONE
		return mr.metaJson, mr.payloadBuffer, nil
	}

	for {
		switch mr.phase {
		case PREFIX:
			fmt.Println("Prefix Phase")
			mr.metaJsonSize = int(binary.LittleEndian.Uint32(buffer[:protocol.PREFIX_HEADER_SIZE]))
			fmt.Printf("[PREFIX PHASE]: Meta Json length: %d\n", mr.metaJsonSize)
			mr.phaseDelim = protocol.PREFIX_HEADER_SIZE
			mr.phase = META_JSON
		case META_JSON:
			fmt.Println("Meta json Phase")

			fmt.Printf("[META PHASE]: Meta json content buffer size: %d\n", len(buffer[mr.phaseDelim:mr.phaseDelim+mr.metaJsonSize]))
			err := json.Unmarshal(buffer[mr.phaseDelim:mr.phaseDelim+mr.metaJsonSize], &mr.metaJson)
			if err != nil {
				fmt.Println("[META PHASE]: Unable to unmarshal meta data json")
				return mr.metaJson, nil, err
			}
			fmt.Printf("Meta data received %+v\n", mr.metaJson)
			mr.phase = PAYLOAD
			mr.phaseDelim += mr.metaJsonSize
		case PAYLOAD:
			fmt.Println("Payload Phase")
			payloadSectionRemaining := len(buffer[mr.phaseDelim:])
			mr.payloadBuffer = append(mr.payloadBuffer, buffer[mr.phaseDelim:]...)
			mr.currentPayloadSize += payloadSectionRemaining
			mr.state = PROCESSING
			fmt.Printf("Extracted: %d bytes after meta json\n", mr.currentPayloadSize)
			if mr.currentPayloadSize < int(mr.metaJson.PayloadSize) {
				return mr.metaJson, nil, nil
			}
			fmt.Println("First arrival fits buffer setting phase and state to default")
			mr.phase = PREFIX
			mr.state = DONE

			return mr.metaJson, mr.payloadBuffer, nil
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
