package reporter

import (
	"io"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/openshift/rosa/pkg/color"
	"github.com/openshift/rosa/pkg/debug"
)

func TestReporter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "reporter testing")
}

var _ = Describe("Reporter Tests", func() {

	var reporter *Object

	BeforeEach(func() {
		reporter = CreateReporter()
	})

	AfterEach(func() {
		color.SetColor("auto")
		debug.SetEnabled(false)
	})

	It("Returns a reporter", func() {
		reporter := CreateReporter()
		Expect(reporter).NotTo(BeNil())
	})

	Context("Info", func() {
		It("Prints an info message without color", func() {
			color.SetColor("never")
			Expect(color.UseColor()).To(BeFalse())

			stdOut, stdErr := captureStdOutAndStdError(func() {
				reporter.Infof("Hello World")
			})

			Expect(stdOut).To(Equal(infoPrefix + "Hello World\n"))
			Expect(stdErr).To(BeEmpty())
		})

		It("Prints an info message with color", func() {
			color.SetColor("always")
			Expect(color.UseColor()).To(BeTrue())

			stdOut, stdErr := captureStdOutAndStdError(func() {
				reporter.Infof("Hello World")
			})

			Expect(stdOut).To(Equal(infoColorPrefix + "Hello World\n"))
			Expect(stdErr).To(BeEmpty())
		})
	})

	Context("Warn", func() {
		It("Prints a warn message without color", func() {
			color.SetColor("never")
			Expect(color.UseColor()).To(BeFalse())

			stdOut, stdErr := captureStdOutAndStdError(func() {
				reporter.Warnf("Hello World")
			})
			Expect(stdErr).To(Equal(warnPrefix + "Hello World\n"))
			Expect(stdOut).To(BeEmpty())
		})

		It("Prints a warn message with color", func() {
			color.SetColor("always")
			Expect(color.UseColor()).To(BeTrue())

			stdOut, stdErr := captureStdOutAndStdError(func() {
				reporter.Warnf("Hello World")
			})
			Expect(stdErr).To(Equal(warnColorPrefix + "Hello World\n"))
			Expect(stdOut).To(BeEmpty())
		})
	})

	Context("Error", func() {
		It("Prints an error message without color", func() {
			color.SetColor("never")
			Expect(color.UseColor()).To(BeFalse())

			stdOut, stdErr := captureStdOutAndStdError(func() {
				reporter.Errorf("Hello World")
			})
			Expect(stdErr).To(Equal(errorPrefix + "Hello World\n"))
			Expect(stdOut).To(BeEmpty())
		})

		It("Prints an error message with color", func() {
			color.SetColor("always")
			Expect(color.UseColor()).To(BeTrue())

			stdOut, stdErr := captureStdOutAndStdError(func() {
				reporter.Errorf("Hello World")
			})
			Expect(stdErr).To(Equal(errorColorPrefix + "Hello World\n"))
			Expect(stdOut).To(BeEmpty())
		})
	})

	Context("Debug", func() {
		It("Does not print if debug is not enabled", func() {
			debug.SetEnabled(false)
			Expect(debug.Enabled()).To(BeFalse())

			stdOut, stdErr := captureStdOutAndStdError(func() {
				reporter.Debugf("Hello World")
			})
			reporter.Debugf("Hello world")
			Expect(stdOut).To(BeEmpty())
			Expect(stdErr).To(BeEmpty())
		})

		It("Prints a debug message without color", func() {
			debug.SetEnabled(true)
			Expect(debug.Enabled()).To(BeTrue())
			color.SetColor("never")
			Expect(color.UseColor()).To(BeFalse())

			stdOut, stdErr := captureStdOutAndStdError(func() {
				reporter.Debugf("Hello World")
			})

			Expect(stdOut).To(Equal(infoPrefix + "Hello World\n"))
			Expect(stdErr).To(BeEmpty())
		})

		It("Prints a debug message with color", func() {
			debug.SetEnabled(true)
			Expect(debug.Enabled()).To(BeTrue())
			color.SetColor("always")
			Expect(color.UseColor()).To(BeTrue())

			stdOut, stdErr := captureStdOutAndStdError(func() {
				reporter.Debugf("Hello World")
			})
			Expect(stdOut).To(Equal(infoColorPrefix + "Hello World\n"))
			Expect(stdErr).To(BeEmpty())
		})
	})
})

func captureStdOutAndStdError(function func()) (string, string) {
	var stdout []byte
	var stderr []byte

	rout, wout, _ := os.Pipe()
	oldOut := os.Stdout
	rerr, werr, _ := os.Pipe()
	oldErr := os.Stderr
	defer func() {
		os.Stdout = oldOut
		os.Stderr = oldErr
	}()
	os.Stdout = wout
	os.Stderr = werr

	go func() {
		function()
		wout.Close()
		werr.Close()
	}()
	stdout, _ = io.ReadAll(rout)
	stderr, _ = io.ReadAll(rerr)

	return string(stdout), string(stderr)
}
