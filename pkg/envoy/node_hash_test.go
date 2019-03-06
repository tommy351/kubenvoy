package envoy

import (
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("NodeHash", func() {
	DescribeTable("ID", func(expected string, node *core.Node) {
		hash := NodeHash{}
		Expect(hash.ID(node)).To(Equal(expected))
	},
		Entry("unknown", "unknown", nil),
		Entry("id", "foo", &core.Node{Id: "foo"}),
	)
})
