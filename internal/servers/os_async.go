package servers

import (
	"errors"
	"log"

	"github.com/c12s/magnetar/internal/domain"
	"github.com/c12s/magnetar/internal/mappers/proto"
	"github.com/c12s/magnetar/internal/services"
	"github.com/c12s/magnetar/pkg/api"
	"github.com/c12s/magnetar/pkg/messaging"
)

// OsAsyncServer lets a node pull the OS image declared for it. The write path
// (operator, via gravity) is gRPC SetDesiredOsImage.
type OsAsyncServer struct {
	desiredGetSubscriber messaging.Subscriber
	publisher            messaging.Publisher
	service              services.OsService
}

func NewOsAsyncServer(desiredGetSubscriber messaging.Subscriber, publisher messaging.Publisher, service services.OsService) (*OsAsyncServer, error) {
	return &OsAsyncServer{
		desiredGetSubscriber: desiredGetSubscriber,
		publisher:            publisher,
		service:              service,
	}, nil
}

func (s *OsAsyncServer) Serve() error {
	return s.desiredGetSubscriber.Subscribe(s.getDesired)
}

func (s *OsAsyncServer) getDesired(msg []byte, replySubject string) {
	req := &api.DesiredOsImageGetReq{}
	if err := req.Unmarshal(msg); err != nil {
		log.Printf("os desired get: invalid request: %s", err)
		return
	}
	resp := &api.DesiredOsImageGetResp{}
	image, err := s.service.GetDesiredOsImage(domain.NodeId{Value: req.NodeId})
	switch {
	case errors.Is(err, domain.ErrNodeNotFound):
		resp.NotFound = true
	case err != nil:
		// no reply: the node keeps its local copy instead of acting on a guess
		log.Printf("os desired get: %s", err)
		return
	default:
		resp.Image = proto.DesiredOsImageFromDomain(image)
	}
	data, err := resp.Marshal()
	if err != nil {
		log.Printf("os desired get: marshal response: %s", err)
		return
	}
	if err := s.publisher.Publish(data, replySubject); err != nil {
		log.Printf("os desired get: publish response: %s", err)
	}
}

func (s *OsAsyncServer) GracefulStop() {
	if err := s.desiredGetSubscriber.Unsubscribe(); err != nil {
		log.Println(err)
	}
}
