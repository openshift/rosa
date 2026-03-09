package ocm

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("FeatureFilter", func() {
	DescribeTable("ParseFeatureFilter", func(
		input string,
		want FeatureFilter,
		wantErr bool,
		errString string,
	) {
		filter, err := ParseFeatureFilter(input)
		if wantErr {
			Expect(filter).To(BeZero())
			Expect(err).To(MatchError(ContainSubstring(errString)))
			return
		}
		Expect(err).NotTo(HaveOccurred())
		Expect(filter).To(Equal(want))
	},
		Entry("fails for empty input", "", FeatureFilter{}, true, "may not be empty"),
		Entry("fails for invalid char @", "fe@ture", FeatureFilter{}, true, "invalid character '@'"),
		Entry("fails for too long of string", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", FeatureFilter{}, true, "must not exceed 32 characters in length"),
		Entry("fails for key=value syntax", "feature=something", FeatureFilter{}, true, "invalid character '='"),
		Entry("fails with single quotes", "'key'", FeatureFilter{}, true, "invalid character '''"),
		Entry("fails with internal whitespace", "feature name", FeatureFilter{}, true, "invalid character ' '"),
		Entry("fails with external whitespace", " feature_name", FeatureFilter{}, true, "invalid character ' '"),
		Entry("fails with external whitespace", "feature_name ", FeatureFilter{}, true, "invalid character ' '"),
		Entry("succeeds for up to 32 characters", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", FeatureFilter{featureName: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, false, ""),
		Entry("succeeds for the feature we're actually adding this for", "win_li", FeatureFilter{featureName: "win_li"}, false, ""),
		Entry("succeeds for coverage of whole lowercase alphabet", "abcdefghijklmnopqrstuvqxyz", FeatureFilter{featureName: "abcdefghijklmnopqrstuvqxyz"}, false, ""),
		Entry("succeeds for coverage of whole uppercase alphabet", "ABCDEFGHIJKLMNOPQRSTUVQXYZ", FeatureFilter{featureName: "ABCDEFGHIJKLMNOPQRSTUVQXYZ"}, false, ""),
		Entry("succeeds for coverage of all digits", "1234567890", FeatureFilter{featureName: "1234567890"}, false, ""),
	)
	DescribeTable("FeatureFilter.String()", func(
		filter FeatureFilter,
		expectedString string,
	) {
		Expect(filter.String()).To(Equal(expectedString))
	},
		Entry("feature win_li",
			FeatureFilter{featureName: "win_li"},
			"features.win_li = 'true'",
		),
		Entry("feature feat1",
			FeatureFilter{featureName: "feat1"},
			"features.feat1 = 'true'",
		),
		Entry("feature with 32 characters",
			FeatureFilter{featureName: "featurewith_thirty_two_characters"},
			"features.featurewith_thirty_two_characters = 'true'",
		),
	)
	DescribeTable("FeatureFilters.String()", func(
		filters FeatureFilters,
		expectedString string,
	) {
		Expect(filters.String()).To(Equal(expectedString))
	},
		Entry("empty list",
			FeatureFilters{},
			"",
		),
		Entry("nil list",
			nil,
			"",
		),
		Entry("feature feat1 and feat2",
			FeatureFilters{FeatureFilter{featureName: "feat1"}, FeatureFilter{featureName: "feat2"}},
			"features.feat1 = 'true' AND features.feat2 = 'true'",
		),
		Entry("feature feat1 and feat2 and feat3",
			FeatureFilters{
				FeatureFilter{featureName: "feat1"},
				FeatureFilter{featureName: "feat2"},
				FeatureFilter{featureName: "feat3"}},
			"features.feat1 = 'true' AND features.feat2 = 'true' AND features.feat3 = 'true'",
		),
	)
})
