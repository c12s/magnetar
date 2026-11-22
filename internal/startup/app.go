package startup

import (
	"context"
	"errors"
	"log"
	"net"
	"sync"

	gravity_api "github.com/c12s/agent_queue/pkg/api"
	"github.com/c12s/magnetar/internal/configs"
	"github.com/c12s/magnetar/internal/domain"
	"github.com/c12s/magnetar/internal/marshallers/proto"
	"github.com/c12s/magnetar/internal/repos"
	"github.com/c12s/magnetar/internal/servers"
	"github.com/c12s/magnetar/internal/services"
	"github.com/c12s/magnetar/pkg/api"
	"github.com/c12s/magnetar/pkg/messaging"
	"github.com/c12s/magnetar/pkg/messaging/nats"
	meridian_api "github.com/c12s/meridian/pkg/api"
	oortapi "github.com/c12s/oort/pkg/api"
	natsgo "github.com/nats-io/nats.go"
	etcd "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

type app struct {
	config                    *configs.Config
	grpcServer                *grpc.Server
	magnetarServer            api.MagnetarServer
	registrationServer        *servers.RegistrationAsyncServer
	nodeUpdateServer          *servers.NodeUpdateAsyncServer
	heartbeatServer           *servers.HeartbeatAsyncServer
	nodeGC                    *services.NodeGC
	vmAsyncServer             *servers.VmAsyncServer
	osAsyncServer             *servers.OsAsyncServer
	osService                 *services.OsService
	osDesiredGetSubscriber    messaging.Subscriber
	nodeService               *services.NodeService
	labelService              *services.LabelService
	vmService                 *services.VmService
	authzService              services.AuthZService
	registrationService       *services.RegistrationService
	nodeUpdateService         *services.NodeUpdateService
	evaluatorClient           oortapi.OortEvaluatorClient
	administratorClient       *oortapi.AdministrationAsyncClient
	meridian                  meridian_api.MeridianClient
	gravity                   gravity_api.AgentQueueClient
	publisher                 messaging.Publisher
	registrationSubscriber    messaging.Subscriber
	nodeUpdateSubscriber      messaging.Subscriber
	heartbeatSubscriber       messaging.Subscriber
	vmDesiredGetSubscriber    messaging.Subscriber
	vmStatusSetSubscriber     messaging.Subscriber
	nodeRepo                  domain.NodeRepo
	nodeMarshaller            domain.NodeMarshaller
	labelMarshaller           domain.LabelMarshaller
	shutdownProcesses         []func()
	gracefulShutdownProcesses []func(wg *sync.WaitGroup)
}

func NewAppWithConfig(config *configs.Config) (*app, error) {
	if config == nil {
		return nil, errors.New("config is nil")
	}
	return &app{
		config:                    config,
		shutdownProcesses:         make([]func(), 0),
		gracefulShutdownProcesses: make([]func(wg *sync.WaitGroup), 0),
	}, nil
}

func (a *app) Start() error {
	a.init()

	err := a.startRegistrationServer()
	if err != nil {
		return err
	}
	err = a.startNodeUpdateServer()
	if err != nil {
		return err
	}
	err = a.startHeartbeatServer()
	if err != nil {
		return err
	}
	err = a.startNodeGC()
	if err != nil {
		return err
	}
	err = a.startVmAsyncServer()
	if err != nil {
		return err
	}
	err = a.startOsAsyncServer()
	if err != nil {
		return err
	}
	return a.startGrpcServer()
}

func (a *app) GracefulStop(ctx context.Context) {
	// call all shutdown processes after a timeout or graceful shutdown processes completion
	defer a.shutdown()

	// wait for all graceful shutdown processes to complete
	wg := &sync.WaitGroup{}
	wg.Add(len(a.gracefulShutdownProcesses))

	for _, gracefulShutdownProcess := range a.gracefulShutdownProcesses {
		go gracefulShutdownProcess(wg)
	}

	// notify when graceful shutdown processes are done
	gracefulShutdownDone := make(chan struct{})
	go func() {
		wg.Wait()
		gracefulShutdownDone <- struct{}{}
	}()

	// wait for graceful shutdown processes to complete or for ctx timeout
	select {
	case <-ctx.Done():
		log.Println("ctx timeout ... shutting down")
	case <-gracefulShutdownDone:
		log.Println("app gracefully stopped")
	}
}

func (a *app) init() {
	natsConn, err := NewNatsConn(a.config.NatsAddress())
	if err != nil {
		log.Fatalln(err)
	}
	a.shutdownProcesses = append(a.shutdownProcesses, func() {
		log.Println("closing nats conn")
		natsConn.Close()
	})

	etcdClient, err := newEtcdClient(a.config.EtcdAddress())
	if err != nil {
		log.Fatalln(err)
	}
	a.shutdownProcesses = append(a.shutdownProcesses, func() {
		log.Println("closing etcd client conn")
		err := etcdClient.Close()
		if err != nil {
			log.Println(err)
		}
	})

	a.initNatsPublisher(natsConn)
	a.initRegistrationNatsSubscriber(natsConn)
	a.initNodeUpdateNatsSubscriber(natsConn)
	a.initHeartbeatNatsSubscriber(natsConn)
	a.initVmNatsSubscribers(natsConn)

	a.initNodeProtoMarshaller()
	a.initLabelProtoMarshaller()
	a.initNodeEtcdRepo(etcdClient)

	a.initAdministratorClient()
	a.initEvaluatorClient()
	a.initMeridian()
	a.initGravity()

	a.initAuthZService()
	a.initNodeService()
	a.initLabelService()
	a.initVmService()
	a.initOsService()
	a.initRegistrationService()
	a.initNodeUpdateService()

	a.initRegistrationServer()
	a.initNodeUpdateServer()
	a.initHeartbeatServer()
	a.initNodeGC()
	a.initVmAsyncServer()
	a.initOsAsyncServer()
	a.initMagnetarServer()
	a.initGrpcServer()
}

func (a *app) initGrpcServer() {
	if a.magnetarServer == nil {
		log.Fatalln("magnetar server is nil")
	}
	s := grpc.NewServer(grpc.UnaryInterceptor(servers.GetAuthInterceptor()))
	api.RegisterMagnetarServer(s, a.magnetarServer)
	reflection.Register(s)
	a.grpcServer = s
}

func (a *app) initMagnetarServer() {
	if a.nodeService == nil {
		log.Fatalln("node service is nil")
	}
	if a.labelService == nil {
		log.Fatalln("label service is nil")
	}
	if a.vmService == nil {
		log.Fatalln("vm service is nil")
	}
	magnetarServer, err := servers.NewMagnetarGrpcServer(*a.nodeService, *a.labelService, *a.vmService, *a.osService, a.authzService)
	if err != nil {
		log.Fatalln(err)
	}
	a.magnetarServer = magnetarServer
}

func (a *app) initRegistrationServer() {
	if a.registrationService == nil {
		log.Fatalln("registration service is nil")
	}
	if a.publisher == nil {
		log.Fatalln("publisher is nil")
	}
	if a.registrationSubscriber == nil {
		log.Fatalln("registration req subscriber is nil")
	}
	server, err := servers.NewRegistrationAsyncServer(a.registrationSubscriber, a.publisher, *a.registrationService)
	if err != nil {
		log.Fatalln(err)
	}
	a.registrationServer = server
}

func (a *app) initRegistrationService() {
	if a.nodeRepo == nil {
		log.Fatalln("node repo is nil")
	}
	registrationService, err := services.NewRegistrationService(a.nodeRepo)
	if err != nil {
		log.Fatalln(err)
	}
	a.registrationService = registrationService
}

func (a *app) initNodeUpdateService() {
	if a.nodeRepo == nil {
		log.Fatalln("node repo is nil")
	}
	nodeUpdateService, err := services.NewNodeUpdateService(a.nodeRepo)
	if err != nil {
		log.Fatalln(err)
	}
	a.nodeUpdateService = nodeUpdateService
}

func (a *app) initVmService() {
	if a.nodeRepo == nil {
		log.Fatalln("node repo is nil")
	}
	vmService, err := services.NewVmService(a.nodeRepo)
	if err != nil {
		log.Fatalln(err)
	}
	a.vmService = vmService
}

func (a *app) initOsService() {
	osService, err := services.NewOsService(a.nodeRepo)
	if err != nil {
		log.Fatalln(err)
	}
	a.osService = osService
}

func (a *app) initOsAsyncServer() {
	server, err := servers.NewOsAsyncServer(a.osDesiredGetSubscriber, a.publisher, *a.osService)
	if err != nil {
		log.Fatalln(err)
	}
	a.osAsyncServer = server
}

func (a *app) startOsAsyncServer() error {
	if err := a.osAsyncServer.Serve(); err != nil {
		return err
	}
	a.gracefulShutdownProcesses = append(a.gracefulShutdownProcesses, func(wg *sync.WaitGroup) {
		a.osAsyncServer.GracefulStop()
		log.Println("os async server gracefully stopped")
		wg.Done()
	})
	return nil
}

func (a *app) initVmAsyncServer() {
	if a.vmService == nil {
		log.Fatalln("vm service is nil")
	}
	if a.publisher == nil {
		log.Fatalln("publisher is nil")
	}
	if a.vmDesiredGetSubscriber == nil {
		log.Fatalln("vm desired get subscriber is nil")
	}
	if a.vmStatusSetSubscriber == nil {
		log.Fatalln("vm status set subscriber is nil")
	}
	server, err := servers.NewVmAsyncServer(a.vmDesiredGetSubscriber, a.vmStatusSetSubscriber, a.publisher, *a.vmService)
	if err != nil {
		log.Fatalln(err)
	}
	a.vmAsyncServer = server
}

func (a *app) initHeartbeatServer() {
	if a.nodeUpdateService == nil {
		log.Fatalln("node update service is nil")
	}
	if a.publisher == nil {
		log.Fatalln("publisher is nil")
	}
	if a.heartbeatSubscriber == nil {
		log.Fatalln("heartbeat subscriber is nil")
	}
	server, err := servers.NewHeartbeatAsyncServer(a.heartbeatSubscriber, a.publisher, *a.nodeUpdateService)
	if err != nil {
		log.Fatalln(err)
	}
	a.heartbeatServer = server
}

func (a *app) initNodeGC() {
	if a.nodeRepo == nil {
		log.Fatalln("node repo is nil")
	}
	nodeGC, err := services.NewNodeGC(a.nodeRepo, a.config.GCPoolAfter())
	if err != nil {
		log.Fatalln(err)
	}
	a.nodeGC = nodeGC
}

func (a *app) initNodeUpdateServer() {
	if a.nodeUpdateService == nil {
		log.Fatalln("node update service is nil")
	}
	if a.publisher == nil {
		log.Fatalln("publisher is nil")
	}
	if a.nodeUpdateSubscriber == nil {
		log.Fatalln("node update subscriber is nil")
	}
	server, err := servers.NewNodeUpdateAsyncServer(a.nodeUpdateSubscriber, a.publisher, *a.nodeUpdateService)
	if err != nil {
		log.Fatalln(err)
	}
	a.nodeUpdateServer = server
}

func (a *app) initNodeService() {
	if a.nodeRepo == nil {
		log.Fatalln("node repo is nil")
	}
	if a.meridian == nil {
		log.Fatalln("meridian is nil")
	}
	if a.gravity == nil {
		log.Fatalln("gravity is nil")
	}
	nodeService, err := services.NewNodeService(a.nodeRepo, a.evaluatorClient, a.administratorClient, a.authzService, a.meridian, a.gravity,
		domain.LivenessPolicy{StaleAfter: a.config.StaleAfter(), OfflineAfter: a.config.OfflineAfter()})
	if err != nil {
		log.Fatalln(err)
	}
	a.nodeService = nodeService
}

func (a *app) initLabelService() {
	if a.nodeRepo == nil {
		log.Fatalln("node repo is nil")
	}
	labelService, err := services.NewLabelService(a.nodeRepo, a.evaluatorClient, a.authzService)
	if err != nil {
		log.Fatalln(err)
	}
	a.labelService = labelService
}

func (a *app) initAuthZService() {
	a.authzService = services.NewAuthZService(a.config.TokenKey())
}

func (a *app) initMeridian() {
	conn, err := grpc.NewClient(a.config.MeridianAddress(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalln(err)
	}
	a.meridian = meridian_api.NewMeridianClient(conn)
}

func (a *app) initGravity() {
	log.Println(a.config.GravityAddress())
	conn, err := grpc.NewClient(a.config.GravityAddress(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalln(err)
	}
	a.gravity = gravity_api.NewAgentQueueClient(conn)
}

func (a *app) initEvaluatorClient() {
	client, err := newOortEvaluatorClient(a.config.OortAddress())
	if err != nil {
		log.Fatalln(err)
	}
	a.evaluatorClient = client
}

func (a *app) initAdministratorClient() {
	client, err := newOortAdministratorClient(a.config.NatsAddress())
	if err != nil {
		log.Fatalln(err)
	}
	a.administratorClient = client
}

func (a *app) initNatsPublisher(conn *natsgo.Conn) {
	publisher, err := nats.NewPublisher(conn)
	if err != nil {
		log.Fatalln(err)
	}
	a.publisher = publisher
}

func (a *app) initRegistrationNatsSubscriber(conn *natsgo.Conn) {
	registrationSubscriber, err := nats.NewSubscriber(conn, api.RegistrationSubject, "magnetar")
	if err != nil {
		log.Fatalln(err)
	}
	a.registrationSubscriber = registrationSubscriber
}

func (a *app) initNodeUpdateNatsSubscriber(conn *natsgo.Conn) {
	nodeUpdateSubscriber, err := nats.NewSubscriber(conn, api.NodeUpdateSubject, "magnetar")
	if err != nil {
		log.Fatalln(err)
	}
	a.nodeUpdateSubscriber = nodeUpdateSubscriber
}

func (a *app) initHeartbeatNatsSubscriber(conn *natsgo.Conn) {
	heartbeatSubscriber, err := nats.NewSubscriber(conn, api.NodeHeartbeatSubject, "magnetar")
	if err != nil {
		log.Fatalln(err)
	}
	a.heartbeatSubscriber = heartbeatSubscriber
}

func (a *app) initVmNatsSubscribers(conn *natsgo.Conn) {
	desiredGetSubscriber, err := nats.NewSubscriber(conn, api.VmDesiredGetSubject, "magnetar")
	if err != nil {
		log.Fatalln(err)
	}
	a.vmDesiredGetSubscriber = desiredGetSubscriber

	statusSetSubscriber, err := nats.NewSubscriber(conn, api.VmStatusSetSubject, "magnetar")
	if err != nil {
		log.Fatalln(err)
	}
	a.vmStatusSetSubscriber = statusSetSubscriber

	osDesiredGetSubscriber, err := nats.NewSubscriber(conn, api.OsDesiredGetSubject, "magnetar")
	if err != nil {
		log.Fatalln(err)
	}
	a.osDesiredGetSubscriber = osDesiredGetSubscriber
}

func (a *app) initNodeEtcdRepo(client *etcd.Client) {
	nodeRepo, err := repos.NewNodeEtcdRepo(client, a.nodeMarshaller, a.labelMarshaller)
	if err != nil {
		log.Fatalln(err)
	}
	a.nodeRepo = nodeRepo
}

func (a *app) initLabelProtoMarshaller() {
	a.labelMarshaller = proto.NewProtoLabelMarshaller()
}

func (a *app) initNodeProtoMarshaller() {
	a.nodeMarshaller = proto.NewProtoNodeMarshaller()
}

func (a *app) startRegistrationServer() error {
	err := a.registrationServer.Serve()
	if err != nil {
		return err
	}
	a.gracefulShutdownProcesses = append(a.gracefulShutdownProcesses, func(wg *sync.WaitGroup) {
		a.registrationServer.GracefulStop()
		log.Println("registration server gracefully stopped")
		wg.Done()
	})
	return nil
}

func (a *app) startNodeUpdateServer() error {
	err := a.nodeUpdateServer.Serve()
	if err != nil {
		return err
	}
	a.gracefulShutdownProcesses = append(a.gracefulShutdownProcesses, func(wg *sync.WaitGroup) {
		a.nodeUpdateServer.GracefulStop()
		log.Println("node update server gracefully stopped")
		wg.Done()
	})
	return nil
}

func (a *app) startHeartbeatServer() error {
	if err := a.heartbeatServer.Serve(); err != nil {
		return err
	}
	a.gracefulShutdownProcesses = append(a.gracefulShutdownProcesses, func(wg *sync.WaitGroup) {
		a.heartbeatServer.GracefulStop()
		log.Println("heartbeat server gracefully stopped")
		wg.Done()
	})
	return nil
}

func (a *app) startNodeGC() error {
	if !a.config.GCEnabled() {
		log.Println("node gc disabled (NODE_GC_ENABLED)")
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.shutdownProcesses = append(a.shutdownProcesses, cancel)
	go a.nodeGC.Run(ctx, a.config.GCInterval())
	log.Printf("node gc running every %s (pool records older than %s)", a.config.GCInterval(), a.config.GCPoolAfter())
	return nil
}

func (a *app) startVmAsyncServer() error {
	err := a.vmAsyncServer.Serve()
	if err != nil {
		return err
	}
	a.gracefulShutdownProcesses = append(a.gracefulShutdownProcesses, func(wg *sync.WaitGroup) {
		a.vmAsyncServer.GracefulStop()
		log.Println("vm async server gracefully stopped")
		wg.Done()
	})
	return nil
}

func (a *app) startGrpcServer() error {
	lis, err := net.Listen("tcp", a.config.ServerAddress())
	if err != nil {
		return err
	}
	go func() {
		log.Printf("server listening at %v", lis.Addr())
		if err := a.grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	a.gracefulShutdownProcesses = append(a.gracefulShutdownProcesses, func(wg *sync.WaitGroup) {
		a.grpcServer.GracefulStop()
		log.Println("magnetar server gracefully stopped")
		wg.Done()
	})
	return nil
}

func (a *app) shutdown() {
	for _, shutdownProcess := range a.shutdownProcesses {
		shutdownProcess()
	}
}
