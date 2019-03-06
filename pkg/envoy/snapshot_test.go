package envoy

import (
	"time"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	envoycache "github.com/envoyproxy/go-control-plane/pkg/cache"
	"github.com/envoyproxy/go-control-plane/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

var _ = Describe("NewSnapshot", func() {
	var (
		version             string
		endpoints, services cache.Store
		snapshot            *envoycache.Snapshot
		err                 error
	)

	addEndpoint := func(ep *corev1.Endpoints, annotations map[string]string) {
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:        ep.Name,
				Namespace:   ep.Namespace,
				Annotations: annotations,
			},
		}

		Expect(endpoints.Add(ep)).NotTo(HaveOccurred())
		Expect(services.Add(svc)).NotTo(HaveOccurred())
	}

	BeforeEach(func() {
		endpoints = cache.NewStore(cache.MetaNamespaceKeyFunc)
		services = cache.NewStore(cache.MetaNamespaceKeyFunc)
		version = ""
	})

	JustBeforeEach(func() {
		snapshot, err = NewSnapshot(&SnapshotOptions{
			Version:   version,
			Endpoints: endpoints,
			Services:  services,
		})

		Expect(err).NotTo(HaveOccurred())
	})

	Describe("given version", func() {
		BeforeEach(func() {
			version = "foo"
		})

		It("should set version to endpoints", func() {
			Expect(snapshot.Endpoints.Version).To(Equal(version))
		})

		It("should set version of clusters", func() {
			Expect(snapshot.Clusters.Version).To(Equal(version))
		})

		It("should set version of routes", func() {
			Expect(snapshot.Routes.Version).To(Equal(version))
		})

		It("should set version of listeners", func() {
			Expect(snapshot.Listeners.Version).To(Equal(version))
		})
	})

	Describe("given a endpoint without domain annotation", func() {
		BeforeEach(func() {
			addEndpoint(&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
			}, map[string]string{})
		})

		It("endpoints should be empty", func() {
			Expect(snapshot.Endpoints.Items).To(BeEmpty())
		})

		It("clusters should be empty", func() {
			Expect(snapshot.Clusters.Items).To(BeEmpty())
		})

		It("routes should be empty", func() {
			Expect(snapshot.Routes.Items).To(BeEmpty())
		})

		It("listeners should be empty", func() {
			Expect(snapshot.Listeners.Items).To(BeEmpty())
		})
	})

	Describe("given a endpoint with domain annotation", func() {
		BeforeEach(func() {
			addEndpoint(&corev1.Endpoints{
				ObjectMeta: metav1.ObjectMeta{
					Name: "foo",
				},
				Subsets: []corev1.EndpointSubset{
					{
						Addresses: []corev1.EndpointAddress{
							{IP: "10.1.1.0"},
						},
						Ports: []corev1.EndpointPort{
							{Port: 80},
						},
					},
				},
			}, map[string]string{
				"kds.kubenvoy.dev/domains": "*",
			})
		})

		It("check endpoints", func() {
			Expect(snapshot.Endpoints.Items).To(Equal(map[string]envoycache.Resource{
				"foo": &api.ClusterLoadAssignment{
					ClusterName: "foo",
					Endpoints: []endpoint.LocalityLbEndpoints{
						{
							LbEndpoints: []endpoint.LbEndpoint{
								{
									HostIdentifier: &endpoint.LbEndpoint_Endpoint{
										Endpoint: &endpoint.Endpoint{
											Address: newSocketAddress("10.1.1.0", 80),
										},
									},
								},
							},
						},
					},
				},
			}))
		})

		It("check clusters", func() {
			Expect(snapshot.Clusters.Items).To(Equal(map[string]envoycache.Resource{
				"foo": &api.Cluster{
					Name:            "foo",
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
				},
			}))
		})

		It("check routes", func() {
			Expect(snapshot.Routes.Items).To(Equal(map[string]envoycache.Resource{
				"kds": &api.RouteConfiguration{
					Name: "kds",
					VirtualHosts: []route.VirtualHost{
						{
							Name:    "*",
							Domains: []string{"*"},
							Routes: []route.Route{
								{
									Match: route.RouteMatch{
										PathSpecifier: &route.RouteMatch_Prefix{
											Prefix: "/",
										},
									},
									Action: &route.Route_Route{
										Route: &route.RouteAction{
											ClusterSpecifier: &route.RouteAction_Cluster{
												Cluster: "foo",
											},
										},
									},
								},
							},
						},
					},
				},
			}))
		})

		It("check listeners", func() {
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
						RouteConfigName: "kds",
					},
				},
				HttpFilters: []*hcm.HttpFilter{
					{
						Name: util.Router,
					},
				},
			})

			Expect(err).NotTo(HaveOccurred())

			Expect(snapshot.Listeners.Items).To(Equal(map[string]envoycache.Resource{
				"kds": &api.Listener{
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
				},
			}))
		})
	})
})
