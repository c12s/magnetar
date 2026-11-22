package services

import (
	"errors"
	"log"
	"time"

	"github.com/c12s/magnetar/internal/domain"
)

type NodeUpdateService struct {
	nodeRepo domain.NodeRepo
}

func NewNodeUpdateService(nodeRepo domain.NodeRepo) (*NodeUpdateService, error) {
	return &NodeUpdateService{
		nodeRepo: nodeRepo,
	}, nil
}

func (s *NodeUpdateService) Update(req domain.NodeUpdateReq) (*domain.NodeUpdateResp, error) {
	node, err := s.nodeRepo.FindById(req.NodeId)
	if errors.Is(err, domain.ErrNodeNotFound) {
		log.Printf("node update skipped, node %s not found", req.NodeId.Value)
		return &domain.NodeUpdateResp{NotFound: true}, nil
	}
	if err != nil {
		return nil, err
	}
	// only overwrite fields the reporter actually sent: a nil field means
	// "unchanged", otherwise every status tick would clobber capabilities
	if req.Capabilities != nil {
		node.Capabilities = req.Capabilities
	}
	if req.OsStatus != nil {
		node.OsStatus = req.OsStatus
	}
	err = s.nodeRepo.Put(*node)
	if err != nil {
		return nil, err
	}
	s.touch(node.Id)
	return &domain.NodeUpdateResp{}, nil
}

// Heartbeat records liveness and reports whether Magnetar still knows the
// node. It never rewrites the node record, so it cannot race (or clobber)
// capability reports, OS-status ticks or label writes.
func (s *NodeUpdateService) Heartbeat(req domain.NodeHeartbeatReq) (*domain.NodeHeartbeatResp, error) {
	_, err := s.nodeRepo.FindById(req.NodeId)
	if errors.Is(err, domain.ErrNodeNotFound) {
		return &domain.NodeHeartbeatResp{Found: false}, nil
	}
	if err != nil {
		return nil, err
	}
	if err := s.nodeRepo.TouchSeen(req.NodeId, time.Now().Unix()); err != nil {
		return nil, err
	}
	return &domain.NodeHeartbeatResp{Found: true}, nil
}

func (s *NodeUpdateService) touch(id domain.NodeId) {
	if err := s.nodeRepo.TouchSeen(id, time.Now().Unix()); err != nil {
		log.Printf("node update: could not record liveness for %s: %v", id.Value, err)
	}
}
