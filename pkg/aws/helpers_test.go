package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
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

var _ = Describe("IsArnAssumedRole", func() {
	When("true", func() {
		It("Is assumed role ARN", func() {
			out, err := IsArnAssumedRole("arn:aws:iam::123456789012:assumed-role/my-role")
			Expect(out).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
		})
	})
	When("false", func() {
		It("Is not assumed role ARN", func() {
			out, err := IsArnAssumedRole("arn:aws:iam::123456789012:role/my-role")
			Expect(out).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})
		It("Error (not ARN)", func() {
			out, err := IsArnAssumedRole("123456789012:role/my-role")
			Expect(out).To(BeFalse())
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("sortAccountRolesByHCPSuffix", func() {
	It("Sorts very jumbled list", func() {
		list := make([]iamtypes.Role, 0)

		roleList := []string{
			"arn:aws:iam::123123123:role/test-local-HCP-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/test2-local-HCP-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/test3-local-HCP-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/test-local-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/test2-local-HCP-ROSA-Worker-Role",
			"arn:aws:iam::123123123:role/test3-local-HCP-ROSA-Installer-Role",
			"arn:aws:iam::123123123:role/test-local-ROSA-Worker-Role",
			"arn:aws:iam::123123123:role/test2-local-ROSA-Worker-Role",
			"arn:aws:iam::123123123:role/test3-local-ROSA-Control-Plane-Role",
			"arn:aws:iam::123123123:role/test3-local-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/test-local-HCP-ROSA-Installer-Role",
		}

		expectedRoleList := []string{
			"arn:aws:iam::123123123:role/test-local-HCP-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/test-local-HCP-ROSA-Installer-Role",
			"arn:aws:iam::123123123:role/test-local-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/test-local-ROSA-Worker-Role",
			"arn:aws:iam::123123123:role/test2-local-HCP-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/test2-local-HCP-ROSA-Worker-Role",
			"arn:aws:iam::123123123:role/test2-local-ROSA-Worker-Role",
			"arn:aws:iam::123123123:role/test3-local-HCP-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/test3-local-HCP-ROSA-Installer-Role",
			"arn:aws:iam::123123123:role/test3-local-ROSA-Control-Plane-Role",
			"arn:aws:iam::123123123:role/test3-local-ROSA-Support-Role",
		}

		for _, val := range roleList {
			roleName := strings.SplitAfter(val, "role/")[1]
			list = append(list, iamtypes.Role{Arn: aws.String(val), RoleName: aws.String(roleName)})
		}

		sortAccountRolesByHCPSuffix(list)

		Expect(len(list)).To(Equal(len(roleList)))
		for i, role := range list {
			Expect(expectedRoleList[i]).To(Equal(*role.Arn))
		}
	})

	It("Sorts a large list, with similarly named roles", func() {
		list := make([]iamtypes.Role, 0)

		roleList := []string{
			"arn:aws:iam::123123123:role/test-local-HCP-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/test2-local-HCP-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/test3-local-HCP-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/test-local-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/test2-local-HCP-ROSA-Worker-Role",
			"arn:aws:iam::123123123:role/test3-local-HCP-ROSA-Installer-Role",
			"arn:aws:iam::123123123:role/test-local-ROSA-Worker-Role",
			"arn:aws:iam::123123123:role/test2-local-ROSA-Worker-Role",
			"arn:aws:iam::123123123:role/test3-local-ROSA-Control-Plane-Role",
			"arn:aws:iam::123123123:role/test3-local-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/test-local-HCP-ROSA-Installer-Role",
			"arn:aws:iam::123123123:role/testing-local-HCP-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/testing2-local-HCP-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/testing3-local-HCP-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/testing-local-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/testing2-local-HCP-ROSA-Worker-Role",
			"arn:aws:iam::123123123:role/testing3-local-HCP-ROSA-Installer-Role",
			"arn:aws:iam::123123123:role/testing-local-ROSA-Worker-Role",
			"arn:aws:iam::123123123:role/testing2-local-ROSA-Worker-Role",
			"arn:aws:iam::123123123:role/testing3-local-ROSA-Control-Plane-Role",
			"arn:aws:iam::123123123:role/testing3-local-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/testing-local-HCP-ROSA-Installer-Role",
		}

		expectedRoleList := []string{
			"arn:aws:iam::123123123:role/test-local-HCP-ROSA-Installer-Role",
			"arn:aws:iam::123123123:role/test-local-HCP-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/test-local-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/test-local-ROSA-Worker-Role",
			"arn:aws:iam::123123123:role/test2-local-HCP-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/test2-local-HCP-ROSA-Worker-Role",
			"arn:aws:iam::123123123:role/test2-local-ROSA-Worker-Role",
			"arn:aws:iam::123123123:role/test3-local-HCP-ROSA-Installer-Role",
			"arn:aws:iam::123123123:role/test3-local-HCP-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/test3-local-ROSA-Control-Plane-Role",
			"arn:aws:iam::123123123:role/test3-local-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/testing-local-HCP-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/testing-local-HCP-ROSA-Installer-Role",
			"arn:aws:iam::123123123:role/testing-local-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/testing-local-ROSA-Worker-Role",
			"arn:aws:iam::123123123:role/testing2-local-HCP-ROSA-Worker-Role",
			"arn:aws:iam::123123123:role/testing2-local-HCP-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/testing2-local-ROSA-Worker-Role",
			"arn:aws:iam::123123123:role/testing3-local-HCP-ROSA-Support-Role",
			"arn:aws:iam::123123123:role/testing3-local-HCP-ROSA-Installer-Role",
			"arn:aws:iam::123123123:role/testing3-local-ROSA-Control-Plane-Role",
			"arn:aws:iam::123123123:role/testing3-local-ROSA-Support-Role",
		}

		for _, val := range roleList {
			roleName := strings.SplitAfter(val, "role/")[1]
			list = append(list, iamtypes.Role{Arn: aws.String(val), RoleName: aws.String(roleName)})
		}

		sortAccountRolesByHCPSuffix(list)

		Expect(len(list)).To(Equal(len(roleList)))
		for i, role := range list {
			Expect(expectedRoleList[i]).To(Equal(*role.Arn))
		}
	})
})

var _ = Describe("PolicyDocument action checks", func() {
	It("detects allowed actions using the allow effect constant", func() {
		doc, err := ParsePolicyDocument(`{
			"Version":"2012-10-17",
			"Statement":[{"Effect":"Allow","Action":"sts:AssumeRole","Resource":"*"}]
		}`)
		Expect(err).NotTo(HaveOccurred(), "policy document should parse")
		Expect(doc.IsActionAllowed("sts:AssumeRole")).To(BeTrue(), "allow statement should permit sts:AssumeRole")
		Expect(doc.GetAllowedActions()).To(ConsistOf("sts:AssumeRole"),
			"allowed actions should include sts:AssumeRole")
	})
})

var _ = Describe("ARNValidator", func() {
	When("input is a valid IAM role ARN", func() {
		It("should return nil", func() {
			err := ARNValidator("arn:aws:iam::123456789012:role/MyRole")
			Expect(err).To(BeNil())
		})

		It("should return nil for a role with a path", func() {
			err := ARNValidator("arn:aws:iam::123456789012:role/path/to/MyRole")
			Expect(err).To(BeNil())
		})

		It("should return nil for a GovCloud ARN", func() {
			err := ARNValidator("arn:aws-us-gov:iam::123456789012:role/MyRole")
			Expect(err).To(BeNil())
		})
	})

	When("input is an empty string", func() {
		It("should return nil", func() {
			err := ARNValidator("")
			Expect(err).To(BeNil())
		})
	})

	When("input is not a valid ARN", func() {
		It("should return an error for a random string", func() {
			err := ARNValidator("not-an-arn")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid ARN"))
		})

		It("should return an error for a malformed ARN", func() {
			err := ARNValidator("arn:aws:iam:role/MyRole")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid ARN"))
		})
	})

	When("input is not a string", func() {
		It("should return an error for an integer", func() {
			err := ARNValidator(42)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("can only validate strings"))
		})

		It("should return an error for a slice", func() {
			err := ARNValidator([]string{"arn:aws:iam::123456789012:role/MyRole"})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("can only validate strings"))
		})
	})
})

var _ = Describe("SecretManagerArnValidator", func() {
	When("input is a valid Secrets Manager ARN", func() {
		It("should return nil", func() {
			err := SecretManagerArnValidator("arn:aws:secretsmanager:us-east-1:123456789012:secret:my-secret-AbCdEf")
			Expect(err).To(BeNil())
		})
	})

	When("input is an empty string", func() {
		It("should return nil", func() {
			err := SecretManagerArnValidator("")
			Expect(err).To(BeNil())
		})
	})

	When("input is not a valid ARN", func() {
		It("should return an error", func() {
			err := SecretManagerArnValidator("not-an-arn")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("is not a valid ARN"))
		})
	})

	When("input is an ARN for a different service", func() {
		It("should return an error for an IAM ARN", func() {
			err := SecretManagerArnValidator("arn:aws:iam::123456789012:role/MyRole")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("is not a valid secrets manager ARN"))
		})

		It("should return an error for an S3 ARN", func() {
			err := SecretManagerArnValidator("arn:aws:s3:::my-bucket")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("is not a valid secrets manager ARN"))
		})
	})

	When("input is not a string", func() {
		It("should return an error", func() {
			err := SecretManagerArnValidator(42)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("can only validate strings"))
		})
	})
})

var _ = Describe("GetPathFromARN", func() {
	When("the ARN has a path", func() {
		It("should return the path", func() {
			path, err := GetPathFromARN("arn:aws:iam::123456789012:role/path/to/MyRole")
			Expect(err).To(BeNil())
			Expect(path).To(Equal("/path/to/"))
		})

		It("should return a single-segment path", func() {
			path, err := GetPathFromARN("arn:aws:iam::123456789012:role/mypath/MyRole")
			Expect(err).To(BeNil())
			Expect(path).To(Equal("/mypath/"))
		})
	})

	When("the ARN has no path", func() {
		It("should return an empty string", func() {
			path, err := GetPathFromARN("arn:aws:iam::123456789012:role/MyRole")
			Expect(err).To(BeNil())
			Expect(path).To(Equal(""))
		})
	})

	When("the ARN is invalid", func() {
		It("should return an error", func() {
			_, err := GetPathFromARN("not-an-arn")
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("GetResourceIdFromARN", func() {
	When("the ARN has a simple resource", func() {
		It("should return the resource name", func() {
			id, err := GetResourceIdFromARN("arn:aws:iam::123456789012:role/MyRole")
			Expect(err).To(BeNil())
			Expect(id).To(Equal("MyRole"))
		})
	})

	When("the ARN has a path", func() {
		It("should return only the last segment", func() {
			id, err := GetResourceIdFromARN("arn:aws:iam::123456789012:role/path/to/MyRole")
			Expect(err).To(BeNil())
			Expect(id).To(Equal("MyRole"))
		})
	})

	When("the ARN is invalid", func() {
		It("should return an error", func() {
			_, err := GetResourceIdFromARN("not-an-arn")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("couldn't parse arn"))
		})
	})

	When("the resource has no slash separator", func() {
		It("should return an error", func() {
			_, err := GetResourceIdFromARN("arn:aws:s3:::my-bucket")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("can't find resource-id"))
		})
	})
})

var _ = Describe("GetResourceIdFromOidcProviderARN", func() {
	When("the ARN is a valid OIDC provider", func() {
		It("should return the provider URL portion", func() {
			id, err := GetResourceIdFromOidcProviderARN(
				"arn:aws:iam::123456789012:oidc-provider/oidc.example.com/id")
			Expect(err).To(BeNil())
			Expect(id).To(Equal("oidc.example.com/id"))
		})

		It("should return the provider URL without extra path", func() {
			id, err := GetResourceIdFromOidcProviderARN(
				"arn:aws:iam::123456789012:oidc-provider/oidc.example.com")
			Expect(err).To(BeNil())
			Expect(id).To(Equal("oidc.example.com"))
		})
	})

	When("the ARN is invalid", func() {
		It("should return an error", func() {
			_, err := GetResourceIdFromOidcProviderARN("not-an-arn")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("couldn't parse arn"))
		})
	})

	When("the resource has no slash separator", func() {
		It("should return an error", func() {
			_, err := GetResourceIdFromOidcProviderARN("arn:aws:s3:::my-bucket")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("can't find resource-id"))
		})
	})
})

var _ = Describe("GetRoleARN", func() {
	It("should build an ARN without a path", func() {
		result := GetRoleARN("123456789012", "MyRole", "", "aws")
		Expect(result).To(Equal("arn:aws:iam::123456789012:role/MyRole"))
	})

	It("should build an ARN with a path", func() {
		result := GetRoleARN("123456789012", "MyRole", "/my/path/", "aws")
		Expect(result).To(Equal("arn:aws:iam::123456789012:role/my/path/MyRole"))
	})

	It("should support GovCloud partition", func() {
		result := GetRoleARN("123456789012", "MyRole", "", "aws-us-gov")
		Expect(result).To(Equal("arn:aws-us-gov:iam::123456789012:role/MyRole"))
	})
})

var _ = Describe("GetOIDCProviderARN", func() {
	It("should build a valid OIDC provider ARN", func() {
		result := GetOIDCProviderARN("aws", "123456789012", "oidc.eks.us-east-1.amazonaws.com/id/EXAMPLE")
		Expect(result).To(Equal("arn:aws:iam::123456789012:oidc-provider/oidc.eks.us-east-1.amazonaws.com/id/EXAMPLE"))
	})

	It("should support GovCloud partition", func() {
		result := GetOIDCProviderARN("aws-us-gov", "123456789012", "oidc.example.com")
		Expect(result).To(Equal("arn:aws-us-gov:iam::123456789012:oidc-provider/oidc.example.com"))
	})
})

var _ = Describe("SetSubnetOption", func() {
	It("should format a subnet with a Name tag", func() {
		subnet := ec2types.Subnet{
			SubnetId:         aws.String("subnet-12345"),
			VpcId:            aws.String("vpc-abc"),
			AvailabilityZone: aws.String("us-east-1a"),
			OwnerId:          aws.String("111222333444"),
			Tags: []ec2types.Tag{
				{Key: aws.String("Name"), Value: aws.String("my-subnet")},
			},
		}
		result := SetSubnetOption(subnet)
		Expect(result).To(Equal(
			fmt.Sprintf("subnet-12345 ('my-subnet','vpc-abc','us-east-1a', Owner ID: '111222333444')")))
	})

	It("should format a subnet without a Name tag", func() {
		subnet := ec2types.Subnet{
			SubnetId:         aws.String("subnet-67890"),
			VpcId:            aws.String("vpc-def"),
			AvailabilityZone: aws.String("us-west-2b"),
			OwnerId:          aws.String("555666777888"),
			Tags:             []ec2types.Tag{},
		}
		result := SetSubnetOption(subnet)
		Expect(result).To(Equal(
			fmt.Sprintf("subnet-67890 ('','vpc-def','us-west-2b', Owner ID: '555666777888')")))
	})

	It("should use Name tag when multiple tags exist", func() {
		subnet := ec2types.Subnet{
			SubnetId:         aws.String("subnet-multi"),
			VpcId:            aws.String("vpc-xyz"),
			AvailabilityZone: aws.String("eu-west-1c"),
			OwnerId:          aws.String("999000111222"),
			Tags: []ec2types.Tag{
				{Key: aws.String("Environment"), Value: aws.String("prod")},
				{Key: aws.String("Name"), Value: aws.String("prod-subnet")},
				{Key: aws.String("Team"), Value: aws.String("platform")},
			},
		}
		result := SetSubnetOption(subnet)
		Expect(result).To(ContainSubstring("prod-subnet"))
		Expect(result).To(ContainSubstring("subnet-multi"))
	})
})

var _ = Describe("ParseOption", func() {
	It("should return the first token", func() {
		Expect(ParseOption("subnet-123 (us-east-1a)")).To(Equal("subnet-123"))
	})

	It("should return the whole string when there are no spaces", func() {
		Expect(ParseOption("subnet-123")).To(Equal("subnet-123"))
	})

	It("should return empty string for empty input", func() {
		Expect(ParseOption("")).To(Equal(""))
	})
})

var _ = Describe("HasDuplicates", func() {
	When("there are no duplicates", func() {
		It("should return false", func() {
			dup, found := HasDuplicates([]string{"a", "b", "c"})
			Expect(found).To(BeFalse())
			Expect(dup).To(Equal(""))
		})
	})

	When("there are duplicates", func() {
		It("should return the duplicate and true", func() {
			dup, found := HasDuplicates([]string{"a", "b", "a"})
			Expect(found).To(BeTrue())
			Expect(dup).To(Equal("a"))
		})

		It("should find the first duplicate", func() {
			dup, found := HasDuplicates([]string{"x", "y", "x", "y"})
			Expect(found).To(BeTrue())
			Expect(dup).To(Equal("x"))
		})
	})

	When("the slice is empty", func() {
		It("should return false", func() {
			_, found := HasDuplicates([]string{})
			Expect(found).To(BeFalse())
		})
	})

	When("the slice has one element", func() {
		It("should return false", func() {
			_, found := HasDuplicates([]string{"only"})
			Expect(found).To(BeFalse())
		})
	})
})

var _ = Describe("TrimRoleSuffix", func() {
	When("the full suffix is present", func() {
		It("should trim it", func() {
			result := TrimRoleSuffix("my-prefix-Installer-Role", "-Installer-Role")
			Expect(result).To(Equal("my-prefix"))
		})
	})

	When("the suffix is partially present due to truncation", func() {
		It("should trim the partial suffix", func() {
			result := TrimRoleSuffix("my-prefix-Installer-Ro", "-Installer-Role")
			Expect(result).To(Equal("my-prefix"))
		})

		It("should trim even a single-char partial suffix", func() {
			result := TrimRoleSuffix("my-prefix-", "-Installer-Role")
			Expect(result).To(Equal("my-prefix"))
		})
	})

	When("the suffix is not present at all", func() {
		It("should return the original string", func() {
			result := TrimRoleSuffix("something-else", "-Installer-Role")
			Expect(result).To(Equal("something-else"))
		})
	})

	When("the name equals the suffix", func() {
		It("should return an empty string", func() {
			result := TrimRoleSuffix("-Installer-Role", "-Installer-Role")
			Expect(result).To(Equal(""))
		})
	})

	When("the suffix is empty", func() {
		It("should return the original string", func() {
			result := TrimRoleSuffix("my-role", "")
			Expect(result).To(Equal("my-role"))
		})
	})
})

var _ = Describe("ARNPathValidator", func() {
	When("input is a valid path", func() {
		It("should return nil", func() {
			err := ARNPathValidator("/my/path/")
			Expect(err).To(BeNil())
		})
	})

	When("input is an empty string", func() {
		It("should return nil", func() {
			err := ARNPathValidator("")
			Expect(err).To(BeNil())
		})
	})

	When("input is an invalid path", func() {
		It("should return an error for a path without slashes", func() {
			err := ARNPathValidator("noslash")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("must begin and end with /"))
		})
	})

	When("input is not a string", func() {
		It("should return an error", func() {
			err := ARNPathValidator(42)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("can only validate strings"))
		})
	})
})

var _ = Describe("UserNoProxyValidator", func() {
	When("input is a valid single IP", func() {
		It("should return nil", func() {
			err := UserNoProxyValidator("10.0.0.1")
			Expect(err).To(BeNil())
		})
	})

	When("input is a valid CIDR", func() {
		It("should return nil", func() {
			err := UserNoProxyValidator("10.0.0.0/16")
			Expect(err).To(BeNil())
		})
	})

	When("input is a valid domain", func() {
		It("should return nil", func() {
			err := UserNoProxyValidator(".example.com")
			Expect(err).To(BeNil())
		})
	})

	When("input is an empty string", func() {
		It("should return nil", func() {
			err := UserNoProxyValidator("")
			Expect(err).To(BeNil())
		})
	})

	When("input is invalid", func() {
		It("should return an error", func() {
			err := UserNoProxyValidator("not a valid proxy")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a valid user no-proxy value"))
		})
	})

	When("input has multiple comma-separated valid values", func() {
		It("should return nil", func() {
			err := UserNoProxyValidator("10.0.0.1,10.0.0.2")
			Expect(err).To(BeNil())
		})
	})

	When("input has one invalid value in comma-separated list", func() {
		It("should return an error", func() {
			err := UserNoProxyValidator("10.0.0.1,not valid")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected a valid user no-proxy value"))
		})
	})

	When("input is not a string", func() {
		It("should return an error", func() {
			err := UserNoProxyValidator(42)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("can only validate strings"))
		})
	})
})

var _ = Describe("UserNoProxyDuplicateValidator", func() {
	When("values are unique", func() {
		It("should return nil", func() {
			err := UserNoProxyDuplicateValidator("a.com,b.com")
			Expect(err).To(BeNil())
		})
	})

	When("values have duplicates", func() {
		It("should return an error", func() {
			err := UserNoProxyDuplicateValidator("a.com,b.com,a.com")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate key"))
		})
	})

	When("input is an empty string", func() {
		It("should return nil", func() {
			err := UserNoProxyDuplicateValidator("")
			Expect(err).To(BeNil())
		})
	})

	When("input is not a string", func() {
		It("should return an error", func() {
			err := UserNoProxyDuplicateValidator(42)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("can only validate strings"))
		})
	})
})

var _ = Describe("GetOCMRoleName", func() {
	It("should build the correct OCM role name", func() {
		result := GetOCMRoleName("ManagedOpenShift", "OCM", "12345")
		Expect(result).To(Equal("ManagedOpenShift-OCM-Role-12345"))
	})
})

var _ = Describe("GetUserRoleName", func() {
	It("should build the correct user role name", func() {
		result := GetUserRoleName("ManagedOpenShift", "User", "testuser")
		Expect(result).To(Equal("ManagedOpenShift-User-testuser-Role"))
	})
})

var _ = Describe("GetOperatorPolicyName", func() {
	It("should build the correct operator policy name", func() {
		result := GetOperatorPolicyName("my-prefix", "openshift-ingress", "cloud-credentials")
		Expect(result).To(Equal("my-prefix-openshift-ingress-cloud-credentials"))
	})
})

var _ = Describe("GetPolicyArn", func() {
	It("should build a policy ARN without path", func() {
		result := GetPolicyArn("aws", "123456789012", "MyPolicy", "")
		Expect(result).To(Equal("arn:aws:iam::123456789012:policy/MyPolicy"))
	})

	It("should build a policy ARN with path", func() {
		result := GetPolicyArn("aws", "123456789012", "MyPolicy", "/my/path/")
		Expect(result).To(Equal("arn:aws:iam::123456789012:policy/my/path/MyPolicy"))
	})
})

var _ = Describe("GetOperatorPolicyARN", func() {
	It("should build an operator policy ARN without path", func() {
		result := GetOperatorPolicyARN(
			"aws", "123456789012", "my-prefix", "openshift-ingress", "cloud-credentials", "")
		Expect(result).To(Equal(
			"arn:aws:iam::123456789012:policy/my-prefix-openshift-ingress-cloud-credentials"))
	})
})

var _ = Describe("IsStandardNamedAccountRole", func() {
	When("the role name follows the standard pattern", func() {
		It("should return true and the prefix", func() {
			isStandard, prefix := IsStandardNamedAccountRole("my-prefix-Installer-Role", "Installer")
			Expect(isStandard).To(BeTrue())
			Expect(prefix).To(Equal("my-prefix"))
		})
	})

	When("the role name does not follow the standard pattern", func() {
		It("should return false and the original name", func() {
			isStandard, prefix := IsStandardNamedAccountRole("some-other-name", "Installer")
			Expect(isStandard).To(BeFalse())
			Expect(prefix).To(Equal("some-other-name"))
		})
	})
})

var _ = Describe("isSTS", func() {
	When("the ARN is an STS assumed-role", func() {
		It("should return true", func() {
			stsARN := arn.ARN{
				Partition: "aws",
				Service:   "sts",
				AccountID: "123456789012",
				Resource:  "assumed-role/MyRole/session",
			}
			Expect(isSTS(stsARN)).To(BeTrue())
		})
	})

	When("the ARN is an IAM user", func() {
		It("should return false", func() {
			iamARN := arn.ARN{
				Partition: "aws",
				Service:   "iam",
				AccountID: "123456789012",
				Resource:  "user/MyUser",
			}
			Expect(isSTS(iamARN)).To(BeFalse())
		})
	})
})

var _ = Describe("resolveSTSRole", func() {
	When("the ARN is a valid STS assumed-role", func() {
		It("should return the corresponding IAM role ARN", func() {
			stsARN := arn.ARN{
				Partition: "aws",
				Service:   "sts",
				AccountID: "123456789012",
				Resource:  "assumed-role/MyRole/session",
			}
			result, err := resolveSTSRole(stsARN)
			Expect(err).To(BeNil())
			Expect(*result).To(Equal("arn:aws:iam::123456789012:role/MyRole"))
		})
	})

	When("the ARN is not an STS ARN", func() {
		It("should return an error", func() {
			iamARN := arn.ARN{
				Partition: "aws",
				Service:   "iam",
				AccountID: "123456789012",
				Resource:  "user/MyUser",
			}
			_, err := resolveSTSRole(iamARN)
			Expect(err).To(HaveOccurred())
		})
	})

	When("the STS ARN has only two resource segments", func() {
		It("should return an error", func() {
			malformedARN := arn.ARN{
				Partition: "aws",
				Service:   "sts",
				AccountID: "123456789012",
				Resource:  "assumed-role/MyRole",
			}
			_, err := resolveSTSRole(malformedARN)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("doesn't appear to have"))
		})
	})
})
