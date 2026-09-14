// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package output

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("Output flag", func() {

	BeforeEach(func() {
		SetOutput("")
	})

	AfterEach(func() {
		SetOutput("")
	})

	It("Adds flag to command", func() {

		cmd := &cobra.Command{}
		Expect(cmd.Flag(FLAG_NAME)).To(BeNil())

		AddFlag(cmd)

		flag := cmd.Flag(FLAG_NAME)
		Expect(flag).NotTo(BeNil())
		Expect(flag.Name).To(Equal(FLAG_NAME))
		Expect(flag.Shorthand).To(Equal(FLAG_SHORTHAND))
		Expect(flag.Value.String()).To(Equal(""))
		Expect(flag.Usage).To(Equal("Output format. Allowed formats are [json yaml]"))
	})

	It("Has a completion function", func() {
		args, directive := completion(nil, nil, "")
		Expect(len(args)).To(Equal(2))
		Expect(args).To(ContainElements(JSON, YAML))

		Expect(directive).To(Equal(cobra.ShellCompDirectiveDefault))
	})

	It("Has flag", func() {
		Expect(HasFlag()).To(BeFalse())
		SetOutput(JSON)
		Expect(HasFlag()).To(BeTrue())
		Expect(Output()).To(Equal(JSON))
	})

	It("Does not have flag", func() {
		Expect(HasFlag()).To(BeFalse())
	})

	It("IsStructuredOutput returns true only for json and yaml", func() {
		Expect(IsStructuredOutput()).To(BeFalse())
		SetOutput(JSON)
		Expect(IsStructuredOutput()).To(BeTrue())
		SetOutput(YAML)
		Expect(IsStructuredOutput()).To(BeTrue())
		SetOutput("xml")
		Expect(IsStructuredOutput()).To(BeFalse())
	})

})

var _ = Describe("PrintError", func() {

	var (
		origStderr *os.File
		readPipe   *os.File
		writePipe  *os.File
	)

	BeforeEach(func() {
		SetOutput("")
		origStderr = os.Stderr
		var err error
		readPipe, writePipe, err = os.Pipe()
		Expect(err).ToNot(HaveOccurred())
		os.Stderr = writePipe
	})

	AfterEach(func() {
		os.Stderr = origStderr
		SetOutput("")
		readPipe.Close()
	})

	captureOutput := func() string {
		writePipe.Close()
		os.Stderr = origStderr
		var buf bytes.Buffer
		_, err := io.Copy(&buf, readPipe)
		Expect(err).ToNot(HaveOccurred())
		return buf.String()
	}

	It("returns false and prints nothing when no output flag is set", func() {
		result := PrintError(errors.New("some error"))
		captured := captureOutput()
		Expect(result).To(BeFalse())
		Expect(captured).To(BeEmpty())
	})

	It("returns true and prints JSON error when JSON output is set", func() {
		SetOutput(JSON)
		result := PrintError(errors.New("connection failed"))
		captured := captureOutput()
		Expect(result).To(BeTrue())
		var parsed map[string]string
		Expect(json.Unmarshal([]byte(captured), &parsed)).To(Succeed())
		Expect(parsed["error"]).To(Equal("connection failed"))
	})

	It("returns true and prints JSON error when YAML output is set", func() {
		SetOutput(YAML)
		result := PrintError(errors.New("token expired"))
		captured := captureOutput()
		Expect(result).To(BeTrue())
		var parsed map[string]string
		Expect(json.Unmarshal([]byte(captured), &parsed)).To(Succeed())
		Expect(parsed["error"]).To(Equal("token expired"))
	})

	It("returns false and prints nothing when an unsupported output format is set", func() {
		SetOutput("xml")
		result := PrintError(errors.New("some error"))
		captured := captureOutput()
		Expect(result).To(BeFalse())
		Expect(captured).To(BeEmpty())
	})

})

var _ = Describe("PrintWarn", func() {

	var (
		origStderr *os.File
		readPipe   *os.File
		writePipe  *os.File
	)

	BeforeEach(func() {
		SetOutput("")
		origStderr = os.Stderr
		var err error
		readPipe, writePipe, err = os.Pipe()
		Expect(err).ToNot(HaveOccurred())
		os.Stderr = writePipe
	})

	AfterEach(func() {
		os.Stderr = origStderr
		SetOutput("")
		readPipe.Close()
	})

	captureOutput := func() string {
		writePipe.Close()
		os.Stderr = origStderr
		var buf bytes.Buffer
		_, err := io.Copy(&buf, readPipe)
		Expect(err).ToNot(HaveOccurred())
		return buf.String()
	}

	It("returns false and prints nothing when no output flag is set", func() {
		result := PrintWarn(errors.New("some warning"))
		captured := captureOutput()
		Expect(result).To(BeFalse())
		Expect(captured).To(BeEmpty())
	})

	It("returns true and prints JSON warning when JSON output is set", func() {
		SetOutput(JSON)
		result := PrintWarn(errors.New("region mismatch"))
		captured := captureOutput()
		Expect(result).To(BeTrue())
		var parsed map[string]string
		Expect(json.Unmarshal([]byte(captured), &parsed)).To(Succeed())
		Expect(parsed["warning"]).To(Equal("region mismatch"))
	})

	It("returns true and prints JSON warning when YAML output is set", func() {
		SetOutput(YAML)
		result := PrintWarn(errors.New("region mismatch"))
		captured := captureOutput()
		Expect(result).To(BeTrue())
		var parsed map[string]string
		Expect(json.Unmarshal([]byte(captured), &parsed)).To(Succeed())
		Expect(parsed["warning"]).To(Equal("region mismatch"))
	})

	It("returns false and prints nothing when an unsupported output format is set", func() {
		SetOutput("xml")
		result := PrintWarn(errors.New("some warning"))
		captured := captureOutput()
		Expect(result).To(BeFalse())
		Expect(captured).To(BeEmpty())
	})

})
