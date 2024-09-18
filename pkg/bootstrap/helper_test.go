package bootstrap

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParseParams", func() {
	It("should correctly parse parameters and user tags", func() {
		params := []string{
			"Key1=Value1",
			"Key2=Value2",
			"Tags=TagKey1=TagValue1,TagKey2=TagValue2",
		}

		expectedResult := map[string]string{
			"Key1": "Value1",
			"Key2": "Value2",
		}

		expectedUserTags := map[string]string{
			"TagKey1": "TagValue1",
			"TagKey2": "TagValue2",
		}

		result, userTags := ParseParams(params)

		Expect(result).To(Equal(expectedResult))
		Expect(userTags).To(Equal(expectedUserTags))
	})

	It("should handle parameters without user tags", func() {
		params := []string{
			"Key1=Value1",
			"Key2=Value2",
		}

		expectedResult := map[string]string{
			"Key1": "Value1",
			"Key2": "Value2",
		}

		expectedUserTags := map[string]string{}

		result, userTags := ParseParams(params)

		Expect(result).To(Equal(expectedResult))
		Expect(userTags).To(Equal(expectedUserTags))
	})

	It("should handle empty parameters", func() {
		params := []string{}

		expectedResult := map[string]string{}
		expectedUserTags := map[string]string{}

		result, userTags := ParseParams(params)

		Expect(result).To(Equal(expectedResult))
		Expect(userTags).To(Equal(expectedUserTags))
	})
})
