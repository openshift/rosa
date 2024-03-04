package version

import (
	"fmt"
	"io"
	"os"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/cmd/verify/rosa"
	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/pkg/reporter"
)

var _ = Describe("RosaVersionOptions", func() {
	var (
		ctrl       *gomock.Controller
		opts       *RosaVersionOptions
		mockVerify *rosa.MockVerifyRosa
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockVerify = rosa.NewMockVerifyRosa(ctrl)

		rpt, err := reporter.CreateReporter()
		if err != nil {
			Expect(err).To(BeNil())
		}

		opts = &RosaVersionOptions{
			verifyRosa: mockVerify,
			reporter:   rpt,
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
		args.clientOnly = false
		err := opts.Version()
		Expect(err).To(BeNil())
	})

	It("should return error if verify fails and clientOnly is false", func() {
		mockVerify.EXPECT().Verify().Return(fmt.Errorf("dummy error")).Times(1)
		args.clientOnly = false
		err := opts.Version()
		Expect(err).ToNot(BeNil())
		Expect(err.Error()).To(ContainSubstring("failed to verify rosa"))
	})

	It("should not verify rosa if clientOnly is true", func() {
		args.clientOnly = true
		err := opts.Version()
		Expect(err).To(BeNil())
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

		args.verbose = true
		args.clientOnly = true
		go func() {
			err = opts.Version()
			wout.Close()
		}()
		Expect(err).To(BeNil())
		stdout, _ = io.ReadAll(rout)

		// Verify the outputs
		expectedVersionInfo := fmt.Sprintf("%s (Build: %s)", info.Version, info.Build)
		Expect(string(stdout)).To(ContainSubstring(expectedVersionInfo))

		if args.verbose {
			expectedVerboseInfo := fmt.Sprintf("Information and download locations:\n\t%s\n\t%s\n",
				rosa.ConsoleLatestFolder, rosa.DownloadLatestMirrorFolder)
			Expect(string(stdout)).To(ContainSubstring(expectedVerboseInfo))
		}
	})
})

var _ = Describe("NewRosaVersionCmd", func() {
	var (
		cmd *cobra.Command
		err error
	)

	BeforeEach(func() {
		cmd, err = NewRosaVersionCmd()
		Expect(err).To(BeNil())
		Expect(err).To(BeNil())
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
})
