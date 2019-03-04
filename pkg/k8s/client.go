package k8s

import (
	"context"
	"time"

	"github.com/ansel1/merry"
	"github.com/tommy351/kubenvoy/pkg/config"
	"k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	// Load auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type Client interface {
	WatchEndpoints(ctx context.Context, opts *WatchEndpointsOptions) cache.SharedIndexInformer
	WatchService(ctx context.Context, opts *WatchServiceOptions) cache.SharedIndexInformer
}

type WatchOptions struct {
	ResyncPeriod time.Duration
}

type ListEndpointsOptions struct{}

type WatchEndpointsOptions struct {
	ListEndpointsOptions
	WatchOptions
}

type ListServiceOptions struct{}

type WatchServiceOptions struct {
	ListEndpointsOptions
	WatchOptions
}

type client struct {
	config *config.KubernetesConfig
	client kubernetes.Interface
}

func NewClient(conf *config.KubernetesConfig) (Client, error) {
	restConf, err := LoadConfig()

	if err != nil {
		return nil, merry.Wrap(err)
	}

	kubeClient, err := kubernetes.NewForConfig(restConf)

	if err != nil {
		return nil, merry.Wrap(err)
	}

	return &client{
		config: conf,
		client: kubeClient,
	}, nil
}

func (c *client) WatchEndpoints(ctx context.Context, opts *WatchEndpointsOptions) cache.SharedIndexInformer {
	return v1.NewEndpointsInformer(c.client, c.config.Namespace, opts.ResyncPeriod, cache.Indexers{})
}

func (c *client) WatchService(ctx context.Context, opts *WatchServiceOptions) cache.SharedIndexInformer {
	return v1.NewServiceInformer(c.client, c.config.Namespace, opts.ResyncPeriod, cache.Indexers{})
}
