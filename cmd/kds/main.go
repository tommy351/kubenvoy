package main

import (
	"context"

	"github.com/tommy351/kubenvoy/internal/cmd"
	"github.com/tommy351/kubenvoy/pkg/config"
	"github.com/tommy351/kubenvoy/pkg/k8s"
	"github.com/tommy351/kubenvoy/pkg/kds"
)

func main() {
	conf := config.MustReadConfig()
	ctx := context.Background()
	logger := cmd.NewLogger(&conf.Log)
	kubeClient, err := k8s.NewClient(&conf.Kubernetes)

	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create a Kubernetes client")
	}

	ctx = logger.WithContext(ctx)

	s := &kds.Server{
		Config:           conf,
		KubernetesClient: kubeClient,
	}

	if err := s.Serve(ctx); err != nil {
		logger.Fatal().Stack().Err(err).Msg("Failed to start the server")
	}
}
