package domain

import "errors"

// ErrNodeNotFound means the node has no record; any other repo error is a
// storage failure and must not be treated as "unknown node".
var ErrNodeNotFound = errors.New("node not found")

type Node struct {
	Id             NodeId
	Org            string
	Labels         []Label
	Resources      map[string]float64
	BindAddress    string
	Capabilities   []string
	OsStatus       *OsStatus
	DesiredOsImage *DesiredOsImage
	DesiredVms     []DesiredVm
	VmStatus       []VmStatus
	// LastSeen (unix seconds) and Liveness are a read-time view joined from
	// the last_seen key; they are never persisted on the node record.
	LastSeen int64
	Liveness string
}

func (n Node) Claimed() bool {
	return len(n.Org) > 0
}

type NodeId struct {
	Value string
}

type Query []Selector

type Selector struct {
	LabelKey string
	ShouldBe ComparisonResult
	Value    string
}

type NodeRepo interface {
	Put(node Node) error
	Get(nodeId NodeId, org string) (*Node, error)
	FindById(nodeId NodeId) (*Node, error)
	Delete(node Node) error
	// DeleteNode removes a node completely: record, label index, liveness
	// state and desired OS intent. Delete (above) only moves the record
	// between pool and org namespaces (claim), so it keeps liveness and intent.
	DeleteNode(node Node) error
	ListNodePool() ([]Node, error)
	ListOrgOwnedNodes(org string) ([]Node, error)
	QueryNodePool(query Query) ([]Node, error)
	QueryOrgOwnedNodes(query Query, org string) ([]Node, error)
	PutLabel(node Node, label Label) (*Node, error)
	DeleteLabel(node Node, labelKey string) (*Node, error)
	ListAllNodes() ([]Node, error)
	PutDesiredOsImage(nodeId NodeId, image *DesiredOsImage) error
	GetDesiredOsImage(nodeId NodeId) (*DesiredOsImage, error)
	// TouchSeen records liveness (unix seconds); ListSeen returns every known
	// node's last-seen time, for the liveness view and the GC.
	TouchSeen(nodeId NodeId, at int64) error
	DeleteSeen(nodeId NodeId) error
	ListSeen() (map[string]int64, error)
}

type NodeMarshaller interface {
	Marshal(node Node) ([]byte, error)
	Unmarshal(nodeMarshalled []byte) (*Node, error)
}

type GetFromNodePoolReq struct {
	Id NodeId
}

type GetFromNodePoolResp struct {
	Node Node
}

type GetFromOrgReq struct {
	Id  NodeId
	Org string
}

type GetFromOrgResp struct {
	Node Node
}

type ClaimOwnershipReq struct {
	Query Query
	Org   string
}

type ClaimOwnershipResp struct {
	Nodes []Node
}

type ListNodePoolReq struct {
}

type ListNodePoolResp struct {
	Nodes []Node
}

type ListOrgOwnedNodesReq struct {
	Org string
}

type ListOrgOwnedNodesResp struct {
	Nodes []Node
}

type QueryNodePoolReq struct {
	Query Query
}

type QueryNodePoolResp struct {
	Nodes []Node
}

type QueryOrgOwnedNodesReq struct {
	Query Query
	Org   string
}

type QueryOrgOwnedNodesResp struct {
	Nodes []Node
}
