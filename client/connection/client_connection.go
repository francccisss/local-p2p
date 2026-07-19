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
	JSONBuffer    []byte
	MetaJSON      protocol.RPCMsgHeader
	PayloadSize   int
	MetaJSONSize  uint32
	Phase         PHASE
	State         MessageReaderState
}

func HandleConn(l *net.Listener, node *protocol.Node, clt *protocol.ClusterTable) error {

	for {

		// creates a new file descriptor for incoming tcp connection from node/peer
		conn, err := (*l).Accept()
		if err != nil {
			fmt.Printf("[ Client Connection ]: Connection from %s failed\n", conn.LocalAddr().String())
			return err
		}
		fmt.Print("[ Client Connection ]: Received TCP Connection\n")
		go MessageHandler(conn, node, clt)
	}
}

// Loops through its current connection to a node
func MessageHandler(conn io.ReadWriteCloser, node *protocol.Node, clt *protocol.ClusterTable) {

	bodyBuf := make([]byte, 4096)
	var mr MessageReader = MessageReader{
		PayloadBuffer: make([]byte, 0, 4096),
		JSONBuffer:    make([]byte, 0, 4096),
		State:         PROCESSING,
	}
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
			fmt.Println("[ MESSAGE HANDLER ]: Failed to extract message")
			fmt.Printf("[ MESSAGE HANDLER ]: ERROR - %s\n", err)
			mr = MessageReader{}
			continue
		}
		if mr.State == DONE {
			fmt.Printf("[ MESSAGE HANDLER ]: Payload: %s\n", pl)
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

func (mr *MessageReader) ExtractMessage(buffer []byte) (protocol.RPCMsgHeader, protocol.Payload, error) {

	fmt.Println("[ MESSAGE HANDLER ]: Extracting Message")
	for {
		switch mr.Phase {
		case PREFIX:
			mr.MetaJSONSize = binary.LittleEndian.Uint32(buffer[:protocol.PREFIX_HEADER_SIZE])
			fmt.Printf("[ PREFIX PHASE ]: Meta JSON length: %d\n", mr.MetaJSONSize)
			mr.Phase = META_JSON

			buffer = buffer[protocol.PREFIX_HEADER_SIZE:]
			fmt.Printf("[ PREFIX PHASE ]: left in buffer: %d\n", len(buffer))
		case META_JSON:
			fmt.Printf("[ METAJSON PHASE ]: %d\n", mr.MetaJSONSize)
			remaining := int(mr.MetaJSONSize) - len(mr.JSONBuffer) // making sure we dont overextend to the payload sectionA
			if len(buffer) < remaining {
				mr.JSONBuffer = append(mr.JSONBuffer, buffer...)
				if len(mr.JSONBuffer) < int(mr.MetaJSONSize) {
					fmt.Printf("[ METAJSON PHASE ]: rem-%d, Delivered/read%d\n", remaining, len(buffer))
					return mr.MetaJSON, nil, nil
				}
				fmt.Printf("[ METAJSON PHASE  ]: rem-%d, Delivered/read%d\n", remaining, len(buffer))
				buffer = buffer[len(buffer):] // push index to end
			} else {
				// NOTE: This leaves extra bytes for the next phase
				mr.JSONBuffer = append(mr.JSONBuffer, buffer[:remaining]...) // ga subra ko index tungod sa buffer access

				fmt.Printf("[ METAJSON PHASE ]: rem-%d, Delivered/read%d\n", remaining, len(buffer))
				fmt.Printf("[ METAJSON PHASE ]: Rem in Buffer After :%d\n", len(buffer[remaining:]))
				buffer = buffer[remaining:]
			}
			fmt.Printf("[ METAJSON PHASE ]: Read Buffer Len: %d\n", len(mr.JSONBuffer))
			fmt.Printf("[ METAJSON PHASE ]: Meta JSON content buffer size: %d\n", mr.MetaJSONSize)
			err := json.Unmarshal(mr.JSONBuffer, &mr.MetaJSON)
			if err != nil {
				fmt.Println("[ PREFIX PHASE ]: METAJSON PHASE - Unable to unmarshal meta data JSON")
				return mr.MetaJSON, nil, err
			}
			fmt.Printf("[ METAJSON PHASE ]: Meta data received %+v\n", mr.MetaJSON)
			mr.Phase = PAYLOAD
			mr.MetaJSON.RPCType = protocol.MsgType(ntoh(uint32(mr.MetaJSON.RPCType)))
			mr.MetaJSON.PayloadSize = ntoh(mr.MetaJSON.PayloadSize)
			mr.MetaJSON.Method = protocol.Method(ntoh(uint32(mr.MetaJSON.Method)))

			mr.PayloadSize = int(mr.MetaJSON.PayloadSize)
			fmt.Printf("[ METAJSON PHASE ]: Remaining bytes in Buffer: %d\n", len(buffer))

			fmt.Printf("[ METAJSON PHASE ]: Bytes left to read: %d\n", (int(mr.MetaJSONSize)+protocol.PREFIX_HEADER_SIZE+mr.PayloadSize)-(int(mr.MetaJSONSize)+protocol.PREFIX_HEADER_SIZE))
		case PAYLOAD:

			remaining := mr.PayloadSize - len(mr.PayloadBuffer)
			if len(buffer) < remaining {
				fmt.Printf("[ PAYLOAD PHASE ]: Remaining bytes from payload: %d, received: %d\n", remaining, len(buffer))
				mr.PayloadBuffer = append(mr.PayloadBuffer, buffer...)
				// guard
				fmt.Printf("[ PAYLOAD PHASE ]: Remaining: %d, IN: %d\n", remaining, len(mr.PayloadBuffer))
				if len(mr.PayloadBuffer) < mr.PayloadSize {
					return mr.MetaJSON, nil, nil
				}
			} else {
				mr.PayloadBuffer = append(mr.PayloadBuffer, buffer[:remaining]...)
			}

			mr.Phase = PREFIX
			mr.State = DONE
			mr.PayloadSize = 0
			return mr.MetaJSON, mr.PayloadBuffer, nil
		}

	}
}

func ntoh(num uint32) uint32 {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, num)
	return binary.LittleEndian.Uint32(buf)
}
