package test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test helpers", func() {
	Context("StdOutReader", func() {
		var stringToPrint = "Testing\nTesting\n\t123"
		var stringToPrint2 = "testing out recording and reading stdout, which should allow us to easily test actual " +
			"output printed to the terminal it's nice to have, and eases testing by not only focusing on function " +
			"returns, but also functions which do not return anything\t\t\t123"
		It("Record and read stdout", func() {
			t := NewTestRuntime()
			err := t.StdOutReader.Record()
			Expect(err).ToNot(HaveOccurred())
			fmt.Println(stringToPrint)
			out, err := t.StdOutReader.Read()
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal(stringToPrint + "\n"))

			err = t.StdOutReader.Record()
			Expect(err).ToNot(HaveOccurred())
			fmt.Println(stringToPrint2)
			out, err = t.StdOutReader.Read()
			Expect(err).ToNot(HaveOccurred())
			Expect(out).To(Equal(stringToPrint2 + "\n"))

			// Make sure it does not continue to capture
			fmt.Println("!!!!!")
			_, err = t.StdOutReader.Read()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("close |1: file already closed"))
		})
	})
})
