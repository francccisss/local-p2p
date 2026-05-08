package protocol

type Method uint32
type MethodString string

const (
	PING_NODE Method = iota
	PING_CLUSTER
	LEECH
	PROBE
	JOIN
	FIND_CLUSTER
)

var MethodStringMap = map[Method]MethodString{
	0: "PING NODE",
	1: "PING CLUSTER",
	2: "LEECH",
	3: "PROBE",
	4: "JOIN",
	5: "FIND CLUSTER",
}

const PREFIX_HEADER_SIZE = 4

type ClientConn interface {
	FindCluster(cname ClusterName)
	Join()
	Ping(cname ClusterName) error
	Leech(cname ClusterName, spawnThreads bool, fr FileRequest) error
	ProbeFile(fileKey string) (StatusCode, error)
	RecvRPCMessage(msg RPCMsg) error
}

type MsgType uint32

const (
	CALL MsgType = iota
	REPLY
)

type StatusCode int

const (
	SUCCESS StatusCode = iota
	ERROR
)

// MsgType could be either reply or call
type RPCMsg struct {
	RPCType     MsgType
	IP          []byte
	Port        []byte
	NodeID      NodeID
	Method      Method
	PayloadSize uint32
	StatusCode  StatusCode
	Message     string
}

type ClusterRequest string

type ClusterResponse struct {
	ClusterName
	Peers []ClusterPeer
}

type NodeStatus string
type NodeStatusEnum int

const (
	ACTIVE NodeStatusEnum = iota
)

var NodeStatusMap = map[NodeStatusEnum]NodeStatus{
	0: "ACTIVE",
}

type PingResponse NodeStatus
type PingRequest string
