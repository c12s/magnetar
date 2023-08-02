package startup

import (
	"github.com/c12s/magnetar/pkg/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
)

func startServer(address string, server api.MagnetarServer) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	api.RegisterMagnetarServer(s, server)
	log.Printf("server listening at %v", lis.Addr())
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}