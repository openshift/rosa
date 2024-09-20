package ocm

import (
	. "github.com/onsi/ginkgo/v2/dsl/core"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("Registry Config", func() {
	Context("BuildAllowedRegistriesForImport", func() {
		It("OK: should pass if the user passes a valid string", func() {
			obj, err := BuildAllowedRegistriesForImport("abc.com:true")
			Expect(err).ToNot(HaveOccurred())
			Expect(obj).To(Equal(map[string]bool{
				"abc.com": true,
			}))

		})
		It("OK: should pass if the user passes a valid long string", func() {
			obj, err := BuildAllowedRegistriesForImport("abc.com:true,efg.io:false,test.com:true")
			Expect(err).ToNot(HaveOccurred())
			Expect(obj).To(Equal(map[string]bool{
				"abc.com":  true,
				"efg.io":   false,
				"test.com": true,
			}))
		})
		It("KO: should fail if the user does not pass the correct regex", func() {
			_, err := BuildAllowedRegistriesForImport("abc")
			Expect(err).To(MatchError("invalid identifier 'abc' for 'allowed registries for import.' " +
				"Should be in a <registry>:<boolean> format. " +
				"The boolean indicates whether the registry is secure or not."))

		})
	})

	Context("BuildRegistryConfig", func() {
		It("Returns expected output", func() {
			spec := Spec{}
			spec.AllowedRegistries = []string{"example.com", "quay.io"}
			spec.BlockedRegistries = []string{"insecure.com"}
			spec.InsecureRegistries = []string{"insecure.com"}

			additionalTrustedCa := map[string]string{}
			additionalTrustedCa["userRegistry.io"] = "-----BEGIN CERTIFICATE-----\n/abc\n-----END CERTIFICATE-----"
			spec.AdditionalTrustedCa = additionalTrustedCa
			spec.PlatformAllowlist = "abc.com"
			spec.AllowedRegistriesForImport = "quay.io:true"

			output, err := BuildRegistryConfig(spec)
			Expect(err).To(Not(HaveOccurred()))
			expectedOutput := cmv1.NewClusterRegistryConfig().
				AdditionalTrustedCa(additionalTrustedCa).
				PlatformAllowlist(cmv1.NewRegistryAllowlist().ID(spec.PlatformAllowlist)).
				AllowedRegistriesForImport(cmv1.NewRegistryLocation().
					Insecure(true).DomainName("quay.io")).
				RegistrySources(cmv1.NewRegistrySources().InsecureRegistries("insecure.com").
					BlockedRegistries("insecure.com").
					AllowedRegistries([]string{"example.com", "quay.io"}...))
			Expect(output).To(Equal(expectedOutput))
		})
	})
})
