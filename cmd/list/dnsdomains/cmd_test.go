package dnsdomains

import (
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("List dns domains", func() {
	var ctrl *gomock.Controller

	var domains = make([]*v1.DNSDomain, 0)

	var hostedCpDomainIds = []string{"1234.i3.devshift.org", "4321.i3.devshift.org"}
	var domainIds = []string{"1234.i1.devshift.org", hostedCpDomainIds[0], hostedCpDomainIds[1],
		"4321.i1.devshift.org"}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		b := v1.DNSDomainBuilder{}
		domain1, err := b.ID(domainIds[0]).ClusterArch(v1.ClusterArchitectureClassic).Build()
		Expect(err).NotTo(HaveOccurred())
		domains = append(domains, domain1)
		domain2, err := b.ID(domainIds[1]).ClusterArch(v1.ClusterArchitectureHcp).Build()
		Expect(err).NotTo(HaveOccurred())
		domains = append(domains, domain2)
		domain3, err := b.ID(domainIds[2]).ClusterArch(v1.ClusterArchitectureHcp).Build()
		Expect(err).NotTo(HaveOccurred())
		domains = append(domains, domain3)
		domain4, err := b.ID(domainIds[3]).ClusterArch(v1.ClusterArchitectureClassic).Build()
		Expect(err).NotTo(HaveOccurred())
		domains = append(domains, domain4)
	})
	AfterEach(func() {
		ctrl.Finish()
		domains = make([]*v1.DNSDomain, 0)
	})

	Context("List DNS domains", func() {
		When("filterByBaseDomain", func() {
			It("OK: should return only hosted-cp DNS domains", func() {
				filtered := filterByClusterArch(domains, v1.ClusterArchitectureHcp)
				Expect(filtered).To(HaveLen(2))
				for i, domain := range filtered {
					Expect(domain.ID()).To(Equal(hostedCpDomainIds[i]))
				}
			})
			It("OK: should return only classic DNS domains", func() {
				filtered := filterByClusterArch(domains, v1.ClusterArchitectureClassic)
				Expect(filtered).To(HaveLen(2))
				for i, domain := range filtered {
					Expect(domain.ID()).ToNot(Equal(hostedCpDomainIds[i]))
				}
			})
		})
		When("Combining returnBaseDomain and filterByBaseDomain", func() {
			It("OK: should return only hosted-cp DNS domains", func() {
				filtered := filterByClusterArch(domains, v1.ClusterArchitectureHcp)
				Expect(filtered).To(HaveLen(2))
				for i, domain := range filtered {
					Expect(domain.ID()).To(Equal(hostedCpDomainIds[i]))
				}
			})
		})
	})
})
