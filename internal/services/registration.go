package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/c12s/magnetar/internal/domain"
	"github.com/google/uuid"
)

type RegistrationService struct {
	nodeRepo domain.NodeRepo
}

func NewRegistrationService(nodeRepo domain.NodeRepo) (*RegistrationService, error) {
	return &RegistrationService{
		nodeRepo: nodeRepo,
	}, nil
}

// Register enrolls a node. A request that carries a node id (an agent that
// still has its id file) is idempotent: the existing record is updated in
// place — or recreated with the same id when the control plane lost it —
// instead of producing a second record with a fresh UUID.
func (r *RegistrationService) Register(req domain.RegistrationReq) (*domain.RegistrationResp, error) {
	id := req.NodeId
	if id == "" {
		id = uuid.NewString()
	} else if _, err := uuid.Parse(id); err != nil {
		return nil, fmt.Errorf("invalid node id %q: %w", id, err)
	}

	existing, err := r.nodeRepo.FindById(domain.NodeId{Value: id})
	switch {
	case err == nil:
		return r.updateExisting(existing, req)
	case !errors.Is(err, domain.ErrNodeNotFound):
		return nil, err
	}

	node := domain.Node{
		Id:           domain.NodeId{Value: id},
		Labels:       req.Labels,
		Resources:    req.Resources,
		BindAddress:  req.BindAddress,
		Capabilities: req.Capabilities,
		OsStatus:     req.OsStatus,
	}
	if err := r.nodeRepo.Put(node); err != nil {
		return nil, err
	}
	r.touch(node.Id)
	return &domain.RegistrationResp{NodeId: id}, nil
}

// updateExisting refreshes the fields the node owns (labels, resources,
// address, capabilities, OS status) while preserving what the operator owns:
// org ownership, desired VMs and VM status. The desired OS image lives under
// its own key and is untouched either way.
func (r *RegistrationService) updateExisting(node *domain.Node, req domain.RegistrationReq) (*domain.RegistrationResp, error) {
	if node.BindAddress != "" && req.BindAddress != "" && node.BindAddress != req.BindAddress {
		log.Printf("node %s re-registered from %s (was %s); updating the record in place",
			node.Id.Value, req.BindAddress, node.BindAddress)
	}
	node.Labels = req.Labels
	node.Resources = req.Resources
	node.BindAddress = req.BindAddress
	node.Capabilities = req.Capabilities
	if req.OsStatus != nil {
		node.OsStatus = req.OsStatus
	}
	if err := r.nodeRepo.Put(*node); err != nil {
		return nil, err
	}
	r.touch(node.Id)
	return &domain.RegistrationResp{NodeId: node.Id.Value}, nil
}

func (r *RegistrationService) touch(id domain.NodeId) {
	if err := r.nodeRepo.TouchSeen(id, time.Now().Unix()); err != nil {
		log.Printf("registration: could not record liveness for %s: %v", id.Value, err)
	}
}
