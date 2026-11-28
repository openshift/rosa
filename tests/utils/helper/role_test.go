// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package helper

import (
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// paramSep mirrors awscb.ParamNewLineSeparator, which rosacli uses to wrap the
// params of a single command onto following lines.
const paramSep = " \\\n\t"

var _ = Describe("ExtractCommandsToDeleteAWSResources", func() {
	It("splits commands joined by a single newline", func() {
		// `rosa delete operator-roles --mode manual` joins with "\n".
		output := bytes.Buffer{}
		output.WriteString(
			"aws iam detach-role-policy" + paramSep +
				"--policy-arn arn:aws:iam::aws:policy/service-role/ROSANodePoolManagementPolicy" + paramSep +
				"--role-name opp60956h-ro-kube-system-capa-controller-manager\n" +
				"aws iam delete-role" + paramSep +
				"--role-name opp60956h-ro-kube-system-capa-controller-manager\n" +
				"aws iam detach-role-policy" + paramSep +
				"--policy-arn arn:aws:iam::aws:policy/service-role/ROSAKMSProviderPolicy" + paramSep +
				"--role-name opp60956h-ro-kube-system-kms-provider\n\n")

		Expect(ExtractCommandsToDeleteAWSResources(output)).To(Equal([]string{
			"aws iam detach-role-policy " +
				"--policy-arn arn:aws:iam::aws:policy/service-role/ROSANodePoolManagementPolicy " +
				"--role-name opp60956h-ro-kube-system-capa-controller-manager",
			"aws iam delete-role --role-name opp60956h-ro-kube-system-capa-controller-manager",
			"aws iam detach-role-policy " +
				"--policy-arn arn:aws:iam::aws:policy/service-role/ROSAKMSProviderPolicy " +
				"--role-name opp60956h-ro-kube-system-kms-provider",
		}))
	})

	It("splits commands joined by a blank line", func() {
		// `rosa delete oidc-config --mode manual` joins with awscb.JoinCommands.
		output := bytes.Buffer{}
		output.WriteString(
			"aws iam delete-role" + paramSep + "--role-name test-role\n\n" +
				"aws iam delete-policy" + paramSep + "--policy-arn arn:aws:iam::123456789012:policy/test\n")

		Expect(ExtractCommandsToDeleteAWSResources(output)).To(Equal([]string{
			"aws iam delete-role --role-name test-role",
			"aws iam delete-policy --policy-arn arn:aws:iam::123456789012:policy/test",
		}))
	})

	It("skips the reporter preamble instead of returning it as a command", func() {
		output := bytes.Buffer{}
		output.WriteString(
			"INFO: Run the following commands to delete the Operator roles and policies:\n\n" +
				"aws iam delete-role" + paramSep + "--role-name test-role\n")

		Expect(ExtractCommandsToDeleteAWSResources(output)).To(Equal([]string{
			"aws iam delete-role --role-name test-role",
		}))
	})

	It("skips log output that trails the last command", func() {
		output := bytes.Buffer{}
		output.WriteString(
			"aws iam delete-role" + paramSep + "--role-name test-role\n" +
				"WARN: If policies created are not attached, try re-running with force-policy-creation\n")

		Expect(ExtractCommandsToDeleteAWSResources(output)).To(Equal([]string{
			"aws iam delete-role --role-name test-role",
		}))
	})

	It("keeps non-iam commands emitted when deleting an unmanaged oidc-config", func() {
		output := bytes.Buffer{}
		output.WriteString(
			"aws secretsmanager delete-secret" + paramSep +
				"--region us-east-1" + paramSep +
				"--secret-id arn:aws:secretsmanager:us-east-1:123456789012:secret:rosa-key\n\n" +
				"aws s3 rm s3://test-oidc-bucket" + paramSep +
				"--recursive\n\n" +
				"aws s3 rb s3://test-oidc-bucket\n")

		Expect(ExtractCommandsToDeleteAWSResources(output)).To(Equal([]string{
			"aws secretsmanager delete-secret --region us-east-1 " +
				"--secret-id arn:aws:secretsmanager:us-east-1:123456789012:secret:rosa-key",
			"aws s3 rm s3://test-oidc-bucket --recursive",
			"aws s3 rb s3://test-oidc-bucket",
		}))
	})

	It("handles a single-line command with no wrapped params", func() {
		output := bytes.Buffer{}
		output.WriteString("aws s3 rb s3://test-oidc-bucket\naws s3 rb s3://other-bucket\n")

		Expect(ExtractCommandsToDeleteAWSResources(output)).To(Equal([]string{
			"aws s3 rb s3://test-oidc-bucket",
			"aws s3 rb s3://other-bucket",
		}))
	})

	It("strips the quotes rosacli wraps around param values", func() {
		output := bytes.Buffer{}
		output.WriteString("aws iam delete-role" + paramSep + "--role-name 'test-role'\n")

		Expect(ExtractCommandsToDeleteAWSResources(output)).To(Equal([]string{
			"aws iam delete-role --role-name test-role",
		}))
	})

	It("returns nothing when the output holds no commands", func() {
		output := bytes.Buffer{}
		output.WriteString("INFO: There are no operator roles to delete\n")

		Expect(ExtractCommandsToDeleteAWSResources(output)).To(BeEmpty())
	})
})
