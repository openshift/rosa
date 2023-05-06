package accountroles

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Delete account roles", func() {
	It("Deletes all account-roles if the user didn't specify topology", func() {
		deleteClassic, deleteHostedCP := setDeleteRoles(false, false)
		Expect(deleteClassic).To(Equal(true))
		Expect(deleteHostedCP).To(Equal(true))
	})
	It("Deletes only hosted CP account-roles if the user selected '--hosted-cp'", func() {
		deleteClassic, deleteHostedCP := setDeleteRoles(false, true)
		Expect(deleteClassic).To(Equal(false))
		Expect(deleteHostedCP).To(Equal(true))
	})
	It("Deletes only classic account-roles if the user selected '--classic'", func() {
		deleteClassic, deleteHostedCP := setDeleteRoles(true, false)
		Expect(deleteClassic).To(Equal(true))
		Expect(deleteHostedCP).To(Equal(false))
	})
	It("Deletes all account-roles if the user selected both '--classic' and '--hosted-cp'", func() {
		deleteClassic, deleteHostedCP := setDeleteRoles(true, true)
		Expect(deleteClassic).To(Equal(true))
		Expect(deleteHostedCP).To(Equal(true))
	})
})
