package roles

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/ocm"
)

var _ = Describe("Classic managed policies validation", func() {
	DescribeTable("determines when classic managed policies are unsupported",
		func(isManagedSet bool, managedPolicies bool, env string, expectedDecision bool) {
			decision := ClassicManagedPoliciesUnsupportedInEnv(isManagedSet, managedPolicies, env)
			Expect(decision).To(Equal(expectedDecision))
		},
		Entry("rejects when managed policies are enabled in production", true, true, ocm.Production, true),
		Entry("allows explicit false in production", true, false, ocm.Production, false),
		Entry("allows managed policies in non-production", true, true, "staging", false),
		Entry("allows default unmanaged state in production", false, false, ocm.Production, false),
		Entry("does not reject when flag is not set", false, true, ocm.Production, false),
		Entry("allows empty environment", true, true, "", false),
		Entry("does not reject production alias string", true, true, ocm.ProductionAlias, false),
	)
})
