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

func HandleConn(l *net.Listener, node *protocol.Node, clt *protocol.ClusterTable) error {

	for {
		// creates a new file descriptor for incoming tcp connection from node/peer
		conn, err := (*l).Accept()
		if err != nil {
			fmt.Printf("Connection from %s failed\n", conn.LocalAddr().String())
			return err
		}
		go MessageHandler(&conn, node, clt)
	}
}

func MessageHandler(conn *net.Conn, node *protocol.Node, clt *protocol.ClusterTable) {

	bodyBuf := make([]byte, 4096)
	var mr MessageReader = MessageReader{
		payloadBuffer: make([]byte, 0, 4096),
		jsonBuffer:    make([]byte, 0, 4096),
		state:         PROCESSING,
	}
	fmt.Printf("Received TCP Connection: %+v\n", conn)
	for {
		// Reads from the file descriptor set by the kernel for the node that was accepted
		n, err := (*conn).Read(bodyBuf)
		if n == 0 {
			return
		}
		defer (*conn).Close()
		if err != nil {
			return
		}
		header, pl, err := mr.extractMessage(bodyBuf[:n])
		if err != nil {
			fmt.Println("Failed to extract message")
			fmt.Printf("ERROR: %s\n", err)
			mr = MessageReader{}
			continue
		}
		if mr.state == DONE {
			fmt.Println("Process rpc message and payload")
			fmt.Printf("Payload: %s\n", pl)
			go func() {
				err := node.RecvRPCMessage(header, pl, conn, clt)
				if err != nil {
					fmt.Println(err)
					return
				}
			}()
			mr = MessageReader{}
			continue
		}
	}

}

func (mr *MessageReader) extractMessage(buffer []byte) (protocol.RPCMsg, Payload, error) {

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
			mr.phase = PAYLOAD
			mr.metaJson.RPCType = protocol.MsgType(ntoh(uint32(mr.metaJson.RPCType)))
			mr.metaJson.PayloadSize = ntoh(mr.metaJson.PayloadSize)
			mr.metaJson.Method = protocol.Method(ntoh(uint32(mr.metaJson.Method)))

			mr.payloadSize = int(mr.metaJson.PayloadSize)
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

func ntoh(num uint32) uint32 {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, num)
	return binary.LittleEndian.Uint32(buf)
}
