package api

import (
	"context"

	"github.com/golang/protobuf/proto"
	natsgo "github.com/nats-io/nats.go"
)

type RegistrationAsyncClient struct {
	conn *natsgo.Conn
}

func NewRegistrationAsyncClient(natsAddress string) (*RegistrationAsyncClient, error) {
	conn, err := connect(natsAddress)
	if err != nil {
		return nil, err
	}
	return &RegistrationAsyncClient{conn: conn}, nil
}

// Register enrolls the node and returns the id Magnetar assigned.
func (n *RegistrationAsyncClient) Register(ctx context.Context, req *RegistrationReq) (*RegistrationResp, error) {
	resp := &RegistrationResp{}
	if err := request(ctx, n.conn, RegistrationSubject, req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

type RegistrationReqBuilder struct {
	req *RegistrationReq
}

func NewRegistrationReqBuilder() RegistrationReqBuilder {
	return RegistrationReqBuilder{
		req: &RegistrationReq{
			Labels:    make([]*Label, 0),
			Resources: map[string]float64{},
		},
	}
}

func (r RegistrationReqBuilder) AddBoolLabel(key string, value bool) RegistrationReqBuilder {
	valueMarshalled, err := proto.Marshal(&BoolValue{Value: value})
	if err != nil {
		return r
	}
	return r.addLabel(key, Value_Bool, valueMarshalled)
}

func (r RegistrationReqBuilder) AddFloat64Label(key string, value float64) RegistrationReqBuilder {
	valueMarshalled, err := proto.Marshal(&Float64Value{Value: value})
	if err != nil {
		return r
	}
	return r.addLabel(key, Value_Float64, valueMarshalled)
}

func (r RegistrationReqBuilder) AddStringLabel(key string, value string) RegistrationReqBuilder {
	valueMarshalled, err := proto.Marshal(&StringValue{Value: value})
	if err != nil {
		return r
	}
	return r.addLabel(key, Value_String, valueMarshalled)
}

func (r RegistrationReqBuilder) AddCapabilities(capabilities ...string) RegistrationReqBuilder {
	r.req.Capabilities = append(r.req.Capabilities, capabilities...)
	return r
}

func (r RegistrationReqBuilder) SetOsStatus(osStatus *OsStatus) RegistrationReqBuilder {
	r.req.OsStatus = osStatus
	return r
}

// SetNodeId re-asserts the node's persisted id on re-registration, so
// Magnetar updates (or recreates) the same record instead of minting a new
// UUID. Leave unset for a brand-new node.
func (r RegistrationReqBuilder) SetNodeId(nodeId string) RegistrationReqBuilder {
	r.req.NodeId = nodeId
	return r
}

func (r RegistrationReqBuilder) addLabel(key string, valueType Value_ValueTYpe, valueMarshalled []byte) RegistrationReqBuilder {
	label := &Label{
		Key: key,
		Value: &Value{
			Type:       valueType,
			Marshalled: valueMarshalled,
		},
	}
	r.req.Labels = append(r.req.Labels, label)
	return r
}

func (r RegistrationReqBuilder) Request() *RegistrationReq {
	return r.req
}

func (r RegistrationReqBuilder) Clear() RegistrationReqBuilder {
	return NewRegistrationReqBuilder()
}
