package idp_test

import (
	gomock "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/cmd/create/idp"
	"github.com/openshift/rosa/cmd/create/idp/mocks"
)

var _ = Describe("Cmd", func() {
	var (
		mockCtrl *gomock.Controller
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})
	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("GenerateIdpName", func() {

		var (
			idpType string
			idps    []idp.IdentityProvider
		)
		BeforeEach(func() {
			idpType = "github"
			idps = []idp.IdentityProvider{}
		})
		Context("when no IDP exists", func() {
			It("generates a idp name name-1", func() {
				name := idp.GenerateIdpName(idpType, idps)
				Expect(name).To(Equal(idpType + "-1"))
			})
		})

		Context("when an IDP with the name of the type already exists", func() {
			BeforeEach(func() {
				mockIdp := mocks.NewMockIdentityProvider(mockCtrl)
				mockIdp.EXPECT().Name().Return("github").AnyTimes()
				idps = append(idps, mockIdp)
			})
			It("generates a unique idp name", func() {
				name := idp.GenerateIdpName(idpType, idps)
				expectUnique(name, idps)
			})
		})

		Context("when an IDP with a generated name already exists", func() {
			BeforeEach(func() {
				mockIdp := mocks.NewMockIdentityProvider(mockCtrl)
				mockIdp.EXPECT().Name().Return(idp.GenerateIdpName(idpType, idps)).AnyTimes()
				idps = append(idps, mockIdp)
			})
			It("generates a unique idp name", func() {
				name := idp.GenerateIdpName(idpType, idps)
				expectUnique(name, idps)
			})
		})
	})

	DescribeTable("Validate Idp Name",
		func(nameVal interface{}, errorExcepted bool) {
			err := idp.ValidateIdpName(nameVal)

			if errorExcepted {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}
		},
		Entry("Type not string",
			1234, true),
		Entry("Invalid identifier",
			"///", true),
		Entry("Reserved name cluster-admin",
			"cluster-admin", true),
		Entry("Valid name",
			"awesometeam", false),
	)
})

func expectUnique(name string, idps []idp.IdentityProvider) {
	for _, idp := range idps {
		Expect(name).NotTo(Equal(idp.Name()))
	}
}
