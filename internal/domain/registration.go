package domain

type RegistrationReq struct {
	// NodeId, when set, is the id the agent already had: Magnetar updates or
	// recreates that record instead of minting a new UUID. Empty = new node.
	NodeId       string
	Labels       []Label
	Resources    map[string]float64
	BindAddress  string
	Capabilities []string
	OsStatus     *OsStatus
}

type RegistrationResp struct {
	NodeId string
}

type NodeUpdateReq struct {
	NodeId       NodeId
	Capabilities []string
	OsStatus     *OsStatus
}

type NodeUpdateResp struct {
	NotFound bool
}

type NodeHeartbeatReq struct {
	NodeId NodeId
}

type NodeHeartbeatResp struct {
	Found bool
}
