package rosacli

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nodepool lookup", func() {
	It("resolves HyperFleet nodepools by name or UID", func() {
		pool := &NodePool{ID: "pool-uid", Name: "workers"}
		list := NodePoolList{NodePools: []*NodePool{pool}}
		Expect(list.Nodepool("workers")).To(BeIdenticalTo(pool))
		Expect(list.Nodepool("pool-uid")).To(BeIdenticalTo(pool))
		Expect(list.Nodepool("missing")).To(BeNil())
	})
	It("retains OCM lookup by ID without a name column", func() {
		pool := &NodePool{ID: "workers"}
		list := NodePoolList{NodePools: []*NodePool{pool}}
		Expect(list.Nodepool("workers")).To(BeIdenticalTo(pool))
		Expect(list.Nodepool("")).To(BeNil())
	})
})
