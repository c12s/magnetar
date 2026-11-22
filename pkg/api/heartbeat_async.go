package api

import (
	"context"

	natsgo "github.com/nats-io/nats.go"
)

// NodeHeartbeatAsyncClient is the node-side client for liveness reporting.
// Every node (regardless of capabilities) sends a heartbeat on its own
// cadence; the reply says whether Magnetar still has a record for the node.
type NodeHeartbeatAsyncClient struct {
	conn *natsgo.Conn
}

func NewNodeHeartbeatAsyncClient(natsAddress string) (*NodeHeartbeatAsyncClient, error) {
	conn, err := connect(natsAddress)
	if err != nil {
		return nil, err
	}
	return &NodeHeartbeatAsyncClient{conn: conn}, nil
}

// Heartbeat reports liveness and returns false when Magnetar does not know
// the node anymore (the caller must re-register, asserting its id).
func (c *NodeHeartbeatAsyncClient) Heartbeat(ctx context.Context, nodeId string) (bool, error) {
	resp := &NodeHeartbeatResp{}
	if err := request(ctx, c.conn, NodeHeartbeatSubject, &NodeHeartbeatReq{NodeId: nodeId}, resp); err != nil {
		return false, err
	}
	return resp.Found, nil
}
