package protocol

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// buffer is the payload received from a peer
func ReadRPCMessage(buffer []byte) (RPCMsgHeader, error) {

	var msg RPCMsgHeader
	err := json.Unmarshal(buffer, &msg)
	if err != nil {
		fmt.Println("Unable to Unmarshal")
		return RPCMsgHeader{}, err
	}
	return msg, nil
}

func (n *Node) RecvRPCMessage(msg RPCMsgHeader, payload Payload, conn io.ReadWriteCloser, clt *ClusterTable) error {
	if msg.StatusCode == ERROR {
		return ProtocolErrorWrap(msg.Message, msg.RPCType, msg.Method)
	}

	switch msg.RPCType {

	case CALL: // when peers/nodes send a call RPCType

		var newRPCMsgHeader RPCMsgHeader
		fmt.Println("[ RPC MESSAGE ]: Call Message")

		newRPCMsgHeader = RPCMsgHeader{
			Method:      msg.Method,
			RPCType:     REPLY,
			NodeID:      n.NodeID,
			IP:          n.Addr.IP,
			Message:     "",
			StatusCode:  SUCCESS,
			PayloadSize: 0,
			Port:        make([]byte, 4),
		}
		// setting requesting node's port to host byte ordering
		binary.LittleEndian.PutUint32(newRPCMsgHeader.Port, uint32(n.Addr.Port))
		switch msg.Method {
		case HAVE:

		case PING_NODE:
			requstNodePort := binary.LittleEndian.Uint32(msg.Port)
			requestNodeAddr := NodeAddr{IP: msg.IP, Port: int(requstNodePort)}
			b, err := HandlePingNodeRequest(newRPCMsgHeader, msg, payload)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), CALL, PING_NODE)
			}
			err = SendMsg(b, requestNodeAddr)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), CALL, PING_NODE)
			}

		case PING_CLUSTER:
			b, err := HandlePingClusterRequest(newRPCMsgHeader, msg, payload)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), CALL, PING_CLUSTER)
			}
			err = SendViaExistingConn(b, conn)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), CALL, PING_CLUSTER)
			}

		case FIND_CLUSTER:
			b, err := HandleFindClusterRequest(newRPCMsgHeader, msg, payload, clt)
			// Sends an error that a cluster does not exist
			if err != nil {
				newRPCMsgHeader.StatusCode = ERROR
				newRPCMsgHeader.Message = err.Error()
				b, err := WrapPayloadToBuffer(newRPCMsgHeader, nil)
				if err != nil {
					return ProtocolErrorWrap(err.Error(), CALL, FIND_CLUSTER)
				}
				err = SendViaExistingConn(b, conn)
				if err != nil {
					return ProtocolErrorWrap(err.Error(), CALL, FIND_CLUSTER)
				}
				return nil
			}

			err = SendViaExistingConn(b, conn)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), CALL, FIND_CLUSTER)
			}
		case JOIN:
			b, err := HandleJoinClusterRequest(newRPCMsgHeader, msg, payload, conn, clt)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), CALL, JOIN)
			}
			err = SendViaExistingConn(b, conn)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), CALL, JOIN)
			}
			return nil

		case PROBE:
			b, err := HandleProbeRequest(newRPCMsgHeader, msg, payload, n.FILE_LOCATION)
			if err != nil {
				fmt.Printf("[ PROBE REQUEST ]: %s\n", err)
				newRPCMsgHeader.Message = err.Error()
				newRPCMsgHeader.StatusCode = ERROR
				b, err := WrapPayloadToBuffer(newRPCMsgHeader, nil)
				if err != nil {
					return fmt.Errorf("Unable to send message")
				}
				err = SendViaExistingConn(b, conn)
				if err != nil {
					return fmt.Errorf("Unable to send message")
				}
			}

			buf, err := WrapPayloadToBuffer(newRPCMsgHeader, b)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), CALL, PROBE)
			}
			err = SendViaExistingConn(buf, conn)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), CALL, PROBE)
			}

		case LEECH:
			fmt.Printf("[TESTING]: '%s' received a LEECH REQUEST\n", n.NodeID)
			return HandleLeechRequest(newRPCMsgHeader, msg, payload, conn, n.FILE_LOCATION, clt)
		}

	case REPLY: // when peers/nodes send a reaply RPCType

		if msg.StatusCode == ERROR {
			return ProtocolErrorWrap(msg.Message, REPLY, msg.Method)
		}

		switch msg.Method {

		case HAVE:

		case PING_NODE:
			pingResponse := HandlePingNodeResponse(msg, payload)
			fmt.Printf("[ PING ]: Retrived Ping reponse %+v", pingResponse)
			conn.Close()
		case PING_CLUSTER:
			pingResponse := HandlePingClusterResponse(msg, payload)
			fmt.Printf("[ PING ]: Retrived Ping reponse %+v", pingResponse)
		case FIND_CLUSTER:
			cr, err := HandleFindClusterResponse(msg, payload, clt)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), REPLY, FIND_CLUSTER)
			}
			fmt.Printf("[ REPLY ]: FIND CLUSTER - %d new peers added to %s cluster\n", len(cr.Peers), cr.ClusterHash)
		case JOIN:
			fmt.Println("HANDLING")
			err := HandleJoinClusterResponse(msg, payload, conn, clt)
			if err != nil {
				return ProtocolErrorWrap(err.Error(), CALL, JOIN)
			}

		case LEECH:
			HandleLeechResponse(msg, payload, conn, clt)
			// if err != nil {
			// 	return err
			// }
			return nil

		case PROBE:
			return HandleProbeResponse(msg, payload)

		}
	}
	return nil
}
