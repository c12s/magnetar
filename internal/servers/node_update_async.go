package servers

import (
	"log"

	"github.com/c12s/magnetar/internal/mappers/proto"
	"github.com/c12s/magnetar/internal/services"
	"github.com/c12s/magnetar/pkg/api"
	"github.com/c12s/magnetar/pkg/messaging"
)

type NodeUpdateAsyncServer struct {
	subscriber messaging.Subscriber
	publisher  messaging.Publisher
	service    services.NodeUpdateService
}

func NewNodeUpdateAsyncServer(subscriber messaging.Subscriber, publisher messaging.Publisher, service services.NodeUpdateService) (*NodeUpdateAsyncServer, error) {
	return &NodeUpdateAsyncServer{
		subscriber: subscriber,
		publisher:  publisher,
		service:    service,
	}, nil
}

func (n *NodeUpdateAsyncServer) Serve() error {
	return n.subscriber.Subscribe(n.update)
}

func (n *NodeUpdateAsyncServer) update(msg []byte, replySubject string) {
	reqProto := &api.NodeUpdateReq{}
	err := reqProto.Unmarshal(msg)
	if err != nil {
		log.Println(err)
		return
	}
	req, err := proto.NodeUpdateReqToDomain(reqProto)
	if err != nil {
		log.Println(err)
		return
	}
	resp, err := n.service.Update(*req)
	if err != nil {
		log.Println(err)
		return
	}
	respProto, err := proto.NodeUpdateRespFromDomain(*resp)
	if err != nil {
		log.Println(err)
		return
	}
	respMarshalled, err := respProto.Marshal()
	if err != nil {
		log.Println(err)
		return
	}
	err = n.publisher.Publish(respMarshalled, replySubject)
	if err != nil {
		log.Println(err)
	}
}

func (n *NodeUpdateAsyncServer) GracefulStop() {
	err := n.subscriber.Unsubscribe()
	if err != nil {
		log.Println(err)
	}
}
