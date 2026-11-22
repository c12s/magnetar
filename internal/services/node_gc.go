package services

import (
	"context"
	"log"
	"sort"
	"time"

	"github.com/c12s/magnetar/internal/domain"
)

// NodeGC reaps node records that can no longer be real. Policy:
//   - only pool (unclaimed) records are deleted automatically; org-owned
//     nodes are removed only by the explicit DeleteNode RPC;
//   - only records with a known last heartbeat are considered. A record
//     without liveness state (created before heartbeats existed) is left
//     alone, so an upgrade never deletes a fleet that has not reported in yet;
//   - a pool record is deleted after PoolAfter without a heartbeat.
//
// Orphaned liveness keys (record deleted out of band) are cleaned too.
type NodeGC struct {
	nodeRepo  domain.NodeRepo
	poolAfter time.Duration
}

func NewNodeGC(nodeRepo domain.NodeRepo, poolAfter time.Duration) (*NodeGC, error) {
	return &NodeGC{
		nodeRepo:  nodeRepo,
		poolAfter: poolAfter,
	}, nil
}

// Sweep runs one GC pass and returns what it removed (sorted, for stable
// logs and tests).
func (g *NodeGC) Sweep(now time.Time) (deletedNodes, orphanSeen []string, err error) {
	nodes, err := g.nodeRepo.ListAllNodes()
	if err != nil {
		return nil, nil, err
	}
	seen, err := g.nodeRepo.ListSeen()
	if err != nil {
		return nil, nil, err
	}

	known := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		known[node.Id.Value] = struct{}{}
		if node.Claimed() {
			continue
		}
		last, ok := seen[node.Id.Value]
		if !ok {
			continue
		}
		if now.Sub(time.Unix(last, 0)) <= g.poolAfter {
			continue
		}
		if err := g.nodeRepo.DeleteNode(node); err != nil {
			log.Printf("node gc: deleting %s failed: %v", node.Id.Value, err)
			continue
		}
		deletedNodes = append(deletedNodes, node.Id.Value)
	}
	for id := range seen {
		if _, ok := known[id]; ok {
			continue
		}
		if err := g.nodeRepo.DeleteSeen(domain.NodeId{Value: id}); err != nil {
			log.Printf("node gc: cleaning liveness of %s failed: %v", id, err)
			continue
		}
		orphanSeen = append(orphanSeen, id)
	}
	sort.Strings(deletedNodes)
	sort.Strings(orphanSeen)
	return deletedNodes, orphanSeen, nil
}

// Run sweeps every interval until ctx is done.
func (g *NodeGC) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			deleted, orphans, err := g.Sweep(now)
			if err != nil {
				log.Printf("node gc: sweep failed: %v", err)
				continue
			}
			if len(deleted) > 0 || len(orphans) > 0 {
				log.Printf("node gc: deleted %d stale pool node(s) %v, cleaned %d orphan liveness key(s) %v",
					len(deleted), deleted, len(orphans), orphans)
			}
		}
	}
}
