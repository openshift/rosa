/**
Copyright (c) 2023 Red Hat, Inc.

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

package ocm

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/aws/aws-sdk-go-v2/aws"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocmCommonValidations "github.com/openshift-online/ocm-common/pkg/ocm/validations"
	commonUtils "github.com/openshift-online/ocm-common/pkg/utils"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	ocmerrors "github.com/openshift-online/ocm-sdk-go/errors"

	mock "github.com/openshift/rosa/pkg/aws"
)

var _ = Describe("Error Handler", func() {
	sendError := fmt.Errorf("response errored")
	It("Populates error message from ocm response", func() {
		now := time.Now().UTC()
		ocmError, err := ocmerrors.NewError().
			Code("404").
			OperationID("123").
			Reason("test").
			Timestamp(&now).
			Build()
		Expect(err).NotTo(HaveOccurred())
		err = handleErr(ocmError, sendError)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(ocmError.Error()))
	})
	It("Populates error message from send error", func() {
		now := time.Now().UTC()
		ocmError, err := ocmerrors.NewError().
			Code("404").
			OperationID("123").
			Reason("").
			Timestamp(&now).
			Build()
		Expect(err).NotTo(HaveOccurred())
		err = handleErr(ocmError, sendError)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal(sendError.Error()))
	})
})

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
		err := ocmCommonValidations.ValidateIssuerUrlMatchesAssumePolicyDocument(
			fakeOperatorRoleArn, parsedUrl, fakeAssumePolicyDocument)
		Expect(err).NotTo(HaveOccurred())
	})
	It("OK: Matching with path", func() {
		//nolint
		fakeAssumePolicyDocument := `%7B%22Version%22%3A%222012-10-17%22%2C%22Statement%22%3A%5B%7B%22Effect%22%3A%22Allow%22%2C%22Principal%22%3A%7B%22Federated%22%3A%22arn%3Aaws%3Aiam%3A%3A765374464689%3Aoidc-provider%2Ffake-oidc.s3.us-east-1.amazonaws.com%2F23g84jr4cdfpej0ghlr4teqiog8747gt%22%7D%2C%22Action%22%3A%22sts%3AAssumeRoleWithWebIdentity%22%2C%22Condition%22%3A%7B%22StringEquals%22%3A%7B%22fake.s3.us-east-1.amazonaws.com%2F23g84jr4cdfpej0ghlr4teqiog8747gt%3Asub%22%3A%5B%22system%3Aserviceaccount%3Aopenshift-image-registry%3Acluster-image-registry-operator%22%2C%22system%3Aserviceaccount%3Aopenshift-image-registry%3Aregistry%22%5D%7D%7D%7D%5D%7D`
		parsedUrl, _ := url.Parse("https://fake-oidc.s3.us-east-1.amazonaws.com/23g84jr4cdfpej0ghlr4teqiog8747gt")
		err := ocmCommonValidations.ValidateIssuerUrlMatchesAssumePolicyDocument(
			fakeOperatorRoleArn, parsedUrl, fakeAssumePolicyDocument)
		Expect(err).NotTo(HaveOccurred())
	})
	It("KO: Not matching", func() {
		//nolint
		fakeAssumePolicyDocument := `%7B%22Version%22%3A%222012-10-17%22%2C%22Statement%22%3A%5B%7B%22Effect%22%3A%22Allow%22%2C%22Principal%22%3A%7B%22Federated%22%3A%22arn%3Aaws%3Aiam%3A%3A765374464689%3Aoidc-provider%2Ffake-oidc.s3.us-east-1.amazonaws.com%22%7D%2C%22Action%22%3A%22sts%3AAssumeRoleWithWebIdentity%22%2C%22Condition%22%3A%7B%22StringEquals%22%3A%7B%22fake.s3.us-east-1.amazonaws.com%3Asub%22%3A%5B%22system%3Aserviceaccount%3Aopenshift-image-registry%3Acluster-image-registry-operator%22%2C%22system%3Aserviceaccount%3Aopenshift-image-registry%3Aregistry%22%5D%7D%7D%7D%5D%7D`
		fakeIssuerUrl := "https://fake-oidc-2.s3.us-east-1.amazonaws.com"
		parsedUrl, _ := url.Parse(fakeIssuerUrl)
		err := ocmCommonValidations.ValidateIssuerUrlMatchesAssumePolicyDocument(
			fakeOperatorRoleArn, parsedUrl, fakeAssumePolicyDocument)
		Expect(err).To(HaveOccurred())
		//nolint
		Expect(
			fmt.Sprintf(
				"Operator role '%s' does not have trusted relationship to '%s' issuer URL",
				fakeOperatorRoleArn,
				parsedUrl.Host,
			),
		).To(Equal(err.Error()))
	})
	It("KO: Not matching with path", func() {
		//nolint
		fakeAssumePolicyDocument := `%7B%22Version%22%3A%222012-10-17%22%2C%22Statement%22%3A%5B%7B%22Effect%22%3A%22Allow%22%2C%22Principal%22%3A%7B%22Federated%22%3A%22arn%3Aaws%3Aiam%3A%3A765374464689%3Aoidc-provider%2Ffake-oidc.s3.us-east-1.amazonaws.com%2F23g84jr4cdfpej0ghlr4teqiog8747gt%22%7D%2C%22Action%22%3A%22sts%3AAssumeRoleWithWebIdentity%22%2C%22Condition%22%3A%7B%22StringEquals%22%3A%7B%22fake.s3.us-east-1.amazonaws.com%2F23g84jr4cdfpej0ghlr4teqiog8747gt%3Asub%22%3A%5B%22system%3Aserviceaccount%3Aopenshift-image-registry%3Acluster-image-registry-operator%22%2C%22system%3Aserviceaccount%3Aopenshift-image-registry%3Aregistry%22%5D%7D%7D%7D%5D%7D`
		fakeIssuerUrl := "https://fake-oidc-2.s3.us-east-1.amazonaws.com/23g84jr4cdfpej0ghlr4teqiog8747g"
		parsedUrl, _ := url.Parse(fakeIssuerUrl)
		err := ocmCommonValidations.ValidateIssuerUrlMatchesAssumePolicyDocument(
			fakeOperatorRoleArn, parsedUrl, fakeAssumePolicyDocument)
		Expect(err).To(HaveOccurred())
		//nolint
		Expect(
			fmt.Sprintf(
				"Operator role '%s' does not have trusted relationship to '%s' issuer URL",
				fakeOperatorRoleArn,
				parsedUrl.Host+parsedUrl.Path,
			),
		).To(Equal(err.Error()))
	})
})

var _ = Describe("ParseDiskSizeToGigibyte", func() {
	It("returns an error for invalid unit: 1foo", func() {
		size := "1foo"
		_, err := ParseDiskSizeToGigibyte(size)
		Expect(err).To(HaveOccurred())
	})

	It("returns 0 for valid unit: 0", func() {
		size := "0"
		got, err := ParseDiskSizeToGigibyte(size)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(0))
	})

	It("returns 0 for invalid unit no suffix: 1 but return 0", func() {
		size := "0"
		got, err := ParseDiskSizeToGigibyte(size)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(0))
	})

	It("returns an error for invalid unit: 1K", func() {
		size := "1K"
		_, err := ParseDiskSizeToGigibyte(size)
		Expect(err).To(HaveOccurred())
	})

	It("returns an error for invalid unit: 1KiB", func() {
		size := "1KiB"
		_, err := ParseDiskSizeToGigibyte(size)
		Expect(err).To(HaveOccurred())
	})

	It("returns an error for invalid unit: 1 MiB", func() {
		size := "1 MiB"
		_, err := ParseDiskSizeToGigibyte(size)
		Expect(err).To(HaveOccurred())
	})

	It("returns an error for invalid unit: 1 mib", func() {
		size := "1 mib"
		_, err := ParseDiskSizeToGigibyte(size)
		Expect(err).To(HaveOccurred())
	})

	It("returns 0 for invalid unit: 0 GiB", func() {
		size := "0 GiB"
		got, err := ParseDiskSizeToGigibyte(size)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(0))
	})

	It("returns the correct value for valid unit: 100 G", func() {
		size := "100 G"
		got, err := ParseDiskSizeToGigibyte(size)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(93))
	})

	It("returns the correct value for valid unit: 100GB", func() {
		size := "100GB"
		got, err := ParseDiskSizeToGigibyte(size)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(93))
	})

	It("returns the correct value for valid unit: 100Gb", func() {
		size := "100Gb"
		got, err := ParseDiskSizeToGigibyte(size)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(93))
	})

	It("returns the correct value for valid unit: 100g", func() {
		size := "100g"
		got, err := ParseDiskSizeToGigibyte(size)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(93))
	})

	It("returns the correct value for valid unit: 100GiB", func() {
		size := "100GiB"
		got, err := ParseDiskSizeToGigibyte(size)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(100))
	})

	//
	It("returns the correct value for valid unit: 100gib", func() {
		size := "100gib"
		got, err := ParseDiskSizeToGigibyte(size)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(100))
	})

	It("returns the correct value for valid unit: 100 gib", func() {
		size := "100 gib"
		got, err := ParseDiskSizeToGigibyte(size)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(100))
	})

	It("returns the correct value for valid unit: 100 TB", func() {
		size := "100 TB"
		got, err := ParseDiskSizeToGigibyte(size)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(93132))
	})

	It("returns the correct value for valid unit: 100 T ", func() {
		size := "100 T "
		got, err := ParseDiskSizeToGigibyte(size)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(93132))
	})

	It("returns the correct value for valid unit: 1000 Ti", func() {
		size := "1000 Ti"
		got, err := ParseDiskSizeToGigibyte(size)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(1024000))
	})

	It("returns the correct value for valid unit: empty string", func() {
		size := ""
		got, err := ParseDiskSizeToGigibyte(size)
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(Equal(0))
	})

	It("returns the correct value for valid unit: -1", func() {
		size := "-1"
		got, err := ParseDiskSizeToGigibyte(size)
		Expect(err).To(HaveOccurred())
		Expect(got).To(Equal(0))
	})

	It("returns the correct value for valid unit: 200000000000000 Ti", func() {
		// Hitting the max int64 value
		size := "200000000000000 Ti"
		got, err := ParseDiskSizeToGigibyte(size)
		Expect(err).To(HaveOccurred())
		Expect(got).To(Equal(0))
	})

	It("returns the correct value for valid unit: 200000000000000000000Ti", func() {
		// Hitting the max int64 value
		size := "200000000000000000000Ti"
		got, err := ParseDiskSizeToGigibyte(size)
		Expect(err).To(HaveOccurred())
		Expect(got).To(Equal(0))
	})

	It("returns the correct value for valid unit: -200000000000000000000Ti", func() {
		// Hitting the max int64 value
		size := "-200000000000000000000Ti"
		got, err := ParseDiskSizeToGigibyte(size)
		Expect(err).To(HaveOccurred())
		Expect(got).To(Equal(0))
	})

})

var _ = Describe("ParseVersion", func() {
	It("returns proper value", func() {
		output, err := ParseVersion("4.12")
		Expect(err).To(BeNil())
		Expect(output).To(Equal("4.12"))
	})

	It("fails with malformed version", func() {
		output, err := ParseVersion("3$%TRD")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("Malformed version: 3$%TRD"))
		Expect(output).To(Equal(""))
	})
})

var _ = Describe("ValidateBalancingIgnoredLabels", func() {
	It("returns an error if didn't got a string", func() {
		var val interface{} = 1
		err := ValidateBalancingIgnoredLabels(val)
		Expect(err).To(HaveOccurred())
	})

	It("passes for an empty string", func() {
		var val interface{} = ""
		err := ValidateBalancingIgnoredLabels(val)
		Expect(err).ToNot(HaveOccurred())
	})

	It("passes for valid label keys", func() {
		var val interface{} = "eks.amazonaws.com/nodegroup,alpha.eksctl.io/nodegroup-name"
		err := ValidateBalancingIgnoredLabels(val)
		Expect(err).ToNot(HaveOccurred())
	})

	It("returns an error for a label that doesn't start with an alphanumeric character", func() {
		var val interface{} = ".t"
		err := ValidateBalancingIgnoredLabels(val)
		Expect(err).To(HaveOccurred())
	})

	It("returns an error for a label that has illegal characters", func() {
		var val interface{} = "a%"
		err := ValidateBalancingIgnoredLabels(val)
		Expect(err).To(HaveOccurred())
	})

	It("returns an error for a label that exceeds 63 characters", func() {
		var val interface{} = strings.Repeat("a", commonUtils.MaxByteSize)
		err := ValidateBalancingIgnoredLabels(val)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("expectedSubnetsCount", func() {
	When("multiAZ and privateLink are true", func() {
		It("Should return privateLinkMultiAZSubnetsCount", func() {
			Expect(expectedSubnetsCount(true, true)).To(Equal(privateLinkMultiAZSubnetsCount))
		})
	})

	When("multiAZ is true and privateLink is false", func() {
		It("Should return privateLinkSingleAZSubnetsCount", func() {
			Expect(expectedSubnetsCount(true, false)).To(Equal(BYOVPCMultiAZSubnetsCount))
		})
	})

	When("multiAZ is false and privateLink is true", func() {
		It("Should return BYOVPCMultiAZSubnetsCount", func() {
			Expect(expectedSubnetsCount(false, true)).To(Equal(privateLinkSingleAZSubnetsCount))
		})
	})

	When("multiAZ and privateLink are false", func() {
		It("Should return BYOVPCSingleAZSubnetsCount", func() {
			Expect(expectedSubnetsCount(false, false)).To(Equal(BYOVPCSingleAZSubnetsCount))
		})
	})
})

var _ = Describe("ValidateSubnetsCount", func() {
	When("When privateLink is true", func() {
		When("multiAZ is true", func() {
			It("should return an error if subnetsInputCount is not equal to privateLinkMultiAZSubnetsCount", func() {
				err := ValidateSubnetsCount(true, true, privateLinkMultiAZSubnetsCount+1)
				Expect(err).To(HaveOccurred())
				Expect(
					err.Error(),
				).To(Equal(fmt.Sprintf("The number of subnets for a 'multi-AZ' 'private link cluster' should be"+
					" '%d', instead received: '%d'", privateLinkMultiAZSubnetsCount, privateLinkMultiAZSubnetsCount+1)))
			})

			It("should not return an error if subnetsInputCount is equal to privateLinkMultiAZSubnetsCount", func() {
				err := ValidateSubnetsCount(true, true, privateLinkMultiAZSubnetsCount)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("multiAZ is false", func() {
			It("should return an error if subnetsInputCount is not equal to privateLinkSingleAZSubnetsCount", func() {
				err := ValidateSubnetsCount(false, true, privateLinkSingleAZSubnetsCount+1)
				Expect(err).To(HaveOccurred())
				Expect(
					err.Error(),
				).To(Equal(fmt.Sprintf("The number of subnets for a 'single AZ' 'private link cluster' should be"+
					" '%d', instead received: '%d'", privateLinkSingleAZSubnetsCount, privateLinkSingleAZSubnetsCount+1)))
			})

			It("should not return an error if subnetsInputCount is equal to privateLinkSingleAZSubnetsCount", func() {
				err := ValidateSubnetsCount(false, true, privateLinkSingleAZSubnetsCount)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	When("privateLink is false", func() {
		When("multiAZ is true", func() {
			It("should return an error if subnetsInputCount is not equal to BYOVPCMultiAZSubnetsCount", func() {
				err := ValidateSubnetsCount(true, false, BYOVPCMultiAZSubnetsCount+1)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(fmt.Sprintf("The number of subnets for a 'multi-AZ' 'cluster' should be"+
					" '%d', instead received: '%d'", BYOVPCMultiAZSubnetsCount, BYOVPCMultiAZSubnetsCount+1)))
			})

			It("should not return an error if subnetsInputCount is equal to BYOVPCMultiAZSubnetsCount", func() {
				err := ValidateSubnetsCount(true, false, BYOVPCMultiAZSubnetsCount)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("multiAZ is false", func() {
			It("should return an error if subnetsInputCount is not equal to BYOVPCSingleAZSubnetsCount", func() {
				err := ValidateSubnetsCount(false, false, BYOVPCSingleAZSubnetsCount+1)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(fmt.Sprintf("The number of subnets for a 'single AZ' 'cluster' should"+
					" be '%d', instead received: '%d'", BYOVPCSingleAZSubnetsCount, BYOVPCSingleAZSubnetsCount+1)))
			})

			It("should not return an error if subnetsInputCount is equal to BYOVPCSingleAZSubnetsCount", func() {
				err := ValidateSubnetsCount(false, false, BYOVPCSingleAZSubnetsCount)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})

var _ = Describe("ValidateHostedClusterSubnets for Private Cluster", func() {
	var (
		mockClient *mock.MockClient
		ids        = []string{"subnet-public-1", "subnet-private-2"}
		subnets    = []ec2types.Subnet{
			{SubnetId: aws.String("subnet-public-1")},
			{SubnetId: aws.String("subnet-private-2")},
		}
	)
	BeforeEach(func() {
		mockCtrl := gomock.NewController(GinkgoT())
		mockClient = mock.NewMockClient(mockCtrl)
		mockClient.EXPECT().GetVPCSubnets(gomock.Any()).Return(subnets, nil).AnyTimes()
	})
	It("should not return an error when only private subnets are present for a private cluster", func() {
		mockClient.EXPECT().FilterVPCsPrivateSubnets(gomock.Any()).Return([]ec2types.Subnet{subnets[1]}, nil)
		_, err := ValidateHostedClusterSubnets(mockClient, true, []string{"subnet-private-2"}, true)
		Expect(err).NotTo(HaveOccurred())
	})
	It("should return an error when public subnets are present for a private cluster", func() {
		mockClient.EXPECT().FilterVPCsPrivateSubnets(gomock.Any()).Return([]ec2types.Subnet{}, nil)
		_, err := ValidateHostedClusterSubnets(mockClient, true, ids, true)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("The number of public subnets for a private hosted cluster should be zero"))
	})
})

var _ = Describe("IsValidClusterName()", func() {
	DescribeTable("IsValidClusterName() test cases", func(name string, expected bool) {
		valid := IsValidClusterName(name)
		Expect(expected).To(Equal(valid))
	},
		Entry("returns false when an empty name is given", "", false),
		Entry("returns false when name is not a valid DNS label", "9hjh9", false),
		Entry("returns false when name is not a valid DNS label", "hjh-", false),
		Entry("returns false when name is longer than 54 chars", strings.Repeat("h", 55), false),
		Entry("returns true when name valid", strings.Repeat("h", 25), true))
})

var _ = Describe("ClusterNameValidator()", func() {
	DescribeTable("ClusterNameValidator() test cases", func(name interface{}, shouldErr bool) {
		err := ClusterNameValidator(name)
		if shouldErr {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
	},
		Entry("should error when a non string arg is given", 5, true),
		Entry("should error when an empty name is given", "", true),
		Entry("should error when name is not a valid DNS label", "9hjh9", true),
		Entry("should error when name is not a valid DNS label", "hjh-", true),
		Entry("should error when name is longer than 54 chars", strings.Repeat("h", 55), true),
		Entry("should not error when name valid", strings.Repeat("h", 25), false))
})

var _ = Describe("ValidateRegistryAdditionalCa", func() {
	DescribeTable("ValidateRegistryAdditionalCa() test cases", func(input map[string]string, shouldErr bool) {
		err := ValidateRegistryAdditionalCa(input)
		if shouldErr {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
	},
		Entry("should error when the value is not a PEM certificate", map[string]string{
			"registry.io": "abc",
		}, true),
		Entry("should not error when the value is a PEM certificate", map[string]string{
			"registry.io": "-----BEGIN CERTIFICATE-----\n/abc\n-----END CERTIFICATE-----",
		}, false),
	)
})

var _ = Describe("ValidateAllowedRegistriesForImport", func() {
	DescribeTable("ValidateAllowedRegistriesForImport() test case", func(input interface{}, shouldErr bool) {
		err := ValidateAllowedRegistriesForImport(input)
		if shouldErr {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
	},
		Entry("should error when boolean type was passed", true, true),
		Entry("should error when regex doesn't match", "registry.iolala", true),
		Entry("should not error when input with port is valid", "registry.io:80:false", false),
		Entry("should not error when input is valid", "registry.io:true", false),
		Entry("should not error when long input is valid", "registry.io:true,registry2.io:false", false),
		Entry("should not error when long input with port is valid", "registry.io:80:true,registry2.io:90:false", false))

})

var _ = Describe("IsValidClusterDomainPrefix()", func() {
	DescribeTable("IsValidClusterDomainPrefix() test cases", func(domainPrefix string, expected bool) {
		valid := IsValidClusterDomainPrefix(domainPrefix)
		Expect(expected).To(Equal(valid))
	},
		Entry("returns false when an empty domain prefix is given", "", false),
		Entry("returns false when domain prefix is not a valid DNS label", "9hjh9", false),
		Entry("returns false when domain prefix is not a valid DNS label", "hjh-", false),
		Entry("returns false when domain prefix is longer than 15 chars", strings.Repeat("h", 16), false),
		Entry("returns true when domain prefix valid", strings.Repeat("h", 15), true))
})

var _ = Describe("ClusterDomainPrefixValidator()", func() {
	DescribeTable("ClusterDomainPrefixValidator() test case", func(domainPrefix interface{}, shouldErr bool) {
		err := ClusterDomainPrefixValidator(domainPrefix)
		if shouldErr {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
	},
		Entry("should error when a non string arg is given", 5, true),
		Entry("should not error when an empty domain prefix is given", "", false),
		Entry("shoud error when domain prefix is not a valid DNS label", "9hjh9", true),
		Entry("should error when domain prefix is not a valid DNS label", "hjh-", true),
		Entry("should error when domain prefix is longer than 15 chars", strings.Repeat("h", 16), true),
		Entry("should not error when domain prefix valid", strings.Repeat("h", 15), false))
})

var _ = Describe("ValidateClaimValidationRules()", func() {
	DescribeTable("ValidateClaimValidationRules() test case", func(input interface{}, shouldErr bool, errMsg string) {
		err := ValidateClaimValidationRules(input)
		if shouldErr {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(errMsg))
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
	},
		Entry("should error when a non string arg is given", 5, true, "can only validate string types, got int"),
		Entry("should not error when an empty claim validation rule is given", "", false, ""),
		Entry("shoud error when claim validation rule is not a valid value", "9hjh9", true,
			"invalid identifier '9hjh9' for 'claim validation rule. 'Should be in a <claim>:<required_value> format."),
		Entry("should not error when claim validation rule with single pair is valid", "abc:efg", false, ""))
	Entry("should not error when claim validation rule with multiple pairs is valid", "abc:efg,lala:wuwu", false, "")
})
