package services

import (
	"log"

	"github.com/c12s/magnetar/internal/domain"
)

// VmService manages the desired-VM declaration and observed VM status on a
// host node's record (phase-next.md §2.6.2). Desired state is a standing
// declaration (like a Deployment), not a one-shot command: it lives in etcd
// and survives both CP and host restarts, so the host's reconciler can
// converge to it after any reboot.
type VmService struct {
	nodeRepo domain.NodeRepo
}

func NewVmService(nodeRepo domain.NodeRepo) (*VmService, error) {
	return &VmService{nodeRepo: nodeRepo}, nil
}

func (s *VmService) GetDesiredVms(hostId domain.NodeId) ([]domain.DesiredVm, error) {
	node, err := s.nodeRepo.FindById(hostId)
	if err != nil {
		log.Printf("vm desired get skipped, node %s not found", hostId.Value)
		return nil, err
	}
	return node.DesiredVms, nil
}

func (s *VmService) SetDesiredVms(hostId domain.NodeId, vms []domain.DesiredVm) ([]domain.DesiredVm, error) {
	node, err := s.nodeRepo.FindById(hostId)
	if err != nil {
		return nil, err
	}
	node.DesiredVms = vms
	if err := s.nodeRepo.Put(*node); err != nil {
		return nil, err
	}
	return node.DesiredVms, nil
}

func (s *VmService) GetVmStatuses(hostId domain.NodeId) ([]domain.VmStatus, error) {
	node, err := s.nodeRepo.FindById(hostId)
	if err != nil {
		log.Printf("vm status get skipped, node %s not found", hostId.Value)
		return nil, err
	}
	return node.VmStatus, nil
}

func (s *VmService) SetVmStatuses(hostId domain.NodeId, statuses []domain.VmStatus) error {
	node, err := s.nodeRepo.FindById(hostId)
	if err != nil {
		log.Printf("vm status update skipped, node %s not found", hostId.Value)
		return err
	}
	node.VmStatus = statuses
	return s.nodeRepo.Put(*node)
}
