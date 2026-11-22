package api

import (
	"context"
	"errors"
	"testing"
	"time"

	natsgo "github.com/nats-io/nats.go"
)

func testConn(t *testing.T) *natsgo.Conn {
	t.Helper()
	conn, err := natsgo.Connect("nats://localhost:4222", natsgo.Timeout(time.Second))
	if err != nil {
		t.Skipf("no NATS server on localhost:4222: %v", err)
	}
	t.Cleanup(conn.Close)
	return conn
}

func TestRequestWithoutResponderIsAnError(t *testing.T) {
	conn := testConn(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp := &NodeUpdateResp{}
	err := request(ctx, conn, "c12s.test.nobody."+natsgo.NewInbox(), &NodeUpdateReq{NodeId: "n"}, resp)
	if !errors.Is(err, natsgo.ErrNoResponders) {
		t.Fatalf("want ErrNoResponders, got %v", err)
	}
}

func TestRequestDecodesReply(t *testing.T) {
	conn := testConn(t)
	subject := "c12s.test.update." + natsgo.NewInbox()
	sub, err := conn.Subscribe(subject, func(m *natsgo.Msg) {
		data, _ := (&NodeUpdateResp{NotFound: true}).Marshal()
		_ = m.Respond(data)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resp := &NodeUpdateResp{}
	if err := request(ctx, conn, subject, &NodeUpdateReq{NodeId: "n"}, resp); err != nil {
		t.Fatal(err)
	}
	if !resp.NotFound {
		t.Fatal("reply not decoded")
	}
}

func TestConnectWithoutBroker(t *testing.T) {
	// an enrolled node may boot while the control plane is down
	conn, err := connect("127.0.0.1:1")
	if err != nil {
		t.Fatalf("connect must not fail without a broker: %v", err)
	}
	defer conn.Close()
	if conn.IsClosed() {
		t.Fatal("connection closed instead of retrying")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := request(ctx, conn, "x", &DesiredOsImageGetReq{}, &DesiredOsImageGetResp{}); err == nil {
		t.Fatal("request without a broker must fail")
	}
}
