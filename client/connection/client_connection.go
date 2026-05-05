package connection

import (
	"client/protocol"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
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

func HandleConn(l *net.TCPListener, node *protocol.Node) error {
	bodyBuf := make([]byte, 10)
	var mr MessageReader = MessageReader{
		payloadBuffer: make([]byte, 0, 4096),
		jsonBuffer:    make([]byte, 0, 4096),
		state:         PROCESSING,
	}

	for {
		conn, err := (*l).Accept()
		if err != nil {
			fmt.Println("Failed to read data to bodybuf")
			return err
		}
		go MessageHandler(&conn, bodyBuf, &mr, node)
	}
}

func MessageHandler(conn *net.Conn, bodyBuf []byte, mr *MessageReader, node *protocol.Node) {
	fmt.Printf("Received TCP Connection: %+v\n", conn)
	for {
		n, err := (*conn).Read(bodyBuf)
		if err != nil {
			fmt.Println("Failed to read data to bodybuf")
			panic(err)
		}
		header, pl, err := mr.ExtractMessage(bodyBuf[:n])
		if err != nil {
			fmt.Println("Failed to extract message")
			fmt.Printf("ERROR: %s\n", err)
			*mr = MessageReader{}
			continue
		}
		if mr.state == DONE {
			fmt.Println("Process rpc message and payload")
			fmt.Printf("Payload: %s\n", pl)
			go func() {
				err := node.RecvRPCMessage(header, pl)
				if err != nil {
					fmt.Println(err)
				}
			}()
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
