package servers

import (
	"log"

	"github.com/c12s/magnetar/internal/mappers/proto"
	"github.com/c12s/magnetar/internal/services"
	"github.com/c12s/magnetar/pkg/api"
	"github.com/c12s/magnetar/pkg/messaging"
)

// HeartbeatAsyncServer answers node liveness pings. It is intentionally
// request/reply: the agent needs to learn that Magnetar no longer knows it,
// so it can re-register (asserting its id) instead of heartbeating forever
// into the void.
type HeartbeatAsyncServer struct {
	subscriber messaging.Subscriber
	publisher  messaging.Publisher
	service    services.NodeUpdateService
}

func NewHeartbeatAsyncServer(subscriber messaging.Subscriber, publisher messaging.Publisher, service services.NodeUpdateService) (*HeartbeatAsyncServer, error) {
	return &HeartbeatAsyncServer{
		subscriber: subscriber,
		publisher:  publisher,
		service:    service,
	}, nil
}

func (h *HeartbeatAsyncServer) Serve() error {
	return h.subscriber.Subscribe(h.heartbeat)
}

func (h *HeartbeatAsyncServer) heartbeat(msg []byte, replySubject string) {
	reqProto := &api.NodeHeartbeatReq{}
	if err := reqProto.Unmarshal(msg); err != nil {
		log.Println(err)
		return
	}
	resp, err := h.service.Heartbeat(*proto.NodeHeartbeatReqToDomain(reqProto))
	if err != nil {
		log.Println(err)
		return
	}
	respMarshalled, err := proto.NodeHeartbeatRespFromDomain(*resp).Marshal()
	if err != nil {
		log.Println(err)
		return
	}
	if err := h.publisher.Publish(respMarshalled, replySubject); err != nil {
		log.Println(err)
	}
}

func (h *HeartbeatAsyncServer) GracefulStop() {
	if err := h.subscriber.Unsubscribe(); err != nil {
		log.Println(err)
	}
}
