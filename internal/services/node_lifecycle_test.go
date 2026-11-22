package services

import (
	"errors"
	"testing"
	"time"

	"github.com/c12s/magnetar/internal/domain"
)

// fakeRepo is an in-memory NodeRepo for the lifecycle tests. Unused methods
// fall through to the embedded nil interface and panic if ever called.
type fakeRepo struct {
	domain.NodeRepo
	nodes   map[string]*domain.Node
	seen    map[string]int64
	deleted []string
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		nodes: map[string]*domain.Node{},
		seen:  map[string]int64{},
	}
}

func (f *fakeRepo) FindById(id domain.NodeId) (*domain.Node, error) {
	node, ok := f.nodes[id.Value]
	if !ok {
		return nil, domain.ErrNodeNotFound
	}
	cp := *node
	return &cp, nil
}

func (f *fakeRepo) Put(node domain.Node) error {
	cp := node
	f.nodes[node.Id.Value] = &cp
	return nil
}

func (f *fakeRepo) DeleteNode(node domain.Node) error {
	delete(f.nodes, node.Id.Value)
	delete(f.seen, node.Id.Value)
	f.deleted = append(f.deleted, node.Id.Value)
	return nil
}

func (f *fakeRepo) ListAllNodes() ([]domain.Node, error) {
	nodes := make([]domain.Node, 0, len(f.nodes))
	for _, node := range f.nodes {
		nodes = append(nodes, *node)
	}
	return nodes, nil
}

func (f *fakeRepo) TouchSeen(id domain.NodeId, at int64) error {
	f.seen[id.Value] = at
	return nil
}

func (f *fakeRepo) DeleteSeen(id domain.NodeId) error {
	delete(f.seen, id.Value)
	return nil
}

func (f *fakeRepo) ListSeen() (map[string]int64, error) {
	seen := make(map[string]int64, len(f.seen))
	for id, at := range f.seen {
		seen[id] = at
	}
	return seen, nil
}

func TestRegisterNewNodeMintsIdAndTouchesSeen(t *testing.T) {
	repo := newFakeRepo()
	svc, _ := NewRegistrationService(repo)

	resp, err := svc.Register(domain.RegistrationReq{
		Labels:       []domain.Label{domain.NewStringLabel("platform", "linux")},
		BindAddress:  "10.0.0.5",
		Capabilities: []string{"config", "gossip"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.NodeId == "" {
		t.Fatal("expected a generated id")
	}
	node := repo.nodes[resp.NodeId]
	if node == nil {
		t.Fatal("node not stored")
	}
	if len(node.Labels) != 1 || node.BindAddress != "10.0.0.5" {
		t.Fatalf("unexpected record: %+v", node)
	}
	if repo.seen[resp.NodeId] == 0 {
		t.Fatal("registration must record liveness")
	}
}

func TestRegisterWithAssertedIdUpdatesInPlace(t *testing.T) {
	repo := newFakeRepo()
	svc, _ := NewRegistrationService(repo)

	const id = "11111111-1111-4111-8111-111111111111"
	repo.nodes[id] = &domain.Node{
		Id:           domain.NodeId{Value: id},
		Org:          "c12s",
		Labels:       []domain.Label{domain.NewStringLabel("platform", "old")},
		BindAddress:  "10.0.0.5",
		Capabilities: []string{"config"},
		DesiredVms:   []domain.DesiredVm{{Name: "vm-01"}},
		VmStatus:     []domain.VmStatus{{Name: "vm-01", State: "running"}},
	}
	repo.seen[id] = 1

	resp, err := svc.Register(domain.RegistrationReq{
		NodeId:       id,
		Labels:       []domain.Label{domain.NewStringLabel("platform", "new")},
		BindAddress:  "10.0.0.6",
		Capabilities: []string{"config", "gossip", "os_update"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.NodeId != id {
		t.Fatalf("id changed: got %s, want %s", resp.NodeId, id)
	}
	if len(repo.nodes) != 1 {
		t.Fatalf("re-registration created a second record: %d", len(repo.nodes))
	}
	node := repo.nodes[id]
	if node.Org != "c12s" {
		t.Fatalf("org ownership lost: %q", node.Org)
	}
	if len(node.DesiredVms) != 1 || node.DesiredVms[0].Name != "vm-01" {
		t.Fatalf("desired VMs lost: %+v", node.DesiredVms)
	}
	if len(node.VmStatus) != 1 {
		t.Fatalf("VM status lost: %+v", node.VmStatus)
	}
	if node.BindAddress != "10.0.0.6" || len(node.Capabilities) != 3 {
		t.Fatalf("node-owned fields not refreshed: %+v", node)
	}
	if node.Labels[0].StringValue() != "new" {
		t.Fatalf("labels not refreshed: %+v", node.Labels)
	}
	if repo.seen[id] < 2 {
		t.Fatal("re-registration must refresh liveness")
	}
}

func TestRegisterWithAssertedIdRecreatesLostRecord(t *testing.T) {
	repo := newFakeRepo()
	svc, _ := NewRegistrationService(repo)

	const id = "22222222-2222-4222-8222-222222222222"
	resp, err := svc.Register(domain.RegistrationReq{
		NodeId:      id,
		BindAddress: "10.0.0.7",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.NodeId != id {
		t.Fatalf("got %s, want the asserted id %s", resp.NodeId, id)
	}
	if repo.nodes[id] == nil {
		t.Fatal("record not recreated")
	}
}

func TestRegisterRejectsMalformedId(t *testing.T) {
	repo := newFakeRepo()
	svc, _ := NewRegistrationService(repo)

	_, err := svc.Register(domain.RegistrationReq{NodeId: "not-a-uuid"})
	if err == nil {
		t.Fatal("expected an error for a malformed asserted id")
	}
	if len(repo.nodes) != 0 {
		t.Fatal("nothing must be stored for a malformed id")
	}
}

func TestHeartbeatUnknownNode(t *testing.T) {
	repo := newFakeRepo()
	svc, _ := NewNodeUpdateService(repo)

	resp, err := svc.Heartbeat(domain.NodeHeartbeatReq{NodeId: domain.NodeId{Value: "ghost"}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Found {
		t.Fatal("unknown node must report found=false")
	}
	if _, ok := repo.seen["ghost"]; ok {
		t.Fatal("unknown node must not write liveness state")
	}
}

func TestHeartbeatKnownNodeTouchesSeen(t *testing.T) {
	repo := newFakeRepo()
	repo.nodes["n1"] = &domain.Node{Id: domain.NodeId{Value: "n1"}}
	svc, _ := NewNodeUpdateService(repo)

	before := time.Now().Unix()
	resp, err := svc.Heartbeat(domain.NodeHeartbeatReq{NodeId: domain.NodeId{Value: "n1"}})
	if err != nil {
		t.Fatal(err)
	}
	if !resp.Found {
		t.Fatal("known node must report found=true")
	}
	if at := repo.seen["n1"]; at < before {
		t.Fatalf("liveness not refreshed: %d < %d", at, before)
	}
}

func TestUpdateTouchesSeen(t *testing.T) {
	repo := newFakeRepo()
	repo.nodes["n1"] = &domain.Node{Id: domain.NodeId{Value: "n1"}}
	svc, _ := NewNodeUpdateService(repo)

	if _, err := svc.Update(domain.NodeUpdateReq{
		NodeId:       domain.NodeId{Value: "n1"},
		Capabilities: []string{"config"},
	}); err != nil {
		t.Fatal(err)
	}
	if repo.seen["n1"] == 0 {
		t.Fatal("a status update must also count as liveness evidence")
	}
	if _, err := svc.Update(domain.NodeUpdateReq{NodeId: domain.NodeId{Value: "ghost"}}); err != nil {
		t.Fatal(err)
	}
	if _, ok := repo.seen["ghost"]; ok {
		t.Fatal("an update for an unknown node must not create liveness state")
	}
}

func TestNodeGC(t *testing.T) {
	repo := newFakeRepo()
	now := time.Now()
	stale := now.Add(-25 * time.Hour).Unix()
	fresh := now.Add(-time.Minute).Unix()

	repo.nodes["pool-stale"] = &domain.Node{Id: domain.NodeId{Value: "pool-stale"}}
	repo.nodes["pool-fresh"] = &domain.Node{Id: domain.NodeId{Value: "pool-fresh"}}
	repo.nodes["pool-unseen"] = &domain.Node{Id: domain.NodeId{Value: "pool-unseen"}}
	repo.nodes["org-stale"] = &domain.Node{Id: domain.NodeId{Value: "org-stale"}, Org: "c12s"}
	repo.seen["pool-stale"] = stale
	repo.seen["pool-fresh"] = fresh
	repo.seen["org-stale"] = stale
	repo.seen["orphan"] = fresh

	gc, err := NewNodeGC(repo, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	deleted, orphans, err := gc.Sweep(now)
	if err != nil {
		t.Fatal(err)
	}

	if len(deleted) != 1 || deleted[0] != "pool-stale" {
		t.Fatalf("deleted %v, want [pool-stale]", deleted)
	}
	if len(orphans) != 1 || orphans[0] != "orphan" {
		t.Fatalf("orphan cleanup %v, want [orphan]", orphans)
	}
	if repo.nodes["pool-fresh"] == nil || repo.nodes["pool-unseen"] == nil || repo.nodes["org-stale"] == nil {
		t.Fatalf("unexpected deletions: %v", repo.deleted)
	}
	if _, ok := repo.seen["pool-stale"]; ok {
		t.Fatal("deleted node left liveness state behind")
	}
	if _, ok := repo.seen["orphan"]; ok {
		t.Fatal("orphan liveness key not removed")
	}
}

func TestNodeGCDoesNotDeleteUnseenRecords(t *testing.T) {
	repo := newFakeRepo()
	// a pre-heartbeat record: no liveness state at all
	repo.nodes["old"] = &domain.Node{Id: domain.NodeId{Value: "old"}}

	gc, _ := NewNodeGC(repo, time.Second)
	deleted, _, err := gc.Sweep(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(deleted) != 0 {
		t.Fatalf("deleted %v, want none", deleted)
	}
	if repo.nodes["old"] == nil {
		t.Fatal("record without liveness state must be kept")
	}
}

func TestDeleteNodeService(t *testing.T) {
	repo := newFakeRepo()
	repo.nodes["n1"] = &domain.Node{Id: domain.NodeId{Value: "n1"}}
	svc, _ := NewNodeService(repo, nil, nil, AuthZService{}, nil, nil, domain.LivenessPolicy{})

	if err := svc.DeleteNode(domain.NodeId{Value: "n1"}); err != nil {
		t.Fatal(err)
	}
	if repo.nodes["n1"] != nil {
		t.Fatal("record not deleted")
	}
	if err := svc.DeleteNode(domain.NodeId{Value: "ghost"}); !errors.Is(err, domain.ErrNodeNotFound) {
		t.Fatalf("want ErrNodeNotFound, got %v", err)
	}
}
