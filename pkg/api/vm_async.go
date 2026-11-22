package api

import (
	"context"

	natsgo "github.com/nats-io/nats.go"
)

// VmAsyncClient is the host-side (Star) client for VM management: it reads
// the desired VM set from the node record and reports observed VM status
// back. Both are request/reply over the subjects in subjects.go.
type VmAsyncClient struct {
	conn *natsgo.Conn
}

func NewVmAsyncClient(natsAddress string) (*VmAsyncClient, error) {
	conn, err := connect(natsAddress)
	if err != nil {
		return nil, err
	}
	return &VmAsyncClient{conn: conn}, nil
}

// GetDesiredVms returns the desired VM set stored on the host node record.
func (c *VmAsyncClient) GetDesiredVms(ctx context.Context, hostId string) (*VmDesiredGetResp, error) {
	resp := &VmDesiredGetResp{}
	if err := request(ctx, c.conn, VmDesiredGetSubject, &VmDesiredGetReq{HostId: hostId}, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// SetVmStatuses reports the observed VM states of a host.
func (c *VmAsyncClient) SetVmStatuses(ctx context.Context, hostId string, statuses []*VmStatus) (*VmStatusSetResp, error) {
	resp := &VmStatusSetResp{}
	if err := request(ctx, c.conn, VmStatusSetSubject, &VmStatusSetReq{HostId: hostId, Statuses: statuses}, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
