package api

import (
	"context"
	"fmt"
	"time"

	natsgo "github.com/nats-io/nats.go"
)

type protoMessage interface {
	Marshal() ([]byte, error)
	Unmarshal([]byte) error
}

// NatsOptions keep a connection usable across broker or uplink outages of
// any length: the first connect may fail (an enrolled node boots while the
// control plane is down) and reconnects never stop (the default gives up
// after ~2 min, leaving the client deaf until restarted). Requests made while
// disconnected fail on their own deadline.
func NatsOptions() []natsgo.Option {
	return []natsgo.Option{
		natsgo.RetryOnFailedConnect(true),
		natsgo.MaxReconnects(-1),
		natsgo.ReconnectWait(2 * time.Second),
	}
}

func connect(natsAddress string) (*natsgo.Conn, error) {
	return natsgo.Connect(fmt.Sprintf("nats://%s", natsAddress), NatsOptions()...)
}

// request does one NATS request/reply. The library manages the reply inbox
// (nothing to leak) and reports a missing service as natsgo.ErrNoResponders
// instead of an empty reply that would decode as a valid empty message.
// ctx must carry a deadline.
func request(ctx context.Context, conn *natsgo.Conn, subject string, req, resp protoMessage) error {
	data, err := req.Marshal()
	if err != nil {
		return err
	}
	msg, err := conn.RequestWithContext(ctx, subject, data)
	if err != nil {
		return fmt.Errorf("%s: %w", subject, err)
	}
	return resp.Unmarshal(msg.Data)
}
