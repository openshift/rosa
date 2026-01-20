package autonode

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAutoNode(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "AutoNode Helper Suite")
}

var _ = Describe("ValidateAutoNodeValue", func() {
	It("returns nil when value is 'enabled'", func() {
		err := ValidateAutoNodeValue("enabled")
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns error when value is empty", func() {
		err := ValidateAutoNodeValue("")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(fmt.Sprintf("invalid value for --%s, only '%s' is supported",
			AutoNodeFlagName, AutoNodeModeEnabled)))
	})

	It("returns error when value is arbitrary string", func() {
		err := ValidateAutoNodeValue("invalid-value")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(fmt.Sprintf("invalid value for --%s, only '%s' is supported",
			AutoNodeFlagName, AutoNodeModeEnabled)))
	})
})

var _ = Describe("ValidateRoleARN", func() {
	Context("when role ARN is invalid", func() {
		It("returns error when value is empty", func() {
			err := ValidateRoleARN("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("IAM role ARN cannot be empty"))
		})

		It("returns error when ARN format is invalid - missing parts", func() {
			err := ValidateRoleARN("arn:aws:iam::123456789012")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid IAM role ARN format"))
			Expect(err.Error()).To(ContainSubstring("expected format: arn:aws:iam::<account-id>:role/<role-name>"))
		})

		It("returns error when ARN format is invalid - not a role", func() {
			err := ValidateRoleARN("arn:aws:iam::123456789012:user/MyUser")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid IAM role ARN format"))
		})

		It("returns error when ARN has invalid account ID", func() {
			err := ValidateRoleARN("arn:aws:iam::abc:role/MyRole")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid IAM role ARN format"))
		})
	})

	Context("when role ARN is valid", func() {
		It("returns nil for valid standard role ARN", func() {
			err := ValidateRoleARN("arn:aws:iam::123456789012:role/MyRole")
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns nil for valid role ARN with path", func() {
			err := ValidateRoleARN("arn:aws:iam::123456789012:role/path/to/MyRole")
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns nil for valid role ARN with AWS partition", func() {
			err := ValidateRoleARN("arn:aws:iam::123456789012:role/AutoNodeRole")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

var _ = Describe("ValidateAutoNodeConfiguration", func() {
	Context("when enabling AutoNode", func() {
		It("returns error when AutoNode is already enabled", func() {
			err := ValidateAutoNodeConfiguration(true, false, true, "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("AutoNode is already enabled for this cluster"))
		})

		It("returns error when enabling without IAM role ARN", func() {
			err := ValidateAutoNodeConfiguration(true, false, false, "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf(
				"the AutoNode IAM role ARN flag '--%s' is required when enabling AutoNode",
				AutoNodeIAMRoleArnFlagName)))
		})

		It("returns error when enabling with empty IAM role ARN", func() {
			err := ValidateAutoNodeConfiguration(true, true, false, "")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf(
				"the AutoNode IAM role ARN flag '--%s' is required when enabling AutoNode",
				AutoNodeIAMRoleArnFlagName)))
		})

		It("returns nil when enabling with valid IAM role ARN", func() {
			err := ValidateAutoNodeConfiguration(true, true, false, "arn:aws:iam::123456789012:role/MyRole")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when updating IAM role ARN", func() {
		It("returns error when AutoNode is not enabled", func() {
			err := ValidateAutoNodeConfiguration(false, true, false, "arn:aws:iam::123456789012:role/MyRole")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf(
				"cannot update IAM role ARN when AutoNode is not enabled. Enable AutoNode first with --%s=%s",
				AutoNodeFlagName, AutoNodeModeEnabled)))
		})

		It("returns nil when updating role ARN with AutoNode already enabled", func() {
			err := ValidateAutoNodeConfiguration(false, true, true, "arn:aws:iam::123456789012:role/MyRole")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when no changes are made", func() {
		It("returns nil when neither flag is changed", func() {
			err := ValidateAutoNodeConfiguration(false, false, false, "")
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns nil when neither flag is changed and AutoNode is enabled", func() {
			err := ValidateAutoNodeConfiguration(false, false, true, "")
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

var _ = Describe("DetermineAutoNodeMode", func() {
	It("returns the flag value when autonode flag is changed", func() {
		mode := DetermineAutoNodeMode(true, "enabled")
		Expect(mode).To(Equal("enabled"))
	})

	It("returns empty string when autonode flag is not changed", func() {
		mode := DetermineAutoNodeMode(false, "")
		Expect(mode).To(Equal(""))
	})

	It("returns empty string when only updating IAM role", func() {
		mode := DetermineAutoNodeMode(false, "enabled")
		Expect(mode).To(Equal(""))
	})
})
