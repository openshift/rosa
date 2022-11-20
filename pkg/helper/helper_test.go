package helper_test

import (
	"fmt"
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/openshift/rosa/pkg/helper"
)

var _ = Describe("Helper", func() {
	var _ = Describe("Validates Random Label function", func() {
		var _ = Context("when generating random labels", func() {

			It("generates empty label given size 0", func() {
				label := RandomLabel(0)
				Expect("").To(Equal(label))
			})

			It("generates random labels of given size", func() {
				for i := 1; i < 11; i++ {
					label := RandomLabel(i)
					Expect(i).To(Equal(len(label)))
					regex, _ := regexp.Compile("^[a-zA-Z0-9]+$")
					Expect(true).To(Equal(regex.MatchString(label)))
				}
			})
		})

		var _ = Context("Validates RankMapStringInt", func() {
			It("Return empty when input is empty", func() {
				Expect(0).To(Equal(len(RankMapStringInt(map[string]int{}))))
			})

			It("Order descending when input is a populated map", func() {
				populatedMap := make(map[string]int)
				for i := 0; i < 50; i++ {
					populatedMap[fmt.Sprintf("operator%d", i)] = i
				}
				rankedMap := RankMapStringInt(populatedMap)
				Expect("operator49").To(Equal(rankedMap[0]))
			})

			It(
				"Order descending when input is a populated map with two prefixes of same rank, expects longest prefix first",
				func() {
					populatedMap := make(map[string]int)
					for i := 0; i < 50; i++ {
						populatedMap[fmt.Sprintf("operator%d", i)] = i
					}
					populatedMap["operator491"] = 49
					rankedMap := RankMapStringInt(populatedMap)
					Expect("operator491").To(Equal(rankedMap[0]))
				},
			)

			It(
				"Order descending when input is a populated map with two prefixes of same rank, expects highest ranked prefix first",
				func() {
					populatedMap := make(map[string]int)
					for i := 0; i < 50; i++ {
						populatedMap[fmt.Sprintf("operator%d", i)] = i
					}
					populatedMap["operator50"] = 49
					rankedMap := RankMapStringInt(populatedMap)
					Expect("operator50").To(Equal(rankedMap[0]))
				},
			)
		})

		var _ = Context("Validates Contains", func() {
			It("Return false when input is empty", func() {
				Expect(false).To(Equal(Contains([]string{}, "any")))
			})

			It("Return true when input is populated and present", func() {
				Expect(true).To(Equal(Contains([]string{"test", "any"}, "any")))
			})

			It("Return false when input is populated and not present", func() {
				Expect(false).To(Equal(Contains([]string{"test", "any"}, "none")))
			})
		})

		var _ = Context("Validates SliceToMap", func() {
			It("Return empty when slice is empty", func() {
				sliceToMap := SliceToMap([]string{})
				Expect(0).To(Equal(len(sliceToMap)))
			})

			It("Return map of slices when input is populated", func() {
				slice := []string{"test", "", "any", "something"}
				mapped := SliceToMap(slice)
				for k, v := range mapped {
					Expect(true).To(Equal(Contains(slice, k)))
					Expect(true).To(Equal(v))
				}
			})
		})

		var _ = Context("Validates RemoveStrFromSlice", func() {
			It("Do nothing when slice is empty", func() {
				slice := RemoveStrFromSlice([]string{}, "remove")
				Expect(0).To(Equal(len(slice)))
			})

			It("Do nothing when string is not found in slice", func() {
				slice := RemoveStrFromSlice([]string{"dremove", "any", "something"}, "remove")
				Expect(3).To(Equal(len(slice)))
			})

			It("Remove string and return new slice if string if found in slice", func() {
				slice := []string{"test", "", "any", "something", "remove"}
				Expect(true).To(Equal(Contains(slice, "remove")))
				newSlice := RemoveStrFromSlice(slice, "remove")
				Expect(false).To(Equal(Contains(newSlice, "remove")))
				Expect(len(slice) - 1).To(Equal(len(newSlice)))
			})
		})

	})
})
