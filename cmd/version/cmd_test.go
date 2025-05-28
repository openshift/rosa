package version

import (
	"fmt"
	"io"
	"os"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/verify/rosa"
	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/version"
)

var _ = Describe("RosaVersionOptions", func() {
	var (
		ctrl       *gomock.Controller
		opts       *RosaVersionOptions
		mockVerify *rosa.MockVerifyRosa
	)

	When("calling NewRosaVersionOptions", func() {
		It("should initialize ROSA Version Options correctly", func() {
			options, err := NewRosaVersionOptions()
			Expect(err).To(BeNil())
			Expect(options).ToNot(BeNil())
			Expect(options.reporter).ToNot(BeNil())
			Expect(options.verifyRosa).ToNot(BeNil())
			Expect(options.args).ToNot(BeNil())
		})
	})

	When("client only is set to false", func() {
		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			mockVerify = rosa.NewMockVerifyRosa(ctrl)

			rpt := reporter.CreateReporter()

			opts = &RosaVersionOptions{
				verifyRosa: mockVerify,
				reporter:   rpt,

				args: &RosaVersionUserOptions{
					clientOnly: false,
				},
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should print version information", func() {
			mockVerify.EXPECT().Verify().Return(nil).Times(1)
			err := opts.Version()
			Expect(err).To(BeNil())
		})

		It("should verify rosa if clientOnly is false", func() {
			mockVerify.EXPECT().Verify().Return(nil).Times(1)
			err := opts.Version()
			Expect(err).To(BeNil())
		})

		It("should return error if verify fails and clientOnly is false", func() {
			mockVerify.EXPECT().Verify().Return(fmt.Errorf("dummy error")).Times(1)
			err := opts.Version()
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring("failed to verify rosa"))
		})
	})

	When("client only is set to true", func() {
		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			mockVerify = rosa.NewMockVerifyRosa(ctrl)

			rpt := reporter.CreateReporter()

			opts = &RosaVersionOptions{
				verifyRosa: mockVerify,
				reporter:   rpt,

				args: &RosaVersionUserOptions{
					clientOnly: true,
				},
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should not verify rosa if clientOnly is true", func() {
			err := opts.Version()
			Expect(err).To(BeNil())
		})
	})

	When("Both clientOnly and verbose are true", func() {
		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			mockVerify = rosa.NewMockVerifyRosa(ctrl)

			rpt := reporter.CreateReporter()

			opts = &RosaVersionOptions{
				verifyRosa: mockVerify,
				reporter:   rpt,

				args: &RosaVersionUserOptions{
					clientOnly: true,
					verbose:    true,
				},
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should print version information correctly", func() {
			// todo move this to a helper func for capturing output of a func
			var stdout []byte
			var err error

			rout, wout, _ := os.Pipe()
			tmpout := os.Stdout
			defer func() {
				os.Stdout = tmpout
			}()
			os.Stdout = wout

			go func() {
				err = opts.Version()
				wout.Close()
			}()
			Expect(err).To(BeNil())
			stdout, _ = io.ReadAll(rout)

			// Verify the outputs
			Expect(string(stdout)).To(ContainSubstring(info.DefaultVersion))

			if opts.args.verbose {
				expectedVerboseInfo := fmt.Sprintf("Information and download locations:\n\t%s\n\t%s\n",
					version.ConsoleLatestFolder, version.DownloadLatestMirrorFolder)
				Expect(string(stdout)).To(ContainSubstring(expectedVerboseInfo))
			}
		})
	})
})

var _ = Describe("NewRosaVersionCommand", func() {
	var (
		cmd *cobra.Command
	)

	BeforeEach(func() {
		cmd = NewRosaVersionCommand()
	})

	It("should return a valid cobra.Command", func() {
		Expect(cmd).NotTo(BeNil())
		Expect(cmd.Use).To(Equal("version"))
		Expect(cmd.Short).To(Equal("Prints the version of the tool"))
		Expect(cmd.Long).To(Equal("Prints the version number of the tool."))
	})

	It("should add client flag", func() {
		clientFlag := cmd.Flag("client")
		Expect(clientFlag).NotTo(BeNil())
		Expect(clientFlag.Name).To(Equal("client"))
		Expect(clientFlag.Shorthand).To(Equal(""))
		Expect(clientFlag.Usage).To(Equal("Client version only (no remote version check)"))
	})

	It("should add verbose flag", func() {
		verboseFlag := cmd.Flag("verbose")
		Expect(verboseFlag).NotTo(BeNil())
		Expect(verboseFlag.Name).To(Equal("verbose"))
		Expect(verboseFlag.Shorthand).To(Equal("v"))
		Expect(verboseFlag.Usage).To(Equal("Display verbose version information, including download locations"))
	})
})
