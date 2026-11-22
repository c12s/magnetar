package servers

import (
	"context"
	"errors"
	"strings"

	"github.com/c12s/magnetar/internal/domain"
	"github.com/c12s/magnetar/internal/mappers/proto"
	"github.com/c12s/magnetar/internal/services"
	"github.com/c12s/magnetar/pkg/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type MagnetarGrpcServer struct {
	api.UnimplementedMagnetarServer
	nodeService  services.NodeService
	labelService services.LabelService
	vmService    services.VmService
	osService    services.OsService
	authorizer   services.AuthZService
}

func NewMagnetarGrpcServer(nodeService services.NodeService, labelService services.LabelService, vmService services.VmService, osService services.OsService, authorizer services.AuthZService) (api.MagnetarServer, error) {
	return &MagnetarGrpcServer{
		nodeService:  nodeService,
		labelService: labelService,
		vmService:    vmService,
		osService:    osService,
		authorizer:   authorizer,
	}, nil
}

// authorizeHost guards the user-facing VM endpoints; hosts reach the same
// data over NATS (VmAsyncServer) without a user token.
func (m *MagnetarGrpcServer) authorizeHost(ctx context.Context, permission, hostId string) error {
	if !m.authorizer.Authorize(ctx, permission, "node", hostId) {
		return status.Error(codes.PermissionDenied, domain.ErrForbidden.Error())
	}
	return nil
}

func (m *MagnetarGrpcServer) GetFromNodePool(ctx context.Context, req *api.GetFromNodePoolReq) (*api.GetFromNodePoolResp, error) {
	domainReq, err := proto.GetFromNodePoolReqToDomain(req)
	if err != nil {
		return nil, err
	}
	domainResp, err := m.nodeService.GetFromNodePool(ctx, *domainReq)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		return nil, err
	}
	return proto.GetFromNodePoolRespFromDomain(*domainResp)
}

func (m *MagnetarGrpcServer) GetFromOrg(ctx context.Context, req *api.GetFromOrgReq) (*api.GetFromOrgResp, error) {
	domainReq, err := proto.GetFromOrgReqToDomain(req)
	if err != nil {
		return nil, err
	}
	domainResp, err := m.nodeService.GetFromOrg(ctx, *domainReq)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		return nil, err
	}
	return proto.GetFromOrgRespFromDomain(*domainResp)
}

func (m *MagnetarGrpcServer) ClaimOwnership(ctx context.Context, req *api.ClaimOwnershipReq) (*api.ClaimOwnershipResp, error) {
	domainReq, err := proto.ClaimOwnershipReqToDomain(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	domainResp, err := m.nodeService.ClaimOwnership(ctx, *domainReq)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		} else if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, err
	}
	return proto.ClaimOwnershipRespFromDomain(*domainResp)
}

func (m *MagnetarGrpcServer) ListNodePool(ctx context.Context, req *api.ListNodePoolReq) (*api.ListNodePoolResp, error) {
	domainReq, err := proto.ListNodePoolReqToDomain(req)
	if err != nil {
		return nil, err
	}
	domainResp, err := m.nodeService.ListNodePool(ctx, *domainReq)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		return nil, err
	}
	return proto.ListNodePoolRespFromDomain(*domainResp)
}

func (m *MagnetarGrpcServer) ListOrgOwnedNodes(ctx context.Context, req *api.ListOrgOwnedNodesReq) (*api.ListOrgOwnedNodesResp, error) {
	domainReq, err := proto.ListOrgOwnedReqToDomain(req)
	if err != nil {
		return nil, err
	}
	domainResp, err := m.nodeService.ListOrgOwnedNodes(ctx, *domainReq)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		return nil, err
	}
	return proto.ListOrgOwnedNodesRespFromDomain(*domainResp)
}

func (m *MagnetarGrpcServer) QueryNodePool(ctx context.Context, req *api.QueryNodePoolReq) (*api.QueryNodePoolResp, error) {
	domainReq, err := proto.QueryNodePoolReqToDomain(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	domainResp, err := m.nodeService.QueryNodePool(ctx, *domainReq)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		} else if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, err
	}
	return proto.QueryNodePoolRespFromDomain(*domainResp)
}

func (m *MagnetarGrpcServer) QueryOrgOwnedNodes(ctx context.Context, req *api.QueryOrgOwnedNodesReq) (*api.QueryOrgOwnedNodesResp, error) {
	domainReq, err := proto.QueryOrgOwnedNodesReqToDomain(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	domainResp, err := m.nodeService.QueryOrgOwnedNodes(ctx, *domainReq)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		} else if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, err
	}
	return proto.QueryOrgOwnedNodesRespFromDomain(*domainResp)
}

func (m *MagnetarGrpcServer) PutBoolLabel(ctx context.Context, req *api.PutBoolLabelReq) (*api.PutLabelResp, error) {
	domainReq, err := proto.PutBoolLabelReqToDomain(req)
	if err != nil {
		return nil, err
	}
	domainResp, err := m.labelService.PutLabel(ctx, *domainReq)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		return nil, err
	}
	return proto.PutLabelRespFromDomain(*domainResp)
}

func (m *MagnetarGrpcServer) PutFloat64Label(ctx context.Context, req *api.PutFloat64LabelReq) (*api.PutLabelResp, error) {
	domainReq, err := proto.PutFloat64LabelReqToDomain(req)
	if err != nil {
		return nil, err
	}
	domainResp, err := m.labelService.PutLabel(ctx, *domainReq)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return proto.PutLabelRespFromDomain(*domainResp)
}

func (m *MagnetarGrpcServer) PutStringLabel(ctx context.Context, req *api.PutStringLabelReq) (*api.PutLabelResp, error) {
	domainReq, err := proto.PutStringLabelReqToDomain(req)
	if err != nil {
		return nil, err
	}
	domainResp, err := m.labelService.PutLabel(ctx, *domainReq)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return proto.PutLabelRespFromDomain(*domainResp)
}

func (m *MagnetarGrpcServer) DeleteLabel(ctx context.Context, req *api.DeleteLabelReq) (*api.DeleteLabelResp, error) {
	domainReq, err := proto.DeleteLabelReqToDomain(req)
	if err != nil {
		return nil, err
	}
	domainResp, err := m.labelService.DeleteLabel(ctx, *domainReq)
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) {
			return nil, status.Error(codes.PermissionDenied, err.Error())
		} else if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, err
	}
	return proto.DeleteLabelRespFromDomain(*domainResp)
}

func (m *MagnetarGrpcServer) ListAllNodes(ctx context.Context, req *api.ListAllNodesReq) (*api.ListAllNodesResp, error) {
	nodes, err := m.nodeService.ListAllNodes(ctx)
	if err != nil {
		return nil, err
	}
	return proto.ListAlldNodesRespFromDomain(nodes)
}

func (m *MagnetarGrpcServer) SetDesiredVms(ctx context.Context, req *api.VmDesiredSetReq) (*api.VmDesiredSetResp, error) {
	hostId, vms, err := proto.VmDesiredSetReqToDomain(req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := m.authorizeHost(ctx, "node.put", hostId.Value); err != nil {
		return nil, err
	}
	stored, err := m.vmService.SetDesiredVms(hostId, vms)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, err
	}
	return proto.VmDesiredSetRespFromDomain(stored), nil
}

func (m *MagnetarGrpcServer) GetDesiredVms(ctx context.Context, req *api.VmDesiredGetReq) (*api.VmDesiredGetResp, error) {
	if err := m.authorizeHost(ctx, "node.get", req.HostId); err != nil {
		return nil, err
	}
	vms, err := m.vmService.GetDesiredVms(proto.VmDesiredGetReqToDomain(req))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, err
	}
	return proto.VmDesiredGetRespFromDomain(vms), nil
}

func (m *MagnetarGrpcServer) GetVmStatuses(ctx context.Context, req *api.VmStatusGetReq) (*api.VmStatusGetResp, error) {
	if err := m.authorizeHost(ctx, "node.get", req.HostId); err != nil {
		return nil, err
	}
	statuses, err := m.vmService.GetVmStatuses(proto.VmStatusGetReqToDomain(req))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, err
	}
	return proto.VmStatusGetRespFromDomain(statuses), nil
}

func (m *MagnetarGrpcServer) SetVmStatuses(ctx context.Context, req *api.VmStatusSetReq) (*api.VmStatusSetResp, error) {
	hostId, statuses := proto.VmStatusSetReqToDomain(req)
	if err := m.authorizeHost(ctx, "node.put", hostId.Value); err != nil {
		return nil, err
	}
	if err := m.vmService.SetVmStatuses(hostId, statuses); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, err
	}
	return &api.VmStatusSetResp{}, nil
}

func GetAuthInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if ok && len(md.Get("authz-token")) > 0 {
			ctx = context.WithValue(ctx, "authz-token", md.Get("authz-token")[0])
		}
		// Calls the handler
		return handler(ctx, req)
	}
}

// SetDesiredOsImage records the operator's OS intent for a node (called by
// gravity before it notifies the node). Same permission as the update itself:
// os.update on the node or on its org.
func (m *MagnetarGrpcServer) SetDesiredOsImage(ctx context.Context, req *api.DesiredOsImageSetReq) (*api.DesiredOsImageSetResp, error) {
	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "nodeId is required")
	}
	// a per-node grant, or a grant on the node's org
	switch err := m.osService.AuthorizeOsUpdate(ctx, m.authorizer, domain.NodeId{Value: req.NodeId}); {
	case errors.Is(err, domain.ErrNodeNotFound):
		return nil, status.Error(codes.NotFound, err.Error())
	case err != nil:
		return nil, status.Error(codes.PermissionDenied, domain.ErrForbidden.Error())
	}
	err := m.osService.SetDesiredOsImage(domain.NodeId{Value: req.NodeId}, proto.DesiredOsImageToDomain(req.Image))
	if errors.Is(err, domain.ErrNodeNotFound) {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	if err != nil {
		return nil, err
	}
	return &api.DesiredOsImageSetResp{}, nil
}

// DeleteNode removes a node record completely (record, label index, liveness
// state, desired OS intent). Org-owned nodes are never removed automatically:
// this is the explicit operator path for them.
func (m *MagnetarGrpcServer) DeleteNode(ctx context.Context, req *api.DeleteNodeReq) (*api.DeleteNodeResp, error) {
	if req.NodeId == "" {
		return nil, status.Error(codes.InvalidArgument, "nodeId is required")
	}
	if err := m.authorizeHost(ctx, "node.delete", req.NodeId); err != nil {
		return nil, err
	}
	err := m.nodeService.DeleteNode(domain.NodeId{Value: req.NodeId})
	if errors.Is(err, domain.ErrNodeNotFound) {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	if err != nil {
		return nil, err
	}
	return &api.DeleteNodeResp{}, nil
}
