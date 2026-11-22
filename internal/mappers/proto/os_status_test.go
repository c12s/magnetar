package proto

import (
	"testing"

	"github.com/c12s/magnetar/internal/domain"
)

// The rollback deployment must survive storage: `image rollback` reads it.
func TestOsStatusRoundTripKeepsRollback(t *testing.T) {
	in := &domain.OsStatus{
		BootedImage: "r/os:v4.1", BootedDigest: "sha256:new", Phase: "healthy",
		RollbackImage: "r/os:v4.0", RollbackDigest: "sha256:old",
	}
	out := OsStatusToDomain(OsStatusFromDomain(in))
	if out.RollbackImage != in.RollbackImage || out.RollbackDigest != in.RollbackDigest {
		t.Fatalf("rollback lost: %+v", out)
	}
}
