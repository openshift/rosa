package version

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"go.uber.org/mock/gomock"

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
			rout, wout, pipeErr := os.Pipe()
			Expect(pipeErr).ToNot(HaveOccurred())
			tmpout := os.Stdout
			defer func() {
				os.Stdout = tmpout
			}()
			os.Stdout = wout

			type result struct {
				versionErr error
				closeErr   error
			}
			ch := make(chan result, 1)
			go func() {
				vErr := opts.Version()
				cErr := wout.Close()
				ch <- result{versionErr: vErr, closeErr: cErr}
			}()

			stdout, readErr := io.ReadAll(rout)
			Expect(readErr).ToNot(HaveOccurred())

			res := <-ch
			Expect(res.versionErr).ToNot(HaveOccurred())
			Expect(res.closeErr).ToNot(HaveOccurred())

			// Verify the outputs
			Expect(string(stdout)).To(ContainSubstring(info.DefaultVersion))

			if opts.args.verbose {
				expectedVerboseInfo := fmt.Sprintf("Information and download locations:\n\t%s\n\t%s\n",
					version.ConsoleLatestFolder, version.DownloadLatestMirrorFolder)
				Expect(string(stdout)).To(ContainSubstring(expectedVerboseInfo))
			}
		})
	})

	When("Both clientOnly and build are true and build info is unavailable", func() {
		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			mockVerify = rosa.NewMockVerifyRosa(ctrl)

			rpt := reporter.CreateReporter()

			opts = &RosaVersionOptions{
				verifyRosa:    mockVerify,
				reporter:      rpt,
				readBuildInfo: func() (*debug.BuildInfo, bool) { return nil, false },

				args: &RosaVersionUserOptions{
					clientOnly: true,
					verbose:    false,
					build:      true,
				},
			}
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should print version information correctly", func() {
			rout, wout, pipeErr := os.Pipe()
			Expect(pipeErr).ToNot(HaveOccurred())
			tmpout := os.Stdout
			defer func() {
				os.Stdout = tmpout
			}()
			os.Stdout = wout

			type result struct {
				versionErr error
				closeErr   error
			}
			ch := make(chan result, 1)
			go func() {
				vErr := opts.Version()
				cErr := wout.Close()
				ch <- result{versionErr: vErr, closeErr: cErr}
			}()

			stdout, readErr := io.ReadAll(rout)
			Expect(readErr).ToNot(HaveOccurred())

			res := <-ch
			Expect(res.versionErr).ToNot(HaveOccurred())
			Expect(res.closeErr).ToNot(HaveOccurred())

			// Verify the outputs
			Expect(string(stdout)).To(ContainSubstring(info.DefaultVersion))
			Expect(string(stdout)).To(ContainSubstring(fmt.Sprintf("Git commit: %s", info.Build)))
		})
	})

	When("Both clientOnly and build are true and build info is available", func() {
		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			mockVerify = rosa.NewMockVerifyRosa(ctrl)
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("should print git commit info", func() {
			rpt := reporter.CreateReporter()
			opts = &RosaVersionOptions{
				verifyRosa:    mockVerify,
				reporter:      rpt,
				readBuildInfo: fakeBuildInfo("abc1234"),
				args: &RosaVersionUserOptions{
					clientOnly: true,
					build:      true,
				},
			}

			rout, wout, pipeErr := os.Pipe()
			Expect(pipeErr).ToNot(HaveOccurred())
			tmpout := os.Stdout
			defer func() {
				os.Stdout = tmpout
			}()
			os.Stdout = wout

			type result struct {
				versionErr error
				closeErr   error
			}
			ch := make(chan result, 1)
			go func() {
				vErr := opts.Version()
				cErr := wout.Close()
				ch <- result{versionErr: vErr, closeErr: cErr}
			}()

			stdout, readErr := io.ReadAll(rout)
			Expect(readErr).ToNot(HaveOccurred())

			res := <-ch
			Expect(res.versionErr).ToNot(HaveOccurred())
			Expect(res.closeErr).ToNot(HaveOccurred())

			Expect(string(stdout)).To(ContainSubstring("Git commit: abc1234"))
			Expect(string(stdout)).ToNot(ContainSubstring(fmt.Sprintf("Git commit: %s", info.Build)))
		})

		It("should fall back to info.Build when revision is empty", func() {
			rpt := reporter.CreateReporter()
			opts = &RosaVersionOptions{
				verifyRosa:    mockVerify,
				reporter:      rpt,
				readBuildInfo: fakeBuildInfo(""),
				args: &RosaVersionUserOptions{
					clientOnly: true,
					build:      true,
				},
			}

			rout, wout, pipeErr := os.Pipe()
			Expect(pipeErr).ToNot(HaveOccurred())
			tmpout := os.Stdout
			defer func() {
				os.Stdout = tmpout
			}()
			os.Stdout = wout

			type result struct {
				versionErr error
				closeErr   error
			}
			ch := make(chan result, 1)
			go func() {
				vErr := opts.Version()
				cErr := wout.Close()
				ch <- result{versionErr: vErr, closeErr: cErr}
			}()

			stdout, readErr := io.ReadAll(rout)
			Expect(readErr).ToNot(HaveOccurred())

			res := <-ch
			Expect(res.versionErr).ToNot(HaveOccurred())
			Expect(res.closeErr).ToNot(HaveOccurred())

			Expect(string(stdout)).To(ContainSubstring(fmt.Sprintf("Git commit: %s", info.Build)))
		})
	})
})

func fakeBuildInfo(revision string) func() (*debug.BuildInfo, bool) {
	return func() (*debug.BuildInfo, bool) {
		bi := &debug.BuildInfo{}
		if revision != "" {
			bi.Settings = append(bi.Settings, debug.BuildSetting{
				Key:   "vcs.revision",
				Value: revision,
			})
		}
		return bi, true
	}
}

var _ = Describe("NewRosaVersionCommand", func() {
	var cmd *cobra.Command

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

	It("should add build flag", func() {
		buildFlag := cmd.Flag("build")
		Expect(buildFlag).NotTo(BeNil())
		Expect(buildFlag.Name).To(Equal("build"))
		Expect(buildFlag.Shorthand).To(Equal("b"))
		Expect(buildFlag.Hidden).To(BeTrue())
		Expect(buildFlag.Usage).To(Equal("Display extra build info, primarily the git commit the binary was built from"))
	})
})
