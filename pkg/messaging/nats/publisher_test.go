package nats

import (
	"testing"

	"github.com/nats-io/nats.go"
)

func TestPublisherOnReconnectingConnection(t *testing.T) {
	// an enrolled node may start while the control plane is unreachable
	conn, err := nats.Connect("nats://127.0.0.1:1", nats.RetryOnFailedConnect(true), nats.MaxReconnects(-1))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := NewPublisher(conn); err != nil {
		t.Fatalf("publisher on a reconnecting connection: %v", err)
	}
	conn.Close()
	if _, err := NewPublisher(conn); err == nil {
		t.Fatal("publisher on a closed connection must fail")
	}
}
