package servers

import (
	"log"

	"github.com/c12s/magnetar/internal/mappers/proto"
	"github.com/c12s/magnetar/internal/services"
	"github.com/c12s/magnetar/pkg/api"
	"github.com/c12s/magnetar/pkg/messaging"
)

// VmAsyncServer serves the host-facing side of VM management over NATS:
// a host's Star polls the desired VM set and reports observed status. The
// CP-facing write path (operator sets desired VMs) is gRPC, see
// magnetar_grpc.go.
type VmAsyncServer struct {
	desiredGetSubscriber messaging.Subscriber
	statusSetSubscriber  messaging.Subscriber
	publisher            messaging.Publisher
	service              services.VmService
}

func NewVmAsyncServer(desiredGetSubscriber messaging.Subscriber, statusSetSubscriber messaging.Subscriber, publisher messaging.Publisher, service services.VmService) (*VmAsyncServer, error) {
	return &VmAsyncServer{
		desiredGetSubscriber: desiredGetSubscriber,
		statusSetSubscriber:  statusSetSubscriber,
		publisher:            publisher,
		service:              service,
	}, nil
}

func (s *VmAsyncServer) Serve() error {
	if err := s.desiredGetSubscriber.Subscribe(s.getDesired); err != nil {
		return err
	}
	return s.statusSetSubscriber.Subscribe(s.setStatuses)
}

func (s *VmAsyncServer) getDesired(msg []byte, replySubject string) {
	req := &api.VmDesiredGetReq{}
	if err := req.Unmarshal(msg); err != nil {
		log.Printf("vm desired get: invalid request: %s", err)
		return
	}
	vms, err := s.service.GetDesiredVms(proto.VmDesiredGetReqToDomain(req))
	if err != nil {
		log.Printf("vm desired get: %s", err)
		return
	}
	resp, err := proto.VmDesiredGetRespFromDomain(vms).Marshal()
	if err != nil {
		log.Printf("vm desired get: marshal response: %s", err)
		return
	}
	if err := s.publisher.Publish(resp, replySubject); err != nil {
		log.Printf("vm desired get: publish response: %s", err)
	}
}

func (s *VmAsyncServer) setStatuses(msg []byte, replySubject string) {
	req := &api.VmStatusSetReq{}
	if err := req.Unmarshal(msg); err != nil {
		log.Printf("vm status set: invalid request: %s", err)
		return
	}
	hostId, statuses := proto.VmStatusSetReqToDomain(req)
	if err := s.service.SetVmStatuses(hostId, statuses); err != nil {
		log.Printf("vm status set: %s", err)
		return
	}
	resp, err := (&api.VmStatusSetResp{}).Marshal()
	if err != nil {
		log.Printf("vm status set: marshal response: %s", err)
		return
	}
	if err := s.publisher.Publish(resp, replySubject); err != nil {
		log.Printf("vm status set: publish response: %s", err)
	}
}

func (s *VmAsyncServer) GracefulStop() {
	if err := s.desiredGetSubscriber.Unsubscribe(); err != nil {
		log.Println(err)
	}
	if err := s.statusSetSubscriber.Unsubscribe(); err != nil {
		log.Println(err)
	}
}
