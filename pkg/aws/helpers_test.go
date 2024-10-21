package aws

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("UserTagValidator", func() {
	When("given a string input", func() {
		When("input is empty", func() {
			It("should return nil", func() {
				err := UserTagValidator("")
				Expect(err).To(BeNil())
			})
		})

		When("the input contains valid tags", func() {
			It("should return nil", func() {
				err := UserTagValidator("tag1 value1, tag2 value2")
				Expect(err).To(BeNil())
			})
		})

		When("the input contains legacy tags format", func() {
			It("should return nil", func() {
				err := UserTagValidator("tag1:value1,tag2:value2")
				Expect(err).To(BeNil())
			})
		})

		When("the input contains invalid tags", func() {
			It("should return an error", func() {
				err := UserTagValidator("foo bar,tag2=value2")
				Expect(err).To(MatchError("invalid tag format for tag '[tag2=value2]'. Expected tag format: 'key value'"))
			})

			It("should return an error if the tag has too many elements", func() {
				err := UserTagValidator("a:b:c")
				Expect(err).To(MatchError("invalid tag format for tag '[a b c]'. Expected tag format: 'key value'"))
			})

			It("should return an error if a tag is missing a key", func() {
				err := UserTagValidator(":value1,tag2:value2")
				Expect(err).To(MatchError("invalid tag format, tag key or tag value can not be empty"))
			})

			It("should return an error if a tag is missing a key", func() {
				err := UserTagValidator("tag1:,tag2:value2")
				Expect(err).To(MatchError("invalid tag format, tag key or tag value can not be empty"))
			})

			It("should return an error if a tag key contains invalid characters", func() {
				err := UserTagValidator("tag1$:value1,tag2:value2")
				Expect(err).To(MatchError(fmt.Sprintf("expected a valid user tag key 'tag1$' matching %s",
					UserTagKeyRE.String())))
			})

			It("should return an error if a tag value contains invalid characters", func() {
				err := UserTagValidator("tag1:value$1,tag2:value2")
				Expect(err).To(MatchError(fmt.Sprintf("expected a valid user tag value 'value$1' matching %s",
					UserTagValueRE.String())))
			})

			When("the input contains tags with colon or space in the value", func() {
				It("should not return an error if the tag is properly formatted", func() {
					err := UserTagValidator("tag1 value:1,tag:2 value2")
					Expect(err).To(BeNil())
				})
			})
		})
	})

	When("given a slice of strings", func() {
		When("input is empty", func() {
			It("should return nil", func() {
				err := UserTagValidator([]string{})
				Expect(err).To(BeNil())
			})
		})

		When("the input contains valid tags", func() {
			It("should return nil", func() {
				err := UserTagValidator([]string{"tag1 value1", "tag2 value2"})
				Expect(err).To(BeNil())
			})
		})

		When("the input contains legacy tags format", func() {
			It("should return nil", func() {
				err := UserTagValidator([]string{"tag1:value1", "tag2:value2"})
				Expect(err).To(BeNil())
			})
		})

		When("the input contains invalid tags", func() {
			It("should return an error", func() {
				err := UserTagValidator([]string{"foo bar", "tag2=value2"})
				Expect(err).To(MatchError("invalid tag format for tag '[tag2=value2]'. Expected tag format: 'key value'"))
			})

			It("should return an error if a tag is missing a key", func() {
				err := UserTagValidator([]string{":value1", "tag2:value2"})
				Expect(err).To(MatchError("invalid tag format, tag key or tag value can not be empty"))
			})

			It("should return an error if a tag is missing a key", func() {
				err := UserTagValidator([]string{"tag1:", "tag2:value2"})
				Expect(err).To(MatchError("invalid tag format, tag key or tag value can not be empty"))
			})

			It("should return an error if a tag key contains invalid characters", func() {
				err := UserTagValidator([]string{"tag1$:value1", "tag2:value2"})
				Expect(err).To(MatchError(fmt.Sprintf("expected a valid user tag key 'tag1$' matching %s",
					UserTagKeyRE.String())))
			})

			It("should return an error if a tag value contains invalid characters", func() {
				err := UserTagValidator([]string{"tag1:value$1", "tag2:value2"})
				Expect(err).To(MatchError(fmt.Sprintf("expected a valid user tag value 'value$1' matching %s",
					UserTagValueRE.String())))
			})

			When("the input contains tags with colon or space in the value", func() {
				It("should not return an error if the tag is properly formatted", func() {
					err := UserTagValidator([]string{"tag1 value:1", "tag:2 value2"})
					Expect(err).To(BeNil())
				})
			})
		})
	})

	Describe("when given a non-string input", func() {
		It("should return an error", func() {
			err := UserTagValidator(42)
			Expect(err).To(MatchError("can only validate string types, got int"))
		})
	})

	Describe("when given a non-string slice input", func() {
		It("should return an error", func() {
			err := UserTagValidator([]int{42})
			Expect(err).To(MatchError("unable to verify tags, incompatible type," +
				" expected slice of string got: 'slice'"))
		})
	})
})

var _ = Describe("GetTagsDelimiter", func() {
	When("tag contains ' '", func() {
		It("should return ' '", func() {
			Expect(GetTagsDelimiter([]string{"key value", "foo bar"})).To(Equal(" "))
			Expect(GetTagsDelimiter([]string{"foo bar baz", "key value"})).To(Equal(" "))
		})
	})

	When("tag contains :", func() {
		It("should return :", func() {
			Expect(GetTagsDelimiter([]string{"key:value", "foo:bar"})).To(Equal(":"))
			Expect(GetTagsDelimiter([]string{"foo:bar:baz", "key:value"})).To(Equal(":"))
		})
	})

	When("tag does not contain either ' ' or :", func() {
		It("should default to ':'", func() {
			Expect(GetTagsDelimiter([]string{"keyvalue", "foobar"})).To(Equal(":"))
			Expect(GetTagsDelimiter([]string{"foo=bar", "key=value"})).To(Equal(":"))
			Expect(GetTagsDelimiter([]string{""})).To(Equal(":"))
		})
	})
})

var _ = Describe("UserTagDuplicateValidator", func() {
	When("given an empty string", func() {
		It("should return nil", func() {
			err := UserTagDuplicateValidator("")
			Expect(err).To(BeNil())
		})
	})

	Context("when given a non-string input", func() {
		It("should return an error", func() {
			err := UserTagDuplicateValidator(123)
			Expect(err).To(MatchError("can only validate strings, got 123"))
		})
	})

	Context("space separated", func() {
		When("given a string with unique tags", func() {
			It("should return nil", func() {
				err := UserTagDuplicateValidator("key1 value1,key2 value2,key3 value3")
				Expect(err).To(BeNil())
			})
		})

		When("given a string with duplicate tags", func() {
			It("should return an error", func() {
				err := UserTagDuplicateValidator("key1 value1,key2 value2,key1 value3")
				Expect(err).To(MatchError("user tag keys must be unique, duplicate key 'key1' found"))
			})
		})

		When("given a string with a space prefix with unique tags", func() {
			It("should return nil", func() {
				err := UserTagDuplicateValidator(" key1 value1, key2 value2, key3 value3")
				Expect(err).To(BeNil())
			})
		})

		When("given a string with a space prefix with duplicate tags", func() {
			It("should return an error", func() {
				err := UserTagDuplicateValidator(" key1 value1, key2 value2, key1 value3")
				Expect(err).To(MatchError("user tag keys must be unique, duplicate key 'key1' found"))
			})
		})
	})

	Context("colon separated", func() {
		When("given a string with unique tags", func() {
			It("should return nil", func() {
				err := UserTagDuplicateValidator("key1:value1,key2:value2,key3:value3")
				Expect(err).To(BeNil())
			})
		})

		When("given a string with duplicate tags", func() {
			It("should return an error", func() {
				err := UserTagDuplicateValidator("key1:value1,key2:value2,key1:value3")
				Expect(err).To(MatchError("user tag keys must be unique, duplicate key 'key1' found"))
			})
		})

		When("given a string with a space prefix with unique tags", func() {
			It("should return nil", func() {
				err := UserTagDuplicateValidator(" key1:value1, key2:value2, key3:value3")
				Expect(err).To(BeNil())
			})
		})

		When("given a string with a space prefix with duplicate tags colon separated", func() {
			It("should return an error", func() {
				err := UserTagDuplicateValidator(" key1:value1, key2:value2, key1:value3")
				Expect(err).To(MatchError("user tag keys must be unique, duplicate key 'key1' found"))
			})
		})
	})
})

var _ = Describe("GetHcpAccountRolePolicyKeys", func() {
	When("role_type contains instance_worker", func() {
		It("should return correct policy keys", func() {
			policyKeys := GetHcpAccountRolePolicyKeys("instance_worker")
			Expect(len(policyKeys)).To(Equal(1))
			Expect(policyKeys[0]).To(Equal("sts_hcp_instance_worker_permission_policy"))
		})
	})
	When("role_type contains installer", func() {
		It("should return correct policy keys", func() {
			policyKeys := GetHcpAccountRolePolicyKeys("installer")
			Expect(len(policyKeys)).To(Equal(1))
			Expect(policyKeys[0]).To(Equal("sts_hcp_installer_permission_policy"))
		})
	})
	When("role_type contains support", func() {
		It("should return correct policy keys", func() {
			policyKeys := GetHcpAccountRolePolicyKeys("support")
			Expect(len(policyKeys)).To(Equal(1))
			Expect(policyKeys[0]).To(Equal("sts_hcp_support_permission_policy"))
		})
	})
})
