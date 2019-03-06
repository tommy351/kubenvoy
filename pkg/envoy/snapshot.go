package envoy

import (
	"time"

	"github.com/ansel1/merry"
	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/envoyproxy/go-control-plane/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	AnnotationDomains        = "kds.kubenvoy.dev/domains"
	AnnotationPort           = "kds.kubenvoy.dev/port"
	AnnotationConnectTimeout = "kds.kubenvoy.dev/connect_timeout"
	AnnotationLbPolicy       = "kds.kubenvoy.dev/lb_policy"

	DefaultConnectTimeout = time.Second
)

var (
	ErrEmptyEndpointSubset = merry.New("subset of endpoint is empty")
	ErrNoPort              = merry.New("cannot find a port")
)

type SnapshotOptions struct {
	Version   string
	Endpoints cache.Store
	Services  cache.Store
}

func NewSnapshot(options *SnapshotOptions) (*envoycache.Snapshot, error) {
	var (
		clusters, endpoints, routes, listeners []envoycache.Resource
		vhosts                                 []route.VirtualHost
	)

	routeMap := map[string][]route.Route{}
	svcMap := map[string]*corev1.Service{}

	for _, obj := range options.Services.List() {
		if svc, ok := obj.(*corev1.Service); ok {
			svcMap[svc.Name] = svc
		}
	}

	for _, obj := range options.Endpoints.List() {
		ep, ok := obj.(*corev1.Endpoints)

		if !ok {
			continue
		}

		svc, ok := svcMap[ep.Name]

		if !ok {
			continue
		}

		annotations := svc.Annotations
		domain := annotations[AnnotationDomains]

		if domain == "" {
			continue
		}

		if cla, err := newClusterLoadAssignment(svc, ep); err == nil {
			endpoints = append(endpoints, cla)
		} else {
			return nil, merry.Wrap(err)
		}

		if cluster, err := newCluster(svc, ep); err == nil {
			clusters = append(clusters, cluster)
		} else {
			return nil, merry.Wrap(err)
		}

		routeMap[domain] = append(routeMap[domain], *newRoute(ep))
	}

	if len(routeMap) > 0 {
		for domain, routes := range routeMap {
			vhosts = append(vhosts, route.VirtualHost{
				Name:    domain,
				Domains: []string{domain},
				Routes:  routes,
			})
		}

		routeConf := &api.RouteConfiguration{
			Name:         "kds",
			VirtualHosts: vhosts,
		}

		routes = append(routes, routeConf)
		hcmConfig, err := util.MessageToStruct(&hcm.HttpConnectionManager{
			CodecType:  hcm.AUTO,
			StatPrefix: "http",
			RouteSpecifier: &hcm.HttpConnectionManager_Rds{
				Rds: &hcm.Rds{
					ConfigSource: core.ConfigSource{
						ConfigSourceSpecifier: &core.ConfigSource_Ads{
							Ads: &core.AggregatedConfigSource{},
						},
					},
					RouteConfigName: routeConf.Name,
				},
			},
			HttpFilters: []*hcm.HttpFilter{
				{
					Name: util.Router,
				},
			},
		})

		if err != nil {
			return nil, merry.Wrap(err)
		}

		listeners = append(listeners, &api.Listener{
			Name:    "kds",
			Address: *newSocketAddress("0.0.0.0", 10000),
			FilterChains: []listener.FilterChain{
				{
					Filters: []listener.Filter{
						{
							Name: util.HTTPConnectionManager,
							ConfigType: &listener.Filter_Config{
								Config: hcmConfig,
							},
						},
					},
				},
			},
		})
	}

	snapshot := envoycache.NewSnapshot(options.Version, endpoints, clusters, routes, listeners)

	if err := snapshot.Consistent(); err != nil {
		return nil, merry.Wrap(err)
	}

	return &snapshot, nil
}

func getPortByName(ports []corev1.EndpointPort, name string) *corev1.EndpointPort {
	if len(ports) == 0 {
		return nil
	}

	for _, p := range ports {
		p := p

		if p.Name == name {
			return &p
		}
	}

	return &ports[0]
}

func newClusterLoadAssignment(svc *corev1.Service, ep *corev1.Endpoints) (*api.ClusterLoadAssignment, error) {
	if len(ep.Subsets) == 0 {
		return nil, ErrEmptyEndpointSubset.Here().WithValue("service", svc.Name)
	}

	subset := ep.Subsets[0]
	port := getPortByName(subset.Ports, svc.Annotations[AnnotationPort])

	if port == nil {
		return nil, ErrNoPort.Here().WithValue("service", svc.Name)
	}

	lbEndpoints := make([]endpoint.LbEndpoint, len(subset.Addresses))

	for i, addr := range subset.Addresses {
		lbEndpoints[i] = endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: newSocketAddress(addr.IP, uint32(port.Port)),
				},
			},
		}
	}

	return &api.ClusterLoadAssignment{
		ClusterName: ep.Name,
		Endpoints: []endpoint.LocalityLbEndpoints{
			{LbEndpoints: lbEndpoints},
		},
	}, nil
}

func newCluster(svc *corev1.Service, ep *corev1.Endpoints) (*api.Cluster, error) {
	cluster := &api.Cluster{
		Name:            ep.Name,
		ConnectTimeout:  DefaultConnectTimeout,
		DnsLookupFamily: api.Cluster_V4_ONLY,
		Type:            api.Cluster_EDS,
		EdsClusterConfig: &api.Cluster_EdsClusterConfig{
			EdsConfig: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_Ads{
					Ads: &core.AggregatedConfigSource{},
				},
			},
		},
	}

	if s, ok := svc.Annotations[AnnotationConnectTimeout]; ok {
		if timeout, err := time.ParseDuration(s); err == nil {
			cluster.ConnectTimeout = timeout
		} else {
			return nil, merry.Wrap(err)
		}
	}

	if s, ok := svc.Annotations[AnnotationLbPolicy]; ok {
		if v, ok := api.Cluster_LbPolicy_value[s]; ok {
			cluster.LbPolicy = api.Cluster_LbPolicy(v)
		}
	}

	return cluster, nil
}

func newRoute(ep *corev1.Endpoints) *route.Route {
	return &route.Route{
		Match: route.RouteMatch{
			PathSpecifier: &route.RouteMatch_Prefix{
				Prefix: "/",
			},
		},
		Action: &route.Route_Route{
			Route: &route.RouteAction{
				ClusterSpecifier: &route.RouteAction_Cluster{
					Cluster: ep.Name,
				},
			},
		},
	}
}

func newSocketAddress(ip string, port uint32) *core.Address {
	return &core.Address{
		Address: &core.Address_SocketAddress{
			SocketAddress: &core.SocketAddress{
				Address: ip,
				PortSpecifier: &core.SocketAddress_PortValue{
					PortValue: port,
				},
			},
		},
	}
}
