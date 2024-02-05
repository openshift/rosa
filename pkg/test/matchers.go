package test

import (
	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

// MatchExpected ensures that `cmp.Diff(actual, expected)` returns nothing.
// Usage looks like:
//
//	Expect(actual).To(MatchExpected(expected))
func MatchExpected(expected any, opts ...cmp.Option) types.GomegaMatcher {
	return WithTransform(func(actual any) string {
		return cmp.Diff(actual, expected, opts...)
	}, BeEmpty())
}
