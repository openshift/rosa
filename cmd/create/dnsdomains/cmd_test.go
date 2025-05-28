package dnsdomains

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("Create dns domain", func() {
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	Context("createDnsDomain", func() {
		When("Creating DNS domain for API call", func() {
			It("OK: should return dns domain with classic architecture", func() {
				domain, err := createDnsDomain(false)
				Expect(domain.ClusterArch()).To(Equal(cmv1.ClusterArchitectureClassic))
				Expect(err).ToNot(HaveOccurred())
			})
			It("OK: should return dns domain with hcp architecture", func() {
				domain, err := createDnsDomain(true)
				Expect(domain.ClusterArch()).To(Equal(cmv1.ClusterArchitectureHcp))
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
