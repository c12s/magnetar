package api

import (
	"context"

	natsgo "github.com/nats-io/nats.go"
)

// OsAsyncClient is the node-side (Star) client for the OS image the operator
// declared for this node.
type OsAsyncClient struct {
	conn *natsgo.Conn
}

func NewOsAsyncClient(natsAddress string) (*OsAsyncClient, error) {
	conn, err := connect(natsAddress)
	if err != nil {
		return nil, err
	}
	return &OsAsyncClient{conn: conn}, nil
}

// GetDesiredOsImage returns the declared image; resp.Image is nil when none
// is declared, resp.NotFound when Magnetar does not know the node.
func (c *OsAsyncClient) GetDesiredOsImage(ctx context.Context, nodeId string) (*DesiredOsImageGetResp, error) {
	resp := &DesiredOsImageGetResp{}
	if err := request(ctx, c.conn, OsDesiredGetSubject, &DesiredOsImageGetReq{NodeId: nodeId}, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
