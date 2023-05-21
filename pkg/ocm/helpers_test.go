package ocm

import (
	"fmt"
	"net/url"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var _ = Describe("Http tokens", func() {
	Context("Http tokens variable validations", func() {
		It("OK: Validates successfully http tokens required", func() {
			err := ValidateHttpTokensValue(string(cmv1.Ec2MetadataHttpTokensRequired))
			Expect(err).NotTo(HaveOccurred())
		})
		It("OK: Validates successfully http tokens optional", func() {
			err := ValidateHttpTokensValue(string(cmv1.Ec2MetadataHttpTokensOptional))
			Expect(err).NotTo(HaveOccurred())
		})
		It("OK: Validates successfully http tokens empty string", func() {
			err := ValidateHttpTokensValue("")
			Expect(err).NotTo(HaveOccurred())
		})
		It("Error: Validates error for http tokens bad string", func() {
			err := ValidateHttpTokensValue("dummy")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(fmt.Sprintf("ec2-metadata-http-tokens value should be one of '%s', '%s'",
				cmv1.Ec2MetadataHttpTokensRequired, cmv1.Ec2MetadataHttpTokensOptional)))
		})
	})
})

var _ = Describe("Validate Issuer Url Matches Assume Policy Document", func() {
	const (
		fakeOperatorRoleArn = "arn:aws:iam::765374464689:role/fake-arn-openshift-cluster-csi-drivers-ebs-cloud-credentials"
	)
	It("OK: Matching", func() {
		//nolint
		fakeAssumePolicyDocument := `%7B%22Version%22%3A%222012-10-17%22%2C%22Statement%22%3A%5B%7B%22Effect%22%3A%22Allow%22%2C%22Principal%22%3A%7B%22Federated%22%3A%22arn%3Aaws%3Aiam%3A%3A765374464689%3Aoidc-provider%2Ffake-oidc.s3.us-east-1.amazonaws.com%22%7D%2C%22Action%22%3A%22sts%3AAssumeRoleWithWebIdentity%22%2C%22Condition%22%3A%7B%22StringEquals%22%3A%7B%22fake.s3.us-east-1.amazonaws.com%3Asub%22%3A%5B%22system%3Aserviceaccount%3Aopenshift-image-registry%3Acluster-image-registry-operator%22%2C%22system%3Aserviceaccount%3Aopenshift-image-registry%3Aregistry%22%5D%7D%7D%7D%5D%7D`
		parsedUrl, _ := url.Parse("https://fake-oidc.s3.us-east-1.amazonaws.com")
		err := validateIssuerUrlMatchesAssumePolicyDocument(
			fakeOperatorRoleArn, parsedUrl, fakeAssumePolicyDocument)
		Expect(err).NotTo(HaveOccurred())
	})
	It("OK: Matching with path", func() {
		//nolint
		fakeAssumePolicyDocument := `%7B%22Version%22%3A%222012-10-17%22%2C%22Statement%22%3A%5B%7B%22Effect%22%3A%22Allow%22%2C%22Principal%22%3A%7B%22Federated%22%3A%22arn%3Aaws%3Aiam%3A%3A765374464689%3Aoidc-provider%2Ffake-oidc.s3.us-east-1.amazonaws.com%2F23g84jr4cdfpej0ghlr4teqiog8747gt%22%7D%2C%22Action%22%3A%22sts%3AAssumeRoleWithWebIdentity%22%2C%22Condition%22%3A%7B%22StringEquals%22%3A%7B%22fake.s3.us-east-1.amazonaws.com%2F23g84jr4cdfpej0ghlr4teqiog8747gt%3Asub%22%3A%5B%22system%3Aserviceaccount%3Aopenshift-image-registry%3Acluster-image-registry-operator%22%2C%22system%3Aserviceaccount%3Aopenshift-image-registry%3Aregistry%22%5D%7D%7D%7D%5D%7D`
		parsedUrl, _ := url.Parse("https://fake-oidc.s3.us-east-1.amazonaws.com/23g84jr4cdfpej0ghlr4teqiog8747gt")
		err := validateIssuerUrlMatchesAssumePolicyDocument(
			fakeOperatorRoleArn, parsedUrl, fakeAssumePolicyDocument)
		Expect(err).NotTo(HaveOccurred())
	})
	It("KO: Not matching", func() {
		//nolint
		fakeAssumePolicyDocument := `%7B%22Version%22%3A%222012-10-17%22%2C%22Statement%22%3A%5B%7B%22Effect%22%3A%22Allow%22%2C%22Principal%22%3A%7B%22Federated%22%3A%22arn%3Aaws%3Aiam%3A%3A765374464689%3Aoidc-provider%2Ffake-oidc.s3.us-east-1.amazonaws.com%22%7D%2C%22Action%22%3A%22sts%3AAssumeRoleWithWebIdentity%22%2C%22Condition%22%3A%7B%22StringEquals%22%3A%7B%22fake.s3.us-east-1.amazonaws.com%3Asub%22%3A%5B%22system%3Aserviceaccount%3Aopenshift-image-registry%3Acluster-image-registry-operator%22%2C%22system%3Aserviceaccount%3Aopenshift-image-registry%3Aregistry%22%5D%7D%7D%7D%5D%7D`
		fakeIssuerUrl := "https://fake-oidc-2.s3.us-east-1.amazonaws.com"
		parsedUrl, _ := url.Parse(fakeIssuerUrl)
		err := validateIssuerUrlMatchesAssumePolicyDocument(
			fakeOperatorRoleArn, parsedUrl, fakeAssumePolicyDocument)
		Expect(err).To(HaveOccurred())
		//nolint
		Expect(fmt.Sprintf("Operator role '%s' does not have trusted relationship to '%s' issuer URL", fakeOperatorRoleArn, parsedUrl.Host)).To(Equal(err.Error()))
	})
	It("KO: Not matching with path", func() {
		//nolint
		fakeAssumePolicyDocument := `%7B%22Version%22%3A%222012-10-17%22%2C%22Statement%22%3A%5B%7B%22Effect%22%3A%22Allow%22%2C%22Principal%22%3A%7B%22Federated%22%3A%22arn%3Aaws%3Aiam%3A%3A765374464689%3Aoidc-provider%2Ffake-oidc.s3.us-east-1.amazonaws.com%2F23g84jr4cdfpej0ghlr4teqiog8747gt%22%7D%2C%22Action%22%3A%22sts%3AAssumeRoleWithWebIdentity%22%2C%22Condition%22%3A%7B%22StringEquals%22%3A%7B%22fake.s3.us-east-1.amazonaws.com%2F23g84jr4cdfpej0ghlr4teqiog8747gt%3Asub%22%3A%5B%22system%3Aserviceaccount%3Aopenshift-image-registry%3Acluster-image-registry-operator%22%2C%22system%3Aserviceaccount%3Aopenshift-image-registry%3Aregistry%22%5D%7D%7D%7D%5D%7D`
		fakeIssuerUrl := "https://fake-oidc-2.s3.us-east-1.amazonaws.com/23g84jr4cdfpej0ghlr4teqiog8747g"
		parsedUrl, _ := url.Parse(fakeIssuerUrl)
		err := validateIssuerUrlMatchesAssumePolicyDocument(
			fakeOperatorRoleArn, parsedUrl, fakeAssumePolicyDocument)
		Expect(err).To(HaveOccurred())
		//nolint
		Expect(fmt.Sprintf("Operator role '%s' does not have trusted relationship to '%s' issuer URL", fakeOperatorRoleArn, parsedUrl.Host+parsedUrl.Path)).To(Equal(err.Error()))
	})
})
