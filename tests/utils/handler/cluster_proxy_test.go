package handler

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/aws"
)

var _ = Describe("public cluster proxy bypass", func() {
	DescribeTable("adds the cluster domain to a valid no-proxy list",
		func(current, expected string) {
			result := AddClusterDomainToNoProxy(current, "hf-proxy", "abc.openshiftapps.com")
			Expect(result).To(Equal(expected))
			Expect(aws.UserNoProxyValidator(result)).To(Succeed())
			Expect(aws.UserNoProxyDuplicateValidator(result)).To(Succeed())
		},
		Entry("missing no-proxy value", "", ".hf-proxy.abc.openshiftapps.com"),
		Entry("existing destinations", "10.0.0.0/16,.example.com",
			"10.0.0.0/16,.example.com,.hf-proxy.abc.openshiftapps.com"),
		Entry("domain already present", "10.0.0.0/16,.hf-proxy.abc.openshiftapps.com",
			"10.0.0.0/16,.hf-proxy.abc.openshiftapps.com"),
		Entry("empty entries in the existing list", ",10.0.0.0/16,",
			"10.0.0.0/16,.hf-proxy.abc.openshiftapps.com"),
	)
})
