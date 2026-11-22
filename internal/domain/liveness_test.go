package domain

import (
	"testing"
	"time"
)

func TestLivenessPolicyStates(t *testing.T) {
	policy := LivenessPolicy{StaleAfter: 90 * time.Second, OfflineAfter: 5 * time.Minute}
	now := time.Unix(1_800_000_000, 0)

	cases := []struct {
		name     string
		lastSeen int64
		want     string
	}{
		{"never seen", 0, LivenessUnknown},
		{"just now", now.Unix(), LivenessOnline},
		{"at the stale edge", now.Add(-90 * time.Second).Unix(), LivenessOnline},
		{"one second past stale", now.Add(-91 * time.Second).Unix(), LivenessStale},
		{"at the offline edge", now.Add(-5 * time.Minute).Unix(), LivenessStale},
		{"one second past offline", now.Add(-5*time.Minute - time.Second).Unix(), LivenessOffline},
		{"future timestamp counts as online", now.Add(time.Minute).Unix(), LivenessOnline},
	}
	for _, tc := range cases {
		if got := policy.Liveness(tc.lastSeen, now); got != tc.want {
			t.Errorf("%s: got %s, want %s", tc.name, got, tc.want)
		}
	}
}

func TestLivenessPolicyMarkAll(t *testing.T) {
	policy := LivenessPolicy{StaleAfter: time.Minute, OfflineAfter: 2 * time.Minute}
	now := time.Unix(1_800_000_000, 0)
	nodes := []Node{
		{Id: NodeId{Value: "a"}, LastSeen: now.Unix()},
		{Id: NodeId{Value: "b"}, LastSeen: now.Add(-3 * time.Minute).Unix()},
		{Id: NodeId{Value: "c"}},
	}
	policy.MarkAll(nodes, now)
	if nodes[0].Liveness != LivenessOnline || nodes[1].Liveness != LivenessOffline || nodes[2].Liveness != LivenessUnknown {
		t.Fatalf("unexpected states: %s, %s, %s", nodes[0].Liveness, nodes[1].Liveness, nodes[2].Liveness)
	}
}
