package kds

import (
	"context"
	"time"

	"github.com/ansel1/merry"
	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/rs/zerolog"
	"github.com/tommy351/kubenvoy/pkg/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func (s *Server) BuildSnapshot(ctx context.Context, sc envoycache.SnapshotCache) error {
	logger := zerolog.Ctx(ctx)
	updatePeriod := time.Second * 2
	informer := s.KubernetesClient.WatchEndpoints(ctx, &kubernetes.WatchEndpointsOptions{
		ResyncPeriod: updatePeriod,
	})

	// Start the informer
	go informer.Run(ctx.Done())

	// Wait until the informer synced
	waitInformerSynced(ctx, informer)

	// Set initial snapshot
	if err := s.setSnapshot(ctx, sc, informer); err != nil {
		return merry.Wrap(err)
	}

	// Rebuild snapshot
	go func() {
		ticker := time.NewTicker(updatePeriod)

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				if err := s.setSnapshot(ctx, sc, informer); err != nil {
					logger.Error().Err(err).Msg("Failed to set the snapshot")
				}
			}
		}
	}()

	return nil
}

func waitInformerSynced(ctx context.Context, informer cache.SharedIndexInformer) {
	for {
		// TODO: Check if context is done
		if informer.HasSynced() {
			return
		}
	}
}

func (s *Server) setSnapshot(ctx context.Context, sc envoycache.SnapshotCache, informer cache.SharedIndexInformer) error {
	var endpoints []envoycache.Resource
	logger := zerolog.Ctx(ctx)
	node := s.Config.Envoy.Node
	version := informer.LastSyncResourceVersion()

	for _, obj := range informer.GetStore().List() {
		ep := obj.(*corev1.Endpoints)

		for _, subset := range ep.Subsets {
			for _, port := range subset.Ports {
				var lbEndpoints []endpoint.LbEndpoint

				for _, addr := range subset.Addresses {
					lbEndpoints = append(lbEndpoints, endpoint.LbEndpoint{
						HostIdentifier: &endpoint.LbEndpoint_Endpoint{
							Endpoint: &endpoint.Endpoint{
								Address: &core.Address{
									Address: &core.Address_SocketAddress{
										SocketAddress: &core.SocketAddress{
											Address: addr.IP,
											PortSpecifier: &core.SocketAddress_PortValue{
												PortValue: uint32(port.Port),
											},
										},
									},
								},
							},
						},
					})
				}

				name := ep.Name

				if port.Name != "" {
					name = name + "__" + port.Name
				}

				endpoints = append(endpoints, &api.ClusterLoadAssignment{
					ClusterName: name,
					Endpoints: []endpoint.LocalityLbEndpoints{
						{
							LbEndpoints: lbEndpoints,
						},
					},
				})
			}
		}
	}

	snapshot := envoycache.NewSnapshot(version, endpoints, nil, nil, nil)

	if err := snapshot.Consistent(); err != nil {
		return merry.Wrap(err)
	}

	if err := sc.SetSnapshot(node, snapshot); err != nil {
		return merry.Wrap(err)
	}

	logger.Debug().
		Str("node", node).
		Str("version", version).
		Msg("Set snapshot")

	return nil
}
