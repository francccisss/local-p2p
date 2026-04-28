package main

import (
	"client/protocol"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/google/uuid"
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
	jsonBuffer    []byte
	metaJson      protocol.RPCMsg
	payloadSize   int
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

	addr := &net.TCPAddr{IP: ip, Port: port}

	l, err := net.Listen("tcp", ":5656")
	if err != nil {
		fmt.Println(err.Error())
		panic("Shutting down")
	}
	defer l.Close()
	nodeID, err := uuid.NewUUID()
	if err != nil {
		panic(err)
	}

	node := protocol.NewNode(protocol.NodeAddr{IP: addr.IP, Port: addr.Port}, protocol.NodeID(nodeID.String()), FILE_LOCATION)

	// read prefix header first
	bodyBuf := make([]byte, 10)
	var mr MessageReader = MessageReader{
		payloadBuffer: make([]byte, 0, 4096),
		jsonBuffer:    make([]byte, 0, 4096),
		state:         PROCESSING,
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Failed to read data to bodybuf")
			panic(err)
		}
		go func() {
			fmt.Printf("Received TCP Connection: %+v\n", conn)
			for {
				n, err := conn.Read(bodyBuf)
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
					go func() {
						err := protocol.RecvRPCMessage(node, header, pl)
						if err != nil {
							fmt.Println(err)
						}
					}()
				}
			}
		}()
	}

}

func handleConn() {}

func (mr *MessageReader) ExtractMessage(buffer []byte) (protocol.RPCMsg, Payload, error) {

	for {
		switch mr.phase {
		case PREFIX:
			fmt.Println("Prefix Phase")
			mr.metaJsonSize = int(binary.LittleEndian.Uint32(buffer[:protocol.PREFIX_HEADER_SIZE]))
			fmt.Printf("[PREFIX PHASE]: Meta Json length: %d\n", mr.metaJsonSize)
			mr.phase = META_JSON

			buffer = buffer[protocol.PREFIX_HEADER_SIZE:]
			fmt.Printf("left in buffer: %d\n", len(buffer))
		case META_JSON:
			fmt.Printf("Meta Json Phase: %d\n", mr.metaJsonSize)
			remaining := mr.metaJsonSize - len(mr.jsonBuffer) // making sure we dont overextend to the payload sectionA
			if len(buffer) < remaining {
				mr.jsonBuffer = append(mr.jsonBuffer, buffer...)
				if len(mr.jsonBuffer) < mr.metaJsonSize {
					fmt.Printf("Meta json: rem-%d, Delivered/read%d\n", remaining, len(buffer))
					return mr.metaJson, nil, nil
				}
				fmt.Printf("Meta json: rem-%d, Delivered/read%d\n", remaining, len(buffer))
				buffer = buffer[len(buffer):] // push index to end
			} else {
				// NOTE: This leaves extra bytes for the next phase
				mr.jsonBuffer = append(mr.jsonBuffer, buffer[:remaining]...) // ga subra ko index tungod sa buffer access

				fmt.Printf("Meta json: rem-%d, Delivered/read%d\n", remaining, len(buffer))
				fmt.Printf("Rem in Buffer After :%d\n", len(buffer[remaining:]))
				buffer = buffer[remaining:]
			}
			fmt.Printf("Read Buffer Len: %d\n", len(mr.jsonBuffer))
			fmt.Printf("[META PHASE]: Meta json content buffer size: %d\n", mr.metaJsonSize)
			err := json.Unmarshal(mr.jsonBuffer, &mr.metaJson)
			if err != nil {
				fmt.Println("[META PHASE]: Unable to unmarshal meta data json")
				return mr.metaJson, nil, err
			}
			fmt.Printf("Meta data received %+v\n", mr.metaJson)
			mr.payloadSize = int(mr.metaJson.PayloadSize)
			mr.phase = PAYLOAD

			fmt.Printf("Buffer len for next phase: %d\n", len(buffer))

			fmt.Printf("Bytes left to read: %d\n", (mr.metaJsonSize+protocol.PREFIX_HEADER_SIZE+mr.payloadSize)-(mr.metaJsonSize+protocol.PREFIX_HEADER_SIZE))
		case PAYLOAD:

			remaining := mr.payloadSize - len(mr.payloadBuffer)
			if len(buffer) < remaining {
				fmt.Printf("Remaining bytes from payload: %d, received: %d\n", remaining, len(buffer))
				mr.payloadBuffer = append(mr.payloadBuffer, buffer...)
				// guard
				fmt.Printf("Remaining: %d, IN: %d\n", remaining, len(mr.payloadBuffer))
				if len(mr.payloadBuffer) < mr.payloadSize {
					return mr.metaJson, nil, nil
				}
			} else {
				mr.payloadBuffer = append(mr.payloadBuffer, buffer[:remaining]...)
			}

			mr.phase = PREFIX
			mr.state = DONE
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
