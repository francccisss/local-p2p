package protocol

type PeerStatus int

const (
	LEECHING PeerStatus = iota
	SEEDING
	IDLE
)

type NodeID string

type NodeAddr struct {
	IP   []byte
	Port int
}

type Node struct {
	NeighboringNodes []NodeAddr // used for bootstrapping
	NodeID           NodeID
	Addr             NodeAddr
	FILE_LOCATION    string
	ClusterTable     *ClusterTable
}

func NewNode(addr NodeAddr, nodeID NodeID, fileLoc string) *Node {
	newClusterTable := make(ClusterTable)
	return &Node{
		Addr:             addr,
		NodeID:           nodeID,
		FILE_LOCATION:    fileLoc,
		NeighboringNodes: make([]NodeAddr, 10),
		ClusterTable:     &newClusterTable,
	}

}
