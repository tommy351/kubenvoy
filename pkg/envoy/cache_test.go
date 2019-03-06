package envoy

import (
	"context"

	api "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cache", func() {
	Describe("ShouldUpdate", func() {
		cache := NewCache(context.Background())
		cache.lastVersion = "1"

		It("should return true when version changed", func() {
			Expect(cache.ShouldUpdate("2")).To(BeTrue())
		})

		It("should return false when version unchanged", func() {
			Expect(cache.ShouldUpdate("1")).To(BeFalse())
		})
	})

	Describe("UpdateSnapshot", func() {
		var c *Cache
		node := "test-node"
		version := "test"

		BeforeEach(func() {
			c = NewCache(context.Background())
			Expect(c.UpdateSnapshot(node, version, cache.Snapshot{
				Listeners: cache.Resources{Version: version},
			})).NotTo(HaveOccurred())
		})

		It("should update version", func() {
			Expect(c.lastVersion).To(Equal(version))
		})

		It("should update snapshot", func() {
			res, err := c.Fetch(context.Background(), api.DiscoveryRequest{
				Node:    &core.Node{Id: node},
				TypeUrl: cache.ListenerType,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Version).To(Equal(version))
		})
	})
})
