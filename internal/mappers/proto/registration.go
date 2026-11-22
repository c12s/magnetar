package proto

import (
	"github.com/c12s/magnetar/internal/domain"
	"github.com/c12s/magnetar/pkg/api"
)

func RegistrationReqToDomain(req *api.RegistrationReq) (*domain.RegistrationReq, error) {
	var labels []domain.Label
	for _, protoLabel := range req.Labels {
		label, err := LabelToDomain(protoLabel)
		if err != nil {
			return nil, err
		}
		labels = append(labels, label)
	}
	return &domain.RegistrationReq{
		NodeId:       req.NodeId,
		Labels:       labels,
		Resources:    req.Resources,
		BindAddress:  req.BindAddress,
		Capabilities: req.Capabilities,
		OsStatus:     OsStatusToDomain(req.OsStatus),
	}, nil
}

func RegistrationRespFromDomain(resp domain.RegistrationResp) (*api.RegistrationResp, error) {
	return &api.RegistrationResp{
		NodeId: resp.NodeId,
	}, nil
}

func NodeUpdateReqToDomain(req *api.NodeUpdateReq) (*domain.NodeUpdateReq, error) {
	return &domain.NodeUpdateReq{
		NodeId: domain.NodeId{
			Value: req.NodeId,
		},
		Capabilities: req.Capabilities,
		OsStatus:     OsStatusToDomain(req.OsStatus),
	}, nil
}

func NodeUpdateRespFromDomain(resp domain.NodeUpdateResp) (*api.NodeUpdateResp, error) {
	return &api.NodeUpdateResp{Found: !resp.NotFound, NotFound: resp.NotFound}, nil
}

func NodeHeartbeatReqToDomain(req *api.NodeHeartbeatReq) *domain.NodeHeartbeatReq {
	return &domain.NodeHeartbeatReq{
		NodeId: domain.NodeId{Value: req.NodeId},
	}
}

func NodeHeartbeatRespFromDomain(resp domain.NodeHeartbeatResp) *api.NodeHeartbeatResp {
	return &api.NodeHeartbeatResp{Found: resp.Found}
}
