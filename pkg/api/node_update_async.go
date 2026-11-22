package api

import (
	"context"

	natsgo "github.com/nats-io/nats.go"
)

type NodeUpdateAsyncClient struct {
	conn *natsgo.Conn
}

func NewNodeUpdateAsyncClient(natsAddress string) (*NodeUpdateAsyncClient, error) {
	conn, err := connect(natsAddress)
	if err != nil {
		return nil, err
	}
	return &NodeUpdateAsyncClient{conn: conn}, nil
}

// Update sends a partial node update and waits for Magnetar's reply.
func (n *NodeUpdateAsyncClient) Update(ctx context.Context, req *NodeUpdateReq) (*NodeUpdateResp, error) {
	resp := &NodeUpdateResp{}
	if err := request(ctx, n.conn, NodeUpdateSubject, req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
