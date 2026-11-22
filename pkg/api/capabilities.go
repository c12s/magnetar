package api

const (
	CapabilityConfig   = "config"
	CapabilityOsUpdate = "os_update"
	CapabilityGossip   = "gossip"
	CapabilityRunVms   = "run_vms"
)

const (
	OsPhaseHealthy    = "healthy"
	OsPhaseSwitching  = "switching"
	OsPhaseDegraded   = "degraded"
	OsPhaseRolledBack = "rolled_back"
	// no desired image declared (never updated, or after `image rollback`):
	// the node reports what it booted but nothing reconciles its OS
	OsPhaseUnmanaged = "unmanaged"
)

const (
	OsAttemptApplied    = "applied"
	OsAttemptRolledBack = "rolled_back"
	OsAttemptFailed     = "failed"
)
