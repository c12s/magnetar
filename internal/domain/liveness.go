package domain

import "time"

// Liveness states exposed on the node record. They are derived from the last
// heartbeat, not stored: "offline" is a statement about the recent past, and
// a control-plane outage (during which no heartbeats arrive) must only
// degrade the view, never delete records.
const (
	LivenessOnline  = "online"
	LivenessStale   = "stale"
	LivenessOffline = "offline"
	// LivenessUnknown is a record that has never heartbeated (e.g. created
	// before heartbeats existed, or by an agent that does not send them).
	LivenessUnknown = "unknown"
)

// LivenessPolicy turns a last-seen timestamp into a display state.
type LivenessPolicy struct {
	StaleAfter   time.Duration
	OfflineAfter time.Duration
}

func (p LivenessPolicy) Liveness(lastSeen int64, now time.Time) string {
	if lastSeen <= 0 {
		return LivenessUnknown
	}
	age := now.Sub(time.Unix(lastSeen, 0))
	switch {
	case age <= p.StaleAfter:
		return LivenessOnline
	case age <= p.OfflineAfter:
		return LivenessStale
	default:
		return LivenessOffline
	}
}

func (p LivenessPolicy) Mark(node *Node, now time.Time) {
	node.Liveness = p.Liveness(node.LastSeen, now)
}

func (p LivenessPolicy) MarkAll(nodes []Node, now time.Time) {
	for i := range nodes {
		p.Mark(&nodes[i], now)
	}
}
