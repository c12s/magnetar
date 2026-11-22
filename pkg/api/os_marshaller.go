package api

import "github.com/golang/protobuf/proto"

func (x *DesiredOsImageGetReq) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}

func (x *DesiredOsImageGetReq) Unmarshal(marshalled []byte) error {
	return proto.Unmarshal(marshalled, x)
}

func (x *DesiredOsImageGetResp) Marshal() ([]byte, error) {
	return proto.Marshal(x)
}

func (x *DesiredOsImageGetResp) Unmarshal(marshalled []byte) error {
	return proto.Unmarshal(marshalled, x)
}
