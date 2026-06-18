package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

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
