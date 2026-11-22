package services

import (
	"errors"
	"testing"

	"github.com/c12s/magnetar/internal/domain"
)

type fakeNodeRepo struct {
	domain.NodeRepo
	nodes   map[string]bool
	orgs    map[string]string
	desired map[string]*domain.DesiredOsImage
}

func (f *fakeNodeRepo) FindById(id domain.NodeId) (*domain.Node, error) {
	if !f.nodes[id.Value] {
		return nil, domain.ErrNodeNotFound
	}
	return &domain.Node{Id: id, Org: f.orgs[id.Value], DesiredOsImage: f.desired[id.Value]}, nil
}

func (f *fakeNodeRepo) PutDesiredOsImage(id domain.NodeId, image *domain.DesiredOsImage) error {
	if image == nil {
		delete(f.desired, id.Value)
		return nil
	}
	f.desired[id.Value] = image
	return nil
}

func TestOsServiceDesiredImage(t *testing.T) {
	repo := &fakeNodeRepo{nodes: map[string]bool{"n1": true}, desired: map[string]*domain.DesiredOsImage{}}
	svc, _ := NewOsService(repo)
	n1 := domain.NodeId{Value: "n1"}

	if img, err := svc.GetDesiredOsImage(n1); err != nil || img != nil {
		t.Fatalf("fresh node: got %v, %v; want nil, nil", img, err)
	}
	want := &domain.DesiredOsImage{ImageRef: "r", Digest: "sha256:x"}
	if err := svc.SetDesiredOsImage(n1, want); err != nil {
		t.Fatal(err)
	}
	if img, _ := svc.GetDesiredOsImage(n1); img == nil || img.Digest != "sha256:x" {
		t.Fatalf("after set: got %v", img)
	}
	if err := svc.SetDesiredOsImage(n1, nil); err != nil {
		t.Fatal(err)
	}
	if img, _ := svc.GetDesiredOsImage(n1); img != nil {
		t.Fatalf("after clear: got %v", img)
	}
	unknown := domain.NodeId{Value: "nope"}
	if err := svc.SetDesiredOsImage(unknown, want); !errors.Is(err, domain.ErrNodeNotFound) {
		t.Fatalf("set on unknown node: %v", err)
	}
	if _, err := svc.GetDesiredOsImage(unknown); !errors.Is(err, domain.ErrNodeNotFound) {
		t.Fatalf("get on unknown node: %v", err)
	}
}
