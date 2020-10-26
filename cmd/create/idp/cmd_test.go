package idp_test

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/openshift/moactl/cmd/create/idp"
	"github.com/openshift/moactl/cmd/create/idp/mocks"
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
			idps    []IdentityProvider
		)
		BeforeEach(func() {
			idpType = "github"
			idps = []IdentityProvider{}
		})
		Context("when no IDP exists", func() {
			It("generates a idp name name-1", func() {
				name := GenerateIdpName(idpType, idps)
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
				name := GenerateIdpName(idpType, idps)
				expectUnique(name, idps)
			})
		})

		Context("when an IDP with a generated name already exists", func() {
			BeforeEach(func() {
				mockIdp := mocks.NewMockIdentityProvider(mockCtrl)
				mockIdp.EXPECT().Name().Return(GenerateIdpName(idpType, idps)).AnyTimes()
				idps = append(idps, mockIdp)
			})
			It("generates a unique idp name", func() {
				name := GenerateIdpName(idpType, idps)
				expectUnique(name, idps)
			})
		})
	})
})

func expectUnique(name string, idps []IdentityProvider) {
	for _, idp := range idps {
		Expect(name).NotTo(Equal(idp.Name()))
	}
}
