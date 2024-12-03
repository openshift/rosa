package network

import (
	"errors"

	cfTypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
)

var _ = Describe("ParseParams", func() {
	It("should correctly parse parameters and user tags", func() {
		params := []string{
			"Key1=Value1",
			"Key2=Value2",
			"Tags=TagKey1=TagValue1,TagKey2=TagValue2",
		}

		expectedResult := map[string]string{
			"Key1": "Value1",
			"Key2": "Value2",
		}

		expectedUserTags := map[string]string{
			"TagKey1": "TagValue1",
			"TagKey2": "TagValue2",
		}

		result, userTags, err := ParseParams(params)

		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(expectedResult))
		Expect(userTags).To(Equal(expectedUserTags))
	})

	It("should handle parameters without user tags", func() {
		params := []string{
			"Key1=Value1",
			"Key2=Value2",
		}

		expectedResult := map[string]string{
			"Key1": "Value1",
			"Key2": "Value2",
		}

		expectedUserTags := map[string]string{}

		result, userTags, err := ParseParams(params)

		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(expectedResult))
		Expect(userTags).To(Equal(expectedUserTags))
	})

	It("should handle empty parameters", func() {
		params := []string{}

		expectedResult := map[string]string{}
		expectedUserTags := map[string]string{}

		result, userTags, err := ParseParams(params)

		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(expectedResult))
		Expect(userTags).To(Equal(expectedUserTags))
	})
})

var _ = Describe("Helper Functions", func() {
	Describe("SelectTemplate", func() {
		It("should return the correct template file path", func() {
			templateName := "test-template"
			templateDir := "cmd/create/network/templates"
			expectedTemplateFile := "cmd/create/network/templates/test-template/cloudformation.yaml"
			Expect(SelectTemplate(templateDir, templateName)).To(Equal(expectedTemplateFile))
		})
	})

	Describe("formatParams", func() {
		It("should format parameters correctly", func() {
			params := map[string]string{
				"Key1": "Value1",
				"Key2": "Value2",
			}
			expectedParamStr := "ParameterKey=Key1,ParameterValue=Value1 ParameterKey=Key2,ParameterValue=Value2 "
			Expect(formatParams(params)).To(Equal(expectedParamStr))
		})
	})

	Describe("formatTags", func() {
		It("should format tags correctly", func() {
			tags := map[string]string{
				"TagKey1": "TagValue1",
				"TagKey2": "TagValue2",
			}
			expectedTagStr := "Key=TagKey1,Value=TagValue1 Key=TagKey2,Value=TagValue2 "
			Expect(formatTags(tags)).To(Equal(expectedTagStr))
		})
	})

	Describe("deleteHelperMessage", func() {
		It("should log the correct error and info messages", func() {
			logger := logrus.New()
			params := map[string]string{
				"Name":   "test-stack",
				"Region": "us-east-1",
			}
			err := errors.New("test error")
			deleteHelperMessage(logger, params, err)
		})
	})

	Describe("ManualModeHelperMessage", func() {
		It("should return the correct manual mode helper message", func() {
			params := map[string]string{
				"Name":   "test-stack",
				"Region": "us-west-1",
			}
			tags := map[string]string{
				"TagKey1": "TagValue1",
				"TagKey2": "TagValue2",
			}
			expectedMessage := "Run the following command to create the stack manually:\n" +
				"aws cloudformation create-stack --stack-name test-stack --template-body file://<template-file-path> " +
				"--param ParameterKey=Name,ParameterValue=test-stack ParameterKey=Region,ParameterValue=us-west-1 " +
				" --tags Key=TagKey1,Value=TagValue1 Key=TagKey2,Value=TagValue2  --region us-west-1"
			Expect(ManualModeHelperMessage(params, tags)).To(Equal(expectedMessage))
		})
	})

	var _ = Describe("getStatusColor", func() {
		It("should return green for create complete and update complete statuses", func() {
			Expect(getStatusColor(cfTypes.ResourceStatusCreateComplete)).To(Equal(ColorGreen))
			Expect(getStatusColor(cfTypes.ResourceStatusUpdateComplete)).To(Equal(ColorGreen))
		})

		It("should return red for create failed, delete failed, and update failed statuses", func() {
			Expect(getStatusColor(cfTypes.ResourceStatusCreateFailed)).To(Equal(ColorRed))
			Expect(getStatusColor(cfTypes.ResourceStatusDeleteFailed)).To(Equal(ColorRed))
			Expect(getStatusColor(cfTypes.ResourceStatusUpdateFailed)).To(Equal(ColorRed))
		})

		It("should return yellow for any other status", func() {
			Expect(getStatusColor(cfTypes.ResourceStatusCreateInProgress)).To(Equal(ColorYellow))
		})
	})
})
