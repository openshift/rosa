package rosa

import (
	"fmt"

	"go.uber.org/mock/gomock"

	goVer "github.com/hashicorp/go-version"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/version"
)

var _ = Describe("VerifyRosaOptions", func() {
	var ctrl *gomock.Controller

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Context("NewVerifyRosaOptions", func() {
		It("should create VerifyRosaOptions successfully", func() {
			versionMock := version.NewMockRosaVersion(ctrl)
			versionMock.EXPECT().IsLatest(gomock.Any()).Return(
				nil, false, fmt.Errorf("failed to check latest version"))

			opts := &VerifyRosaOptions{
				rosaVersion: versionMock,
			}

			err := opts.Verify()
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(Equal(
				"there was a problem verifying if version is latest: failed to check latest version"))

		})

		It("should return no error if version is up to date", func() {
			versionMock := version.NewMockRosaVersion(ctrl)
			versionMock.EXPECT().IsLatest(gomock.Any()).Return(&goVer.Version{}, true, nil).AnyTimes()

			opts := &VerifyRosaOptions{
				rosaVersion: versionMock,
			}

			err := opts.Verify()
			Expect(err).To(BeNil())
		})
	})
})

var _ = Describe("NewVerifyRosaCommand", func() {
	When("NewVerifyRosa succeeds", func() {
		It("should return a valid command", func() {
			cmd := NewVerifyRosaCommand()
			Expect(cmd).ToNot(BeNil())
			Expect(cmd.Use).To(Equal("rosa-client"))
			Expect(cmd.Aliases).To(Equal([]string{"rosa"}))
			Expect(cmd.Short).To(Equal("Verify ROSA client tools"))
			Expect(cmd.Long).To(Equal("Verify that the ROSA client tools is installed and compatible."))
		})
	})
})
