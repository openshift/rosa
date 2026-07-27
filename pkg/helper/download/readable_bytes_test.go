package helper

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("readableBytes", func() {
	DescribeTable("formats byte values",
		func(input uint64, expected string) {
			Expect(readableBytes(input)).To(Equal(expected))
		},
		Entry("zero bytes", uint64(0), "0 B"),
		Entry("9 bytes", uint64(9), "9 B"),
		Entry("10 bytes", uint64(10), "10 B"),
		Entry("999 bytes", uint64(999), "999 B"),
		Entry("1000 bytes (1 kB boundary)", uint64(1000), "1.0 kB"),
		Entry("1250 bytes rounds up to 1.3 kB", uint64(1250), "1.3 kB"),
		Entry("9999 bytes", uint64(9999), "10 kB"),
		Entry("10 kB", uint64(10000), "10 kB"),
		Entry("1 MB boundary", uint64(1000000), "1.0 MB"),
		Entry("9999999 bytes", uint64(9999999), "10 MB"),
		Entry("10 MB", uint64(10000000), "10 MB"),
		Entry("1 GB boundary", uint64(1000000000), "1.0 GB"),
		Entry("10 GB", uint64(10000000000), "10 GB"),
		Entry("1 TB boundary", uint64(1000000000000), "1.0 TB"),
		Entry("1 PB boundary", uint64(1000000000000000), "1.0 PB"),
		Entry("1 EB boundary", uint64(1000000000000000000), "1.0 EB"),
		Entry("max uint64", uint64(math.MaxUint64), "18 EB"),
	)
})
