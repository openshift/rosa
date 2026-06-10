package accountroles

import (
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
)

const cmdTestExternalID = "223B9588-36A5-ECA4-BE8D-7C673B77CEC1"

var _ = Describe("validateAccountRolesSTSExternalID", func() {
	It("accepts a valid external-id", func() {
		err := validateAccountRolesSTSExternalID(cmdTestExternalID)
		Expect(err).NotTo(HaveOccurred(), "valid external-id should pass validation")
	})

	It("accepts an empty external-id", func() {
		err := validateAccountRolesSTSExternalID("")
		Expect(err).NotTo(HaveOccurred(), "empty external-id should pass validation")
	})

	It("rejects an invalid external-id", func() {
		err := validateAccountRolesSTSExternalID("x")
		Expect(err).To(HaveOccurred(), "invalid external-id should fail validation")
	})
})
