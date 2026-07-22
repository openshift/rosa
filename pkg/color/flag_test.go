package color

import (
	"errors"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

func TestColor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Color Suite")
}

type stubFileInfo struct {
	mode os.FileMode
}

func (s stubFileInfo) Name() string       { return "stdout" }
func (s stubFileInfo) Size() int64        { return 0 }
func (s stubFileInfo) Mode() os.FileMode  { return s.mode }
func (s stubFileInfo) ModTime() time.Time { return time.Time{} }
func (s stubFileInfo) IsDir() bool        { return false }
func (s stubFileInfo) Sys() interface{}   { return nil }

var _ = Describe("Color", func() {
	var previousColor string
	var previousGOOS string
	var previousStdoutStat func() (os.FileInfo, error)

	BeforeEach(func() {
		previousColor = color
		previousGOOS = runtimeGOOS
		previousStdoutStat = stdoutStat
		color = ""
	})

	AfterEach(func() {
		color = previousColor
		runtimeGOOS = previousGOOS
		stdoutStat = previousStdoutStat
	})

	It("registers the color flag with auto as default", func() {
		cmd := &cobra.Command{Use: "test"}

		AddFlag(cmd)

		flag := cmd.PersistentFlags().Lookup("color")
		Expect(flag).NotTo(BeNil())
		Expect(flag.DefValue).To(Equal("auto"))
	})

	It("returns the supported completion options", func() {
		values, directive := completion(&cobra.Command{Use: "test"}, nil, "")

		Expect(values).To(Equal([]string{"auto", "never", "always"}))
		Expect(directive).To(Equal(cobra.ShellCompDirectiveDefault))
	})

	It("disables color when set to never", func() {
		SetColor("never")
		Expect(UseColor()).To(BeFalse())
	})

	It("enables color when set to always", func() {
		SetColor("always")
		Expect(UseColor()).To(BeTrue())
	})

	It("treats unknown values the same as auto", func() {
		SetColor("auto")
		expected := UseColor()

		SetColor("unexpected")

		Expect(UseColor()).To(Equal(expected))
	})

	It("disables color for auto mode on Windows", func() {
		SetColor("auto")
		runtimeGOOS = "windows"
		stdoutStat = func() (os.FileInfo, error) {
			return stubFileInfo{mode: os.ModeDevice}, nil
		}

		Expect(UseColor()).To(BeFalse())
	})

	It("enables color for auto mode when stdout is a terminal", func() {
		SetColor("auto")
		runtimeGOOS = "linux"
		stdoutStat = func() (os.FileInfo, error) {
			return stubFileInfo{mode: os.ModeDevice}, nil
		}

		Expect(UseColor()).To(BeTrue())
	})

	It("disables color for auto mode when stdout is a named pipe", func() {
		SetColor("auto")
		runtimeGOOS = "linux"
		stdoutStat = func() (os.FileInfo, error) {
			return stubFileInfo{mode: os.ModeNamedPipe}, nil
		}

		Expect(UseColor()).To(BeFalse())
	})

	It("enables color for auto mode when stdout stat fails", func() {
		SetColor("auto")
		runtimeGOOS = "linux"
		stdoutStat = func() (os.FileInfo, error) {
			return nil, errors.New("stat failed")
		}

		Expect(UseColor()).To(BeTrue())
	})
})
