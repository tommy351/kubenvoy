package kds

import (
	"context"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
)

func (s *Server) OnFetchRequest(context.Context, *api.DiscoveryRequest) error {
	return nil
}

func (s *Server) OnFetchResponse(*api.DiscoveryRequest, *api.DiscoveryResponse) {
}

func (s *Server) OnStreamClosed(int64) {
}

func (s *Server) OnStreamOpen(context.Context, int64, string) error {
	return nil
}

func (s *Server) OnStreamRequest(int64, *api.DiscoveryRequest) {
}

func (s *Server) OnStreamResponse(int64, *api.DiscoveryRequest, *api.DiscoveryResponse) {
}
