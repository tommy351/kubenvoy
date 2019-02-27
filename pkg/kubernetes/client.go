package kubernetes

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
}

type ListEndpointsOptions struct{}

type WatchEndpointsOptions struct {
	ListEndpointsOptions

	ResyncPeriod time.Duration
	OnAdd        func(obj interface{})
	OnUpdate     func(old, new interface{})
	OnDelete     func(obj interface{})
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
