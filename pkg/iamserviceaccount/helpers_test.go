/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package iamserviceaccount

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IAM Service Account Helpers", func() {
	Context("GenerateRoleName", func() {
		It("should generate a valid role name", func() {
			roleName := GenerateRoleName("my-cluster", "default", "my-app")
			Expect(roleName).To(Equal("my-cluster-default-my-app-role"))
		})

		It("should truncate long role names", func() {
			longCluster := "very-long-cluster-name-that-exceeds-normal-length"
			roleName := GenerateRoleName(longCluster, "default", "my-app")
			Expect(len(roleName)).To(BeNumerically("<=", 64))
			Expect(roleName).To(HaveSuffix("-default-my-app-role"))
		})

		It("should handle extremely long names", func() {
			longCluster := "extremely-long-cluster-name-that-definitely-exceeds-all-reasonable-length-limits"
			longNamespace := "extremely-long-namespace-name"
			longServiceAccount := "extremely-long-service-account-name"

			roleName := GenerateRoleName(longCluster, longNamespace, longServiceAccount)
			Expect(len(roleName)).To(BeNumerically("<=", 64))
			Expect(roleName).To(ContainSubstring("rosa"))
		})
	})

	Context("GenerateTrustPolicy", func() {
		It("should generate a valid trust policy", func() {
			oidcProviderARN := "arn:aws:iam::123456789012:oidc-provider/rh-oidc.s3.us-east-1.amazonaws.com/1234567890abcdef"
			trustPolicy := GenerateTrustPolicy(oidcProviderARN, "default", "my-app")

			Expect(trustPolicy).To(ContainSubstring(oidcProviderARN))
			Expect(trustPolicy).To(ContainSubstring("system:serviceaccount:default:my-app"))
			Expect(trustPolicy).To(ContainSubstring("sts:AssumeRoleWithWebIdentity"))
			Expect(trustPolicy).To(ContainSubstring("rh-oidc.s3.us-east-1.amazonaws.com/1234567890abcdef"))
		})

		It("should handle invalid ARN gracefully", func() {
			invalidARN := "invalid-arn"
			trustPolicy := GenerateTrustPolicy(invalidARN, "default", "my-app")
			Expect(trustPolicy).To(BeEmpty())
		})

		It("should extract OIDC provider URL from ARN correctly", func() {
			oidcProviderARN := "arn:aws:iam::123456789012:oidc-provider/example.com/path/to/provider"
			trustPolicy := GenerateTrustPolicy(oidcProviderARN, "kube-system", "test-sa")

			Expect(trustPolicy).To(ContainSubstring("example.com/path/to/provider"))
			Expect(trustPolicy).To(ContainSubstring("system:serviceaccount:kube-system:test-sa"))
		})
	})

	Context("ValidateServiceAccountName", func() {
		It("should accept valid service account names", func() {
			validNames := []string{
				"my-app",
				"test-service",
				"app123",
				"my.service.account",
				"a",
				"test-app-123",
			}

			for _, name := range validNames {
				err := ValidateServiceAccountName(name)
				Expect(err).ToNot(HaveOccurred(), "Failed for valid name: %s", name)
			}
		})

		It("should reject invalid service account names", func() {
			invalidNames := []string{
				"",        // empty
				"My-App",  // uppercase
				"-my-app", // starts with dash
				"my-app-", // ends with dash
				"my_app",  // underscore
				"my app",  // space
				"my@app",  // special char
			}

			for _, name := range invalidNames {
				err := ValidateServiceAccountName(name)
				Expect(err).To(HaveOccurred(), "Should have failed for invalid name: %s", name)
			}
		})

		It("should reject names that are too long", func() {
			longName := "a" + string(make([]byte, 253)) // 254 characters
			err := ValidateServiceAccountName(longName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot be longer than 253 characters"))
		})
	})

	Context("ValidateNamespaceName", func() {
		It("should accept valid namespace names", func() {
			validNames := []string{
				"default",
				"my-namespace",
				"test123",
				"app-prod",
				"production",
				"a",
			}

			for _, name := range validNames {
				err := ValidateNamespaceName(name)
				Expect(err).ToNot(HaveOccurred(), "Failed for valid name: %s", name)
			}
		})

		It("should reject invalid namespace names", func() {
			invalidNames := []string{
				"",             // empty
				"My-Namespace", // uppercase
				"-namespace",   // starts with dash
				"namespace-",   // ends with dash
				"name_space",   // underscore
				"name space",   // space
				"name.space",   // dot
				"name@space",   // special char
			}

			for _, name := range invalidNames {
				err := ValidateNamespaceName(name)
				Expect(err).To(HaveOccurred(), "Should have failed for invalid name: %s", name)
			}
		})

		It("should reject reserved namespace names", func() {
			reservedNames := []string{
				"kube-system",
				"kube-public",
				"kube-node-lease",
			}

			for _, name := range reservedNames {
				err := ValidateNamespaceName(name)
				Expect(err).To(HaveOccurred(), "Should have failed for reserved name: %s", name)
				Expect(err.Error()).To(ContainSubstring("reserved"))
			}
		})

		It("should reject names that are too long", func() {
			longName := strings.Repeat("a", 64) // 64 characters
			err := ValidateNamespaceName(longName)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot be longer than 63 characters"))
		})
	})

	Context("GenerateDefaultTags", func() {
		It("should generate correct default tags", func() {
			tags := GenerateDefaultTags("my-cluster", "default", "my-app")

			Expect(tags).To(HaveKeyWithValue(RoleTypeTagKey, ServiceAccountRoleType))
			Expect(tags).To(HaveKeyWithValue(ClusterTagKey, "my-cluster"))
			Expect(tags).To(HaveKeyWithValue(NamespaceTagKey, "default"))
			Expect(tags).To(HaveKeyWithValue(ServiceAccountTagKey, "my-app"))
			Expect(tags).To(HaveKeyWithValue("red-hat-managed", "true"))
		})

		It("should handle special characters in values", func() {
			tags := GenerateDefaultTags("cluster-with-dashes", "ns.with.dots", "sa_with_underscores")

			Expect(tags).To(HaveKeyWithValue(ClusterTagKey, "cluster-with-dashes"))
			Expect(tags).To(HaveKeyWithValue(NamespaceTagKey, "ns.with.dots"))
			Expect(tags).To(HaveKeyWithValue(ServiceAccountTagKey, "sa_with_underscores"))
		})
	})

	Context("GetRoleARN", func() {
		It("should construct correct role ARN with default path", func() {
			arn := GetRoleARN("123456789012", "my-role", "", "aws")
			expectedARN := "arn:aws:iam::123456789012:role/my-role"
			Expect(arn).To(Equal(expectedARN))
		})

		It("should construct correct role ARN with custom path", func() {
			arn := GetRoleARN("123456789012", "my-role", "/rosa/", "aws")
			expectedARN := "arn:aws:iam::123456789012:role/rosa/my-role"
			Expect(arn).To(Equal(expectedARN))
		})

		It("should handle different partitions", func() {
			arn := GetRoleARN("123456789012", "my-role", "/", "aws-us-gov")
			expectedARN := "arn:aws-us-gov:iam::123456789012:role/my-role"
			Expect(arn).To(Equal(expectedARN))
		})

		It("should handle empty path correctly", func() {
			arn := GetRoleARN("123456789012", "my-role", "", "aws")
			Expect(arn).To(ContainSubstring("role/my-role"))
			Expect(arn).ToNot(ContainSubstring("role//"))
		})
	})

	Context("GenerateTrustPolicyMultiple", func() {
		It("should generate trust policy for single service account", func() {
			oidcProviderARN := "arn:aws:iam::123456789012:oidc-provider/rh-oidc.s3.us-east-1.amazonaws.com/1234567890abcdef"
			serviceAccounts := []ServiceAccountIdentifier{
				{Name: "my-app", Namespace: "default"},
			}

			trustPolicy := GenerateTrustPolicyMultiple(oidcProviderARN, serviceAccounts)

			Expect(trustPolicy).To(ContainSubstring(oidcProviderARN))
			Expect(trustPolicy).To(ContainSubstring("sts:AssumeRoleWithWebIdentity"))
			// For single service account, should use string format
			Expect(trustPolicy).To(ContainSubstring(`"system:serviceaccount:default:my-app"`))
			Expect(trustPolicy).ToNot(ContainSubstring(`["system:serviceaccount:default:my-app"]`))
		})

		It("should generate trust policy for multiple service accounts", func() {
			oidcProviderARN := "arn:aws:iam::123456789012:oidc-provider/rh-oidc.s3.us-east-1.amazonaws.com/1234567890abcdef"
			serviceAccounts := []ServiceAccountIdentifier{
				{Name: "aws-load-balancer-operator-controller-manager", Namespace: "aws-load-balancer-operator"},
				{Name: "aws-load-balancer-controller-cluster", Namespace: "aws-load-balancer-operator"},
			}

			trustPolicy := GenerateTrustPolicyMultiple(oidcProviderARN, serviceAccounts)

			Expect(trustPolicy).To(ContainSubstring(oidcProviderARN))
			Expect(trustPolicy).To(ContainSubstring("sts:AssumeRoleWithWebIdentity"))
			// For multiple service accounts, should use array format
			Expect(trustPolicy).To(ContainSubstring(`["system:serviceaccount:aws-load-balancer-operator:aws-load-balancer-operator-controller-manager", "system:serviceaccount:aws-load-balancer-operator:aws-load-balancer-controller-cluster"]`))
		})

		It("should handle empty service accounts gracefully", func() {
			oidcProviderARN := "arn:aws:iam::123456789012:oidc-provider/rh-oidc.s3.us-east-1.amazonaws.com/1234567890abcdef"
			serviceAccounts := []ServiceAccountIdentifier{}

			trustPolicy := GenerateTrustPolicyMultiple(oidcProviderARN, serviceAccounts)

			// Should still generate a valid policy structure with empty subjects
			Expect(trustPolicy).To(ContainSubstring(oidcProviderARN))
			Expect(trustPolicy).To(ContainSubstring("[]"))
		})

		It("should handle invalid ARN gracefully", func() {
			invalidARN := "invalid-arn"
			serviceAccounts := []ServiceAccountIdentifier{
				{Name: "my-app", Namespace: "default"},
			}

			trustPolicy := GenerateTrustPolicyMultiple(invalidARN, serviceAccounts)
			Expect(trustPolicy).To(BeEmpty())
		})

		It("should properly format multiple service accounts in different namespaces", func() {
			oidcProviderARN := "arn:aws:iam::123456789012:oidc-provider/example.com/oidc"
			serviceAccounts := []ServiceAccountIdentifier{
				{Name: "service1", Namespace: "namespace1"},
				{Name: "service2", Namespace: "namespace2"},
				{Name: "service3", Namespace: "namespace3"},
			}

			trustPolicy := GenerateTrustPolicyMultiple(oidcProviderARN, serviceAccounts)

			Expect(trustPolicy).To(ContainSubstring(`["system:serviceaccount:namespace1:service1", "system:serviceaccount:namespace2:service2", "system:serviceaccount:namespace3:service3"]`))
		})

		It("should maintain backwards compatibility with GenerateTrustPolicy", func() {
			oidcProviderARN := "arn:aws:iam::123456789012:oidc-provider/rh-oidc.s3.us-east-1.amazonaws.com/1234567890abcdef"
			namespace := "default"
			serviceAccountName := "my-app"

			// Generate trust policy using both methods
			singlePolicy := GenerateTrustPolicy(oidcProviderARN, namespace, serviceAccountName)
			multiplePolicy := GenerateTrustPolicyMultiple(oidcProviderARN, []ServiceAccountIdentifier{
				{Name: serviceAccountName, Namespace: namespace},
			})

			// They should produce identical output for single service account
			Expect(multiplePolicy).To(Equal(singlePolicy))
		})
	})
})
