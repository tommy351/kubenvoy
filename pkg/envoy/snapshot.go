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
		domain := annotations["kds.kubenvoy.dev/domains"]

		if domain == "" {
			continue
		}

		for _, subset := range ep.Subsets {
			for _, port := range subset.Ports {
				var lbEndpoints []endpoint.LbEndpoint

				for _, addr := range subset.Addresses {
					lbEndpoints = append(lbEndpoints, endpoint.LbEndpoint{
						HostIdentifier: &endpoint.LbEndpoint_Endpoint{
							Endpoint: &endpoint.Endpoint{
								Address: newSocketAddress(addr.IP, uint32(port.Port)),
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

				clusters = append(clusters, &api.Cluster{
					Name:            name,
					ConnectTimeout:  time.Second,
					LbPolicy:        api.Cluster_ROUND_ROBIN,
					DnsLookupFamily: api.Cluster_V4_ONLY,
					Type:            api.Cluster_EDS,
					EdsClusterConfig: &api.Cluster_EdsClusterConfig{
						EdsConfig: &core.ConfigSource{
							ConfigSourceSpecifier: &core.ConfigSource_Ads{
								Ads: &core.AggregatedConfigSource{},
							},
						},
					},
				})

				routeMap[domain] = append(routeMap[domain], route.Route{
					Match: route.RouteMatch{
						PathSpecifier: &route.RouteMatch_Prefix{
							Prefix: "/",
						},
					},
					Action: &route.Route_Route{
						Route: &route.RouteAction{
							ClusterSpecifier: &route.RouteAction_Cluster{
								Cluster: name,
							},
						},
					},
				})
			}
		}
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
