package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Is Account Role Version Compatible", func() {
	When("Role isn't an account role", func() {
		It("Should return not compatible", func() {
			isCompatible, err := isAccountRoleVersionCompatible([]*iam.Tag{}, InstallerAccountRole, "4.14")
			Expect(err).To(BeNil())
			Expect(isCompatible).To(Equal(false))
		})
	})
	When("Role OCP version isn't compatible", func() {
		It("Should return not compatible", func() {
			tagsList := []*iam.Tag{
				{
					Key:   aws.String("rosa_openshift_version"),
					Value: aws.String("4.13"),
				},
			}
			isCompatible, err := isAccountRoleVersionCompatible(tagsList, InstallerAccountRole, "4.14")
			Expect(err).To(BeNil())
			Expect(isCompatible).To(Equal(false))
		})
	})
	When("Role version is compatible", func() {
		It("Should return compatible", func() {
			tagsList := []*iam.Tag{
				{
					Key:   aws.String("rosa_openshift_version"),
					Value: aws.String("4.14"),
				},
			}
			isCompatible, err := isAccountRoleVersionCompatible(tagsList, InstallerAccountRole, "4.14")
			Expect(err).To(BeNil())
			Expect(isCompatible).To(Equal(true))
		})
	})
	When("Role has managed policies, ignores openshift version", func() {
		It("Should return compatible", func() {
			tagsList := []*iam.Tag{
				{
					Key:   aws.String("rosa_openshift_version"),
					Value: aws.String("4.12"),
				},
				{
					Key:   aws.String("rosa_managed_policies"),
					Value: aws.String("true"),
				},
			}
			isCompatible, err := isAccountRoleVersionCompatible(tagsList, InstallerAccountRole, "4.14")
			Expect(err).To(BeNil())
			Expect(isCompatible).To(Equal(true))
		})
	})
	When("Role has HCP managed policies when trying to create classic cluster", func() {
		It("Should return incompatible", func() {
			tagsList := []*iam.Tag{
				{
					Key:   aws.String("rosa_openshift_version"),
					Value: aws.String("4.12"),
				},
				{
					Key:   aws.String("rosa_managed_policies"),
					Value: aws.String("true"),
				},
				{
					Key:   aws.String("rosa_hcp_policies"),
					Value: aws.String("true"),
				},
			}
			isCompatible, err := validateAccountRoleVersionCompatibilityClassic(InstallerAccountRole, "4.12",
				tagsList)
			Expect(err).To(BeNil())
			Expect(isCompatible).To(Equal(false))
		})
	})
	When("Role has classic policies when trying to create an HCP cluster", func() {
		It("Should return incompatible", func() {
			tagsList := []*iam.Tag{
				{
					Key:   aws.String("rosa_openshift_version"),
					Value: aws.String("4.12"),
				},
				{
					Key:   aws.String("rosa_managed_policies"),
					Value: aws.String("true"),
				},
			}
			isCompatible, err := validateAccountRoleVersionCompatibilityHostedCp(InstallerAccountRole, "4.12",
				tagsList)
			Expect(err).To(BeNil())
			Expect(isCompatible).To(Equal(false))
		})
	})
})
