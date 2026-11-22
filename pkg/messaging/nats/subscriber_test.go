package nats

import (
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func connect(t *testing.T) *nats.Conn {
	t.Helper()
	conn, err := nats.Connect("nats://localhost:4222", nats.Timeout(time.Second))
	if err != nil {
		t.Skipf("no NATS server on localhost:4222: %v", err)
	}
	t.Cleanup(conn.Close)
	return conn
}

func TestReplyInboxIgnoresNoRespondersStatus(t *testing.T) {
	conn := connect(t)
	inbox := nats.NewInbox()
	sub, _ := NewSubscriber(conn, inbox, "")
	got := make(chan []byte, 1)
	if err := sub.Subscribe(func(msg []byte, _ string) { got <- msg }); err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	if err := conn.PublishRequest("c12s.test.nobody."+nats.NewInbox(), inbox, []byte("req")); err != nil {
		t.Fatal(err)
	}
	select {
	case msg := <-got:
		t.Fatalf("no-responders status delivered as a reply (%d bytes)", len(msg))
	case <-time.After(500 * time.Millisecond):
	}
}

func TestReplyInboxDeliversEmptyReply(t *testing.T) {
	conn := connect(t)
	subject := "c12s.test.echo." + nats.NewInbox()
	responder, err := conn.Subscribe(subject, func(m *nats.Msg) { _ = m.Respond(nil) })
	if err != nil {
		t.Fatal(err)
	}
	defer responder.Unsubscribe()

	inbox := nats.NewInbox()
	sub, _ := NewSubscriber(conn, inbox, "")
	got := make(chan []byte, 1)
	if err := sub.Subscribe(func(msg []byte, _ string) { got <- msg }); err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	if err := conn.PublishRequest(subject, inbox, []byte("req")); err != nil {
		t.Fatal(err)
	}
	select {
	case <-got:
	case <-time.After(2 * time.Second):
		t.Fatal("empty reply from a real responder was dropped")
	}
}
