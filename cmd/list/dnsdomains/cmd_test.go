package dnsdomains

import (
	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"

	"github.com/openshift/rosa/tests/utils/constants"
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
		domain1, err := b.ID(domainIds[0]).Build()
		Expect(err).NotTo(HaveOccurred())
		domains = append(domains, domain1)
		domain2, err := b.ID(domainIds[1]).Build()
		Expect(err).NotTo(HaveOccurred())
		domains = append(domains, domain2)
		domain3, err := b.ID(domainIds[2]).Build()
		Expect(err).NotTo(HaveOccurred())
		domains = append(domains, domain3)
		domain4, err := b.ID(domainIds[3]).Build()
		Expect(err).NotTo(HaveOccurred())
		domains = append(domains, domain4)
	})
	AfterEach(func() {
		ctrl.Finish()
		domains = make([]*v1.DNSDomain, 0)
	})

	Context("List DNS domains", func() {
		When("filterByBaseDomain", func() {
			It("OK: should return all DNS domains", func() {
				domains = filterByBaseDomain(domains, constants.ClassicDnsBaseDomain)
				Expect(domains).To(HaveLen(4))
				for i, domain := range domains {
					Expect(domain.ID()).To(Equal(domainIds[i]))
				}
			})
			It("OK: should return only hosted-cp DNS domains", func() {
				domains = filterByBaseDomain(domains, constants.HostedCpDnsBaseDomain)
				Expect(domains).To(HaveLen(2))
				for i, domain := range domains {
					Expect(domain.ID()).To(Equal(hostedCpDomainIds[i]))
				}
			})
		})
		When("returnBaseDomain", func() {
			It("OK: should return correct DNS base domain when not hosted-cp", func() {
				isHostedCp := false
				baseDomain := returnBaseDomain(isHostedCp)
				Expect(baseDomain).To(Equal(constants.ClassicDnsBaseDomain))
			})
			It("OK: should return correct DNS base domain when hosted-cp", func() {
				isHostedCp := true
				baseDomain := returnBaseDomain(isHostedCp)
				Expect(baseDomain).To(Equal(constants.HostedCpDnsBaseDomain))
			})
		})
		When("Combining returnBaseDomain and filterByBaseDomain", func() {
			It("OK: should return all DNS domains", func() {
				isHostedCp := false
				domains = filterByBaseDomain(domains, returnBaseDomain(isHostedCp))
				Expect(domains).To(HaveLen(4))
				for i, domain := range domains {
					Expect(domain.ID()).To(Equal(domainIds[i]))
				}
			})
			It("OK: should return only hosted-cp DNS domains", func() {
				isHostedCp := true
				domains = filterByBaseDomain(domains, returnBaseDomain(isHostedCp))
				Expect(domains).To(HaveLen(2))
				for i, domain := range domains {
					Expect(domain.ID()).To(Equal(hostedCpDomainIds[i]))
				}
			})
		})
	})
})
