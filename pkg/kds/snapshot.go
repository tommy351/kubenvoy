package kds

import (
	"context"
	"time"

	"github.com/ansel1/merry"
	"github.com/rs/zerolog"
	"github.com/tommy351/kubenvoy/pkg/envoy"
	"github.com/tommy351/kubenvoy/pkg/k8s"
	"k8s.io/client-go/tools/cache"
)

func (s *Server) BuildSnapshot(ctx context.Context, sc *envoy.Cache) error {
	logger := zerolog.Ctx(ctx)
	watchOpts := k8s.WatchOptions{ResyncPeriod: s.Config.Kubernetes.ResyncPeriod}
	svcInformer := s.KubernetesClient.WatchService(ctx, &k8s.WatchServiceOptions{WatchOptions: watchOpts})
	epInformer := s.KubernetesClient.WatchEndpoints(ctx, &k8s.WatchEndpointsOptions{WatchOptions: watchOpts})

	// Start the informer
	runInformer(ctx, svcInformer)
	runInformer(ctx, epInformer)

	// Set initial snapshot
	if err := s.setSnapshot(ctx, sc, svcInformer, epInformer); err != nil {
		return merry.Wrap(err)
	}

	// Rebuild snapshot
	go func() {
		ticker := time.NewTicker(watchOpts.ResyncPeriod)

		for {
			select {
			case <-ctx.Done():
				return

			case <-ticker.C:
				if err := s.setSnapshot(ctx, sc, svcInformer, epInformer); err != nil {
					logger.Error().Stack().Err(err).Msg("Failed to set the snapshot")
				}
			}
		}
	}()

	return nil
}

func runInformer(ctx context.Context, informer cache.SharedIndexInformer) {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		defer cancel()
		informer.Run(ctx.Done())
	}()

	for {
		if informer.HasSynced() {
			return
		}
	}
}

func (s *Server) setSnapshot(ctx context.Context, sc *envoy.Cache, svcInformer, epInformer cache.SharedIndexInformer) error {
	version := epInformer.LastSyncResourceVersion()

	if !sc.ShouldUpdate(version) {
		return nil
	}

	logger := zerolog.Ctx(ctx)
	node := s.Config.Envoy.Node

	snapshot, err := envoy.NewSnapshot(&envoy.SnapshotOptions{
		Version:   version,
		Endpoints: epInformer.GetStore(),
		Services:  svcInformer.GetStore(),
	})

	if err != nil {
		return merry.Wrap(err)
	}

	if err := sc.UpdateSnapshot(node, version, *snapshot); err != nil {
		return merry.Wrap(err)
	}

	logger.Debug().
		Str("node", node).
		Str("version", version).
		Msg("Set snapshot")

	return nil
}
