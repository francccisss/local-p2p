package protocol

type FileHash = string

type Method uint32
type MethodString string

const (
	PING_NODE Method = iota
	PING_CLUSTER
	FIND_CLUSTER
	JOIN
	PROBE
	LEECH
)

var MethodStringMap = map[Method]MethodString{
	0: "PING NODE",
	1: "PING CLUSTER",
	2: "FIND CLUSTER",
	3: "JOIN",
	4: "PROBE",
	5: "LEECH",
}

const PREFIX_HEADER_SIZE = 4

type MsgType uint32

const (
	CALL MsgType = iota
	REPLY
)

type NodeStatus string
type NodeStatusEnum int

const (
	ACTIVE NodeStatusEnum = iota
)

var NodeStatusMap = map[NodeStatusEnum]NodeStatus{
	0: "ACTIVE",
}

type StatusCode int

const (
	SUCCESS StatusCode = iota
	ERROR
)

// MsgType could be either reply or call
type RPCMsgHeader struct {
	RPCType     MsgType
	IP          []byte
	Port        []byte
	NodeID      NodeID
	Method      Method
	PayloadSize uint32
	StatusCode  StatusCode
	Message     string
}

type Payload []byte

type ClusterRequest string

type ClusterResponse struct {
	ClusterName
	FileHash FileHash
	Peers    []ClusterPeer
}

type PingRequest string
type PingResponse = NodeStatus

type ProbeRequest struct {
	ClusterName ClusterName
	FileHash    FileHash
}
type ProbeReponse = FileMetaData

// TODO: do i need to handle any security checks here?
type JoinRequest = ClusterName

type JoinResponse struct {
	Status      PeerStatus
	NodeID      NodeID
	ClusterName ClusterName
}

// general error format - GEF
// "[CALL]: PING - pinged by neighboring node %s\n", pr

type RPCErrorStr struct {
	ErrorMessage string
}

func (re RPCErrorStr) Error() string {
	return re.ErrorMessage
}
