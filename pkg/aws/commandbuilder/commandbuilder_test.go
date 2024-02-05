package commandbuilder_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/openshift/rosa/pkg/aws/commandbuilder"
)

const (
	awsIamCreateRole = "aws iam create-role \\\n"
	testTag          = "test-tag"
)

var _ = Describe("Commandbuilder", func() {
	var _ = Describe("Validates AWS Command Builder", func() {
		var _ = Context("when building IAM commands", func() {

			It("generates empty iam command", func() {
				command := NewIAMCommandBuilder().Build()
				Expect("aws iam").To(Equal(command))
			})

			It("generates iam command without params", func() {
				command := NewIAMCommandBuilder().SetCommand(CreateRole).Build()
				Expect(awsIamCreateRole).To(Equal(command))
			})

			It("generates iam command with params", func() {
				command := NewIAMCommandBuilder().
					SetCommand(CreateRole).
					AddParam(PolicyArn, "arn:aws:iam::765374464689:policy/rosa-awscb-test-Installer-Policy").
					AddParam(RoleName, "rosa-awscb-test-Installer-Role").
					Build()
				Expect(
					awsIamCreateRole +
						"\t--policy-arn arn:aws:iam::765374464689:policy/rosa-awscb-test-Installer-Policy \\\n" +
						"\t--role-name rosa-awscb-test-Installer-Role",
				).To(Equal(command))
			})

			It("generates iam command with param without value", func() {
				command := NewIAMCommandBuilder().
					SetCommand(CreateRole).
					AddParamNoValue(SetAsDefault).
					Build()
				Expect(awsIamCreateRole +
					"\t--set-as-default").To(Equal(command))
			})

			It("generates iam command with param with tags param out in order", func() {
				command := NewIAMCommandBuilder().
					SetCommand(CreateRole).
					AddTags(map[string]string{
						"managed": "value",
						testTag:   "true",
					}).
					Build()
				Expect(
					awsIamCreateRole +
						"\t--tags Key=managed,Value=value Key=test-tag,Value=true",
				).To(Equal(command))
			})

			It("generates iam command with param with tags param out of order", func() {
				command := NewIAMCommandBuilder().
					SetCommand(CreateRole).
					AddTags(map[string]string{
						testTag:   "value",
						"managed": "true",
					}).
					Build()
				Expect(
					awsIamCreateRole +
						"\t--tags Key=managed,Value=true Key=test-tag,Value=value",
				).To(Equal(command))
			})

			It("generates iam command with one of each type of param", func() {
				command := NewIAMCommandBuilder().
					SetCommand(CreateRole).
					AddParam(RoleName, "rosa-awscb-test-Installer-Role").
					AddParam(PolicyArn, "arn:aws:iam::765374464689:policy/rosa-awscb-test-Installer-Policy").
					AddParamNoValue(SetAsDefault).
					AddTags(map[string]string{
						testTag:   "value",
						"managed": "true",
					}).
					Build()
				Expect(
					awsIamCreateRole +
						"\t--policy-arn arn:aws:iam::765374464689:policy/rosa-awscb-test-Installer-Policy \\\n" +
						"\t--role-name rosa-awscb-test-Installer-Role \\\n" +
						"\t--set-as-default \\\n" +
						"\t--tags Key=managed,Value=true Key=test-tag,Value=value",
				).To(Equal(command))
			})
		})
	})
})
