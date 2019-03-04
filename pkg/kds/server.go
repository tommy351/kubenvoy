package kds

import (
	"context"
	"net"

	"github.com/ansel1/merry"
	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
	"github.com/rs/zerolog"
	"github.com/tommy351/kubenvoy/pkg/config"
	"github.com/tommy351/kubenvoy/pkg/k8s"
	"google.golang.org/grpc"
)

type Server struct {
	Config           *config.Config
	KubernetesClient k8s.Client
}

func (s *Server) Serve(ctx context.Context) (err error) {
	logger := zerolog.Ctx(ctx)
	ln, err := net.Listen("tcp", s.Config.Server.Address)

	if err != nil {
		return merry.Wrap(err)
	}

	sc := cache.NewSnapshotCache(true, NodeHash{}, NewLogger(logger))

	if err := s.BuildSnapshot(ctx, sc); err != nil {
		return merry.Wrap(err)
	}

	server := xds.NewServer(sc, s)
	grpcServer := grpc.NewServer()

	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	api.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	api.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	api.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	api.RegisterListenerDiscoveryServiceServer(grpcServer, server)

	go func() {
		logger.Info().Str("addr", ln.Addr().String()).Msg("Starting server")
		err = merry.Wrap(grpcServer.Serve(ln))
	}()

	<-ctx.Done()
	grpcServer.GracefulStop()

	return
}
