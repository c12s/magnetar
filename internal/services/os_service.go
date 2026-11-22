package services

import (
	"context"

	"github.com/c12s/magnetar/internal/domain"
)

// OsService holds the operator's declared OS image per node. Observed OS
// state (os_status) is reported by the node through NodeUpdateService.
type OsService struct {
	nodeRepo domain.NodeRepo
}

func NewOsService(nodeRepo domain.NodeRepo) (*OsService, error) {
	return &OsService{nodeRepo: nodeRepo}, nil
}

// SetDesiredOsImage records (or, with nil, clears) the declared image of an
// existing node.
func (s *OsService) SetDesiredOsImage(nodeId domain.NodeId, image *domain.DesiredOsImage) error {
	if _, err := s.nodeRepo.FindById(nodeId); err != nil {
		return err
	}
	return s.nodeRepo.PutDesiredOsImage(nodeId, image)
}

// GetDesiredOsImage returns the declared image, nil if none;
// domain.ErrNodeNotFound if the node is unknown.
func (s *OsService) GetDesiredOsImage(nodeId domain.NodeId) (*domain.DesiredOsImage, error) {
	node, err := s.nodeRepo.FindById(nodeId)
	if err != nil {
		return nil, err
	}
	if node.DesiredOsImage == nil {
		return nil, nil
	}
	return node.DesiredOsImage, nil
}

// AuthorizeOsUpdate decides whether the caller may change a node's OS: a
// per-node grant (os.update|node|<id>) or a grant on the node's org
// (os.update|org|<org>). Pool nodes have no org, so only the per-node grant
// applies to them. Both an update and a rollback pass through here (gravity
// records the intent before it notifies the node), so this is the gate.
func (s *OsService) AuthorizeOsUpdate(ctx context.Context, authz AuthZService, nodeId domain.NodeId) error {
	if authz.Authorize(ctx, "os.update", "node", nodeId.Value) {
		return nil
	}
	node, err := s.nodeRepo.FindById(nodeId)
	if err != nil {
		return err
	}
	if node.Org != "" && authz.Authorize(ctx, "os.update", "org", node.Org) {
		return nil
	}
	return domain.ErrForbidden
}
