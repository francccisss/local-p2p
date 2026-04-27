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
	payloadBuffer []byte
	readBuffer    []byte
	metaJson      protocol.RPCMsg
	payloadSize   int
	phaseDelim    int // is the previous block of bytes used for previous phase
	metaJsonSize  int
	phase         PHASE
	state         MessageReaderState
}
type Payload []byte

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
	bodyBuf := make([]byte, 1024)
	var mr MessageReader = MessageReader{
		payloadBuffer: make([]byte, 0, 4096),
		readBuffer:    make([]byte, 0, 4096),
		state:         PROCESSING,
	}
	for {
		n, err := connReader.Read(bodyBuf)
		if err != nil {
			fmt.Println("Failed to read data to bodybuf")
			panic(err)
		}
		header, pl, err := mr.ExtractMessage(bodyBuf[:n])
		if err != nil {
			fmt.Println("Failed to extract message")
			fmt.Printf("ERROR: %s\n", err)
			mr = MessageReader{}
			continue
		}
		if mr.state == DONE {
			fmt.Println("Process rpc message and payload")
			fmt.Printf("Header extracted %+v\n", header)
			fmt.Printf("Payload contains: %s", pl)

			// #TODO: Handle payload and rpc message for methods
			// go func() {
			// 	err := protocol.RecvRPCMessage(nil, header, pl)
			// 	if err != nil {
			// 		fmt.Println(err)
			// 	}
			// }()
			fmt.Printf("Payload Message: %s\n", pl)
		}
	}

}

func (mr *MessageReader) ExtractMessage(buffer []byte) (protocol.RPCMsg, Payload, error) {

	for {
		switch mr.phase {
		case PREFIX:
			fmt.Println("Prefix Phase")
			mr.metaJsonSize = int(binary.LittleEndian.Uint32(buffer[:protocol.PREFIX_HEADER_SIZE]))
			fmt.Printf("[PREFIX PHASE]: Meta Json length: %d\n", mr.metaJsonSize)
			mr.phaseDelim = protocol.PREFIX_HEADER_SIZE
			mr.phase = META_JSON

			buffer = buffer[mr.phaseDelim:]
		case META_JSON:
			fmt.Printf("Meta Json Phase: %d\n", mr.metaJsonSize)
			// the payload will read the readBuffer if it doesnt fit then
			// get the rest from the buffer argument on the if statements

			if len(buffer) < mr.metaJsonSize {
				remaining := mr.metaJsonSize - len(mr.readBuffer) // making sure we dont overextend to the payload section
				if remaining > 0 {
					fmt.Printf("Remaining Meta json bytes: %d, delivered: %d\n", remaining, len(buffer))
					// looks for the minimum value between remaining window and the received buffer
					// exceeding buffersize access
					minValue := min(remaining, len(buffer))
					fmt.Printf("Min value: %d\n", minValue)
					mr.readBuffer = append(mr.readBuffer, buffer[:minValue]...) // ga subra ko index tungod sa buffer access
					return mr.metaJson, nil, nil
				}
			} else {
				mr.readBuffer = append(mr.readBuffer, buffer[:mr.metaJsonSize]...) // ga subra ko index tungod sa buffer access
			}
			fmt.Printf("Read Buffer Len: %d\n", len(mr.readBuffer))
			fmt.Printf("[META PHASE]: Meta json content buffer size: %d\n", mr.metaJsonSize)
			err := json.Unmarshal(mr.readBuffer, &mr.metaJson)
			if err != nil {
				fmt.Println("[META PHASE]: Unable to unmarshal meta data json")
				return mr.metaJson, nil, err
			}
			fmt.Printf("Meta data received %+v\n", mr.metaJson)
			mr.payloadSize = int(mr.metaJson.PayloadSize)
			mr.phase = PAYLOAD
			mr.phaseDelim += mr.metaJsonSize
			// updating this for the sequential transition between phases
			// so that the next phase does not need to read from mr.phaseDelim
			buffer = buffer[mr.phaseDelim:]
		case PAYLOAD:
			fmt.Printf("DIPOTA: len %d, str: %s\n", len(buffer), buffer)
			fmt.Printf("Payload Phase: %d\n", mr.payloadSize)

			if len(buffer) < mr.payloadSize {
				remaining := mr.payloadSize - len(mr.payloadBuffer)
				minValue := min(remaining, len(buffer))
				fmt.Printf("Remaining bytes from payload: %d, received: %d\n", remaining, len(buffer))
				if remaining > 0 {
					mr.payloadBuffer = append(mr.payloadBuffer, buffer[:minValue]...)
					return mr.metaJson, nil, nil
				}

			} else {
				mr.payloadBuffer = append(mr.payloadBuffer, buffer[:mr.payloadSize]...)
			}

			mr.phase = PREFIX
			mr.state = DONE
			mr.phaseDelim = 0
			mr.payloadSize = 0
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
