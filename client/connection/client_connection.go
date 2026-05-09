package connection

import (
	"client/protocol"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
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
	PayloadBuffer []byte
	JsonBuffer    []byte
	MetaJson      protocol.RPCMsg
	PayloadSize   int
	MetaJsonSize  uint32
	Phase         PHASE
	State         MessageReaderState
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
		go MessageHandler(conn, node, clt)
	}
}

// handle's its own socket after establshing connection
// so keeping conn in a loop of Read makes it so we don't have to keep accepting requests
func MessageHandler(conn io.ReadWriteCloser, node *protocol.Node, clt *protocol.ClusterTable) {

	bodyBuf := make([]byte, 4096)
	var mr MessageReader = MessageReader{
		PayloadBuffer: make([]byte, 0, 4096),
		JsonBuffer:    make([]byte, 0, 4096),
		State:         PROCESSING,
	}
	fmt.Print("Received TCP Connection\n")
	for {
		// Reads from the file descriptor set by the kernel for the node that was accepted
		n, err := conn.Read(bodyBuf)
		if n == 0 {
			return
		}
		defer conn.Close()
		if err != nil {
			return
		}
		header, pl, err := mr.ExtractMessage(bodyBuf[:n])
		if err != nil {
			fmt.Println("Failed to extract message")
			fmt.Printf("ERROR: %s\n", err)
			mr = MessageReader{}
			continue
		}
		if mr.State == DONE {
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

func (mr *MessageReader) ExtractMessage(buffer []byte) (protocol.RPCMsg, Payload, error) {

	for {
		switch mr.Phase {
		case PREFIX:
			fmt.Println("Prefix Phase")
			mr.MetaJsonSize = binary.LittleEndian.Uint32(buffer[:protocol.PREFIX_HEADER_SIZE])
			fmt.Printf("[PREFIX PHASE]: Meta Json length: %d\n", mr.MetaJsonSize)
			mr.Phase = META_JSON

			buffer = buffer[protocol.PREFIX_HEADER_SIZE:]
			fmt.Printf("left in buffer: %d\n", len(buffer))
		case META_JSON:
			fmt.Printf("Meta Json Phase: %d\n", mr.MetaJsonSize)
			remaining := int(mr.MetaJsonSize) - len(mr.JsonBuffer) // making sure we dont overextend to the payload sectionA
			if len(buffer) < remaining {
				mr.JsonBuffer = append(mr.JsonBuffer, buffer...)
				if len(mr.JsonBuffer) < int(mr.MetaJsonSize) {
					fmt.Printf("Meta json: rem-%d, Delivered/read%d\n", remaining, len(buffer))
					return mr.MetaJson, nil, nil
				}
				fmt.Printf("Meta json: rem-%d, Delivered/read%d\n", remaining, len(buffer))
				buffer = buffer[len(buffer):] // push index to end
			} else {
				// NOTE: This leaves extra bytes for the next phase
				mr.JsonBuffer = append(mr.JsonBuffer, buffer[:remaining]...) // ga subra ko index tungod sa buffer access

				fmt.Printf("Meta json: rem-%d, Delivered/read%d\n", remaining, len(buffer))
				fmt.Printf("Rem in Buffer After :%d\n", len(buffer[remaining:]))
				buffer = buffer[remaining:]
			}
			fmt.Printf("Read Buffer Len: %d\n", len(mr.JsonBuffer))
			fmt.Printf("[META PHASE]: Meta json content buffer size: %d\n", mr.MetaJsonSize)
			err := json.Unmarshal(mr.JsonBuffer, &mr.MetaJson)
			if err != nil {
				fmt.Println("[META PHASE]: Unable to unmarshal meta data json")
				return mr.MetaJson, nil, err
			}
			fmt.Printf("Meta data received %+v\n", mr.MetaJson)
			mr.Phase = PAYLOAD
			mr.MetaJson.RPCType = protocol.MsgType(ntoh(uint32(mr.MetaJson.RPCType)))
			mr.MetaJson.PayloadSize = ntoh(mr.MetaJson.PayloadSize)
			mr.MetaJson.Method = protocol.Method(ntoh(uint32(mr.MetaJson.Method)))

			mr.PayloadSize = int(mr.MetaJson.PayloadSize)
			fmt.Printf("Remaining bytes in Buffer: %d\n", len(buffer))

			fmt.Printf("Bytes left to read: %d\n", (int(mr.MetaJsonSize)+protocol.PREFIX_HEADER_SIZE+mr.PayloadSize)-(int(mr.MetaJsonSize)+protocol.PREFIX_HEADER_SIZE))
		case PAYLOAD:

			remaining := mr.PayloadSize - len(mr.PayloadBuffer)
			if len(buffer) < remaining {
				fmt.Printf("Remaining bytes from payload: %d, received: %d\n", remaining, len(buffer))
				mr.PayloadBuffer = append(mr.PayloadBuffer, buffer...)
				// guard
				fmt.Printf("Remaining: %d, IN: %d\n", remaining, len(mr.PayloadBuffer))
				if len(mr.PayloadBuffer) < mr.PayloadSize {
					return mr.MetaJson, nil, nil
				}
			} else {
				mr.PayloadBuffer = append(mr.PayloadBuffer, buffer[:remaining]...)
			}

			mr.Phase = PREFIX
			mr.State = DONE
			mr.PayloadSize = 0
			return mr.MetaJson, mr.PayloadBuffer, nil
		}

	}
}

func ntoh(num uint32) uint32 {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, num)
	return binary.LittleEndian.Uint32(buf)
}
