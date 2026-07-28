package aws

import (
	"encoding/json"
	"fmt"

	gomock "go.uber.org/mock/gomock"

	awsSdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	iamtypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/openshift/rosa/pkg/aws/mocks"
)

var _ = Describe("PolicyDocument", func() {

	Describe("NewPolicyDocument", func() {
		It("returns a document with Version 2012-10-17 and empty Statement", func() {
			doc := NewPolicyDocument()
			Expect(doc.Version).To(Equal("2012-10-17"))
			Expect(doc.Statement).To(BeEmpty())
		})
	})

	Describe("ParsePolicyDocument", func() {
		Context("with valid JSON", func() {
			It("parses a simple policy", func() {
				raw := `{
					"Version": "2012-10-17",
					"Statement": [{
						"Effect": "Allow",
						"Action": "s3:GetObject",
						"Resource": "*"
					}]
				}`
				doc, err := ParsePolicyDocument(raw)
				Expect(err).ToNot(HaveOccurred())
				Expect(doc.Version).To(Equal("2012-10-17"))
				Expect(doc.Statement).To(HaveLen(1))
				Expect(doc.Statement[0].Effect).To(Equal("Allow"))
			})

			It("parses a policy with multiple statements", func() {
				raw := `{
					"Version": "2012-10-17",
					"Statement": [
						{"Sid": "AllowS3", "Effect": "Allow", "Action": "s3:GetObject", "Resource": "*"},
						{"Sid": "DenyEC2", "Effect": "Deny", "Action": "ec2:TerminateInstances", "Resource": "*"}
					]
				}`
				doc, err := ParsePolicyDocument(raw)
				Expect(err).ToNot(HaveOccurred())
				Expect(doc.Statement).To(HaveLen(2))
				Expect(doc.Statement[0].Sid).To(Equal("AllowS3"))
				Expect(doc.Statement[1].Sid).To(Equal("DenyEC2"))
			})
		})

		Context("with invalid input", func() {
			It("returns an error for malformed JSON", func() {
				_, err := ParsePolicyDocument(`{not json}`)
				Expect(err).To(HaveOccurred())
			})

			It("returns an error for an empty string", func() {
				_, err := ParsePolicyDocument("")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("GetAWSPrincipals", func() {
		It("returns a single principal when AWS is a string", func() {
			stmt := PolicyStatement{
				Principal: &PolicyStatementPrincipal{
					AWS: "arn:aws:iam::123456:root",
				},
			}
			Expect(stmt.GetAWSPrincipals()).To(Equal([]string{"arn:aws:iam::123456:root"}))
		})

		It("returns all principals when AWS is a slice", func() {
			stmt := PolicyStatement{
				Principal: &PolicyStatementPrincipal{
					AWS: []interface{}{"arn:aws:iam::111:root", "arn:aws:iam::222:root"},
				},
			}
			Expect(stmt.GetAWSPrincipals()).To(Equal([]string{
				"arn:aws:iam::111:root",
				"arn:aws:iam::222:root",
			}))
		})

		It("returns an empty slice when AWS is nil", func() {
			stmt := PolicyStatement{
				Principal: &PolicyStatementPrincipal{AWS: nil},
			}
			Expect(stmt.GetAWSPrincipals()).To(BeEmpty())
		})

		It("handles a single-element slice from unmarshalled JSON", func() {
			raw := `{"Principal":{"AWS":["arn:aws:iam::999:role/foo"]}}`
			var stmt PolicyStatement
			Expect(json.Unmarshal([]byte(raw), &stmt)).To(Succeed())
			Expect(stmt.GetAWSPrincipals()).To(Equal([]string{"arn:aws:iam::999:role/foo"}))
		})

		It("handles a string from unmarshalled JSON", func() {
			raw := `{"Principal":{"AWS":"arn:aws:iam::888:role/bar"}}`
			var stmt PolicyStatement
			Expect(json.Unmarshal([]byte(raw), &stmt)).To(Succeed())
			Expect(stmt.GetAWSPrincipals()).To(Equal([]string{"arn:aws:iam::888:role/bar"}))
		})
	})

	Describe("AllowActions", func() {
		It("creates an Allow statement with Resource *", func() {
			doc := NewPolicyDocument()
			doc.AllowActions("s3:GetObject", "s3:PutObject")

			Expect(doc.Statement).To(HaveLen(1))
			stmt := doc.Statement[0]
			Expect(stmt.Effect).To(Equal("Allow"))
			Expect(stmt.Action).To(Equal([]string{"s3:GetObject", "s3:PutObject"}))
			Expect(stmt.Resource).To(Equal("*"))
		})

		It("appends a new statement on each call", func() {
			doc := NewPolicyDocument()
			doc.AllowActions("s3:GetObject")
			doc.AllowActions("ec2:DescribeInstances")

			Expect(doc.Statement).To(HaveLen(2))
			Expect(doc.Statement[0].Action).To(Equal([]string{"s3:GetObject"}))
			Expect(doc.Statement[1].Action).To(Equal([]string{"ec2:DescribeInstances"}))
		})
	})

	Describe("IsActionAllowed", func() {
		It("returns true when the action is a matching string", func() {
			doc := &PolicyDocument{
				Statement: []PolicyStatement{
					{Effect: "Allow", Action: "s3:GetObject"},
				},
			}
			Expect(doc.IsActionAllowed("s3:GetObject")).To(BeTrue())
		})

		It("returns true when the action is in an array", func() {
			doc := &PolicyDocument{
				Statement: []PolicyStatement{
					{Effect: "Allow", Action: []interface{}{"s3:GetObject", "s3:PutObject"}},
				},
			}
			Expect(doc.IsActionAllowed("s3:PutObject")).To(BeTrue())
		})

		It("returns false when the action is not present", func() {
			doc := &PolicyDocument{
				Statement: []PolicyStatement{
					{Effect: "Allow", Action: "s3:GetObject"},
				},
			}
			Expect(doc.IsActionAllowed("ec2:RunInstances")).To(BeFalse())
		})

		It("skips Deny statements", func() {
			doc := &PolicyDocument{
				Statement: []PolicyStatement{
					{Effect: "Deny", Action: "s3:GetObject"},
				},
			}
			Expect(doc.IsActionAllowed("s3:GetObject")).To(BeFalse())
		})

		It("returns false with an empty document", func() {
			doc := NewPolicyDocument()
			Expect(doc.IsActionAllowed("s3:GetObject")).To(BeFalse())
		})

		It("finds the action across multiple statements", func() {
			doc := &PolicyDocument{
				Statement: []PolicyStatement{
					{Effect: "Deny", Action: "s3:GetObject"},
					{Effect: "Allow", Action: []interface{}{"ec2:RunInstances"}},
				},
			}
			Expect(doc.IsActionAllowed("ec2:RunInstances")).To(BeTrue())
		})
	})

	Describe("GetAllowedActions", func() {
		It("returns all allowed actions from Allow statements", func() {
			doc := &PolicyDocument{
				Statement: []PolicyStatement{
					{Effect: "Allow", Action: "s3:GetObject"},
					{Effect: "Allow", Action: []interface{}{"ec2:RunInstances", "ec2:DescribeInstances"}},
				},
			}
			Expect(doc.GetAllowedActions()).To(Equal([]string{
				"s3:GetObject",
				"ec2:RunInstances",
				"ec2:DescribeInstances",
			}))
		})

		It("skips Deny statements", func() {
			doc := &PolicyDocument{
				Statement: []PolicyStatement{
					{Effect: "Allow", Action: "s3:GetObject"},
					{Effect: "Deny", Action: "s3:DeleteObject"},
				},
			}
			Expect(doc.GetAllowedActions()).To(Equal([]string{"s3:GetObject"}))
		})

		It("handles both string and array Action types", func() {
			doc := &PolicyDocument{
				Statement: []PolicyStatement{
					{Effect: "Allow", Action: "iam:CreateRole"},
					{Effect: "Allow", Action: []interface{}{"iam:DeleteRole"}},
				},
			}
			Expect(doc.GetAllowedActions()).To(Equal([]string{"iam:CreateRole", "iam:DeleteRole"}))
		})

		It("returns empty for an empty document", func() {
			doc := NewPolicyDocument()
			Expect(doc.GetAllowedActions()).To(BeEmpty())
		})
	})

	Describe("InterpolatePolicyDocument", func() {
		It("replaces template variables", func() {
			tmpl := `{"Statement":[{"Resource":"arn:aws:iam::%{account_id}:oidc-provider/%{oidc_provider_arn}"}]}`
			result := InterpolatePolicyDocument("aws", tmpl, map[string]string{
				"account_id":        "123456789012",
				"oidc_provider_arn": "oidc.example.com",
			})
			Expect(result).To(ContainSubstring("123456789012"))
			Expect(result).To(ContainSubstring("oidc.example.com"))
			Expect(result).ToNot(ContainSubstring("%{account_id}"))
			Expect(result).ToNot(ContainSubstring("%{oidc_provider_arn}"))
		})

		It("replaces arn:aws: with the given partition", func() {
			tmpl := `{"Resource":"arn:aws:iam::123456:role/MyRole"}`
			result := InterpolatePolicyDocument("aws", tmpl, nil)
			Expect(result).To(Equal(`{"Resource":"arn:aws:iam::123456:role/MyRole"}`))
		})

		It("converts arn:aws: to arn:aws-us-gov: for GovCloud", func() {
			tmpl := `{"Resource":"arn:aws:iam::123456:role/MyRole"}`
			result := InterpolatePolicyDocument("aws-us-gov", tmpl, nil)
			Expect(result).To(Equal(`{"Resource":"arn:aws-us-gov:iam::123456:role/MyRole"}`))
		})

		It("handles multiple arn:aws: occurrences", func() {
			tmpl := `{"Resource":["arn:aws:s3:::bucket","arn:aws:iam::123:role/R"]}`
			result := InterpolatePolicyDocument("aws-cn", tmpl, nil)
			Expect(result).ToNot(ContainSubstring("arn:aws:"))
			Expect(result).To(ContainSubstring("arn:aws-cn:s3"))
			Expect(result).To(ContainSubstring("arn:aws-cn:iam"))
		})

		It("is a no-op when no replacements are needed", func() {
			tmpl := `{"Effect":"Allow"}`
			result := InterpolatePolicyDocument("aws", tmpl, nil)
			Expect(result).To(Equal(`{"Effect":"Allow"}`))
		})
	})

	Describe("GenerateRolePolicyDoc", func() {
		const policyTemplate = `{
			"Version": "2012-10-17",
			"Statement": [{
				"Effect": "Allow",
				"Principal": {"Federated": "arn:aws:iam::%{oidc_provider_arn}"},
				"Action": "sts:AssumeRoleWithWebIdentity",
				"Condition": {
					"StringEquals": {
						"%{issuer_url}:sub": "system:serviceaccount:%{service_accounts}"
					}
				}
			}]
		}`

		It("generates a policy with correct OIDC provider ARN and issuer URL", func() {
			result, err := GenerateRolePolicyDoc(
				"aws",
				"https://oidc.example.com/path",
				"123456789012",
				"my-namespace:my-sa",
				policyTemplate,
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ContainSubstring("oidc.example.com/path"))
			Expect(result).To(ContainSubstring("123456789012"))
			Expect(result).To(ContainSubstring("my-namespace:my-sa"))
			Expect(result).ToNot(ContainSubstring("%{oidc_provider_arn}"))
			Expect(result).ToNot(ContainSubstring("%{issuer_url}"))
			Expect(result).ToNot(ContainSubstring("%{service_accounts}"))
		})

		It("returns an error for an invalid OIDC endpoint URL", func() {
			_, err := GenerateRolePolicyDoc(
				"aws",
				"not-a-valid-url",
				"123456789012",
				"ns:sa",
				policyTemplate,
			)
			Expect(err).To(HaveOccurred())
		})

		It("applies the partition to arn:aws: in the template", func() {
			result, err := GenerateRolePolicyDoc(
				"aws-us-gov",
				"https://oidc.example.com",
				"123456789012",
				"ns:sa",
				policyTemplate,
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(ContainSubstring("arn:aws-us-gov:iam::"))
			Expect(result).ToNot(ContainSubstring("arn:aws:"))
		})
	})

	Describe("checkPermissionsUsingQueryClient", func() {
		var (
			mockCtrl   *gomock.Controller
			mockIamAPI *mocks.MockIamApiClient
			client     *awsClient
		)

		BeforeEach(func() {
			mockCtrl = gomock.NewController(GinkgoT())
			mockIamAPI = mocks.NewMockIamApiClient(mockCtrl)
			client = &awsClient{
				iamClient: mockIamAPI,
				logger:    NewLoggerWrapper(logrus.New(), nil),
			}
		})

		AfterEach(func() {
			mockCtrl.Finish()
		})

		It("returns true when all actions are allowed", func() {
			doc := &PolicyDocument{
				Statement: []PolicyStatement{
					{Effect: "Allow", Action: []interface{}{"s3:GetObject", "s3:PutObject"}},
				},
			}

			mockIamAPI.EXPECT().SimulatePrincipalPolicy(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&iam.SimulatePrincipalPolicyOutput{
					EvaluationResults: []iamtypes.EvaluationResult{
						{EvalActionName: awsSdk.String("s3:GetObject"), EvalDecision: "allowed"},
						{EvalActionName: awsSdk.String("s3:PutObject"), EvalDecision: "allowed"},
					},
					IsTruncated: false,
				}, nil)

			result, err := doc.checkPermissionsUsingQueryClient(client,
				"arn:aws:iam::123456789012:user/TestUser", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("returns false with failed actions when some are denied", func() {
			doc := &PolicyDocument{
				Statement: []PolicyStatement{
					{Effect: "Allow", Action: []interface{}{"s3:GetObject", "ec2:RunInstances"}},
				},
			}

			mockIamAPI.EXPECT().SimulatePrincipalPolicy(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&iam.SimulatePrincipalPolicyOutput{
					EvaluationResults: []iamtypes.EvaluationResult{
						{EvalActionName: awsSdk.String("s3:GetObject"), EvalDecision: "allowed"},
						{EvalActionName: awsSdk.String("ec2:RunInstances"), EvalDecision: "implicitDeny"},
					},
					IsTruncated: false,
				}, nil)

			result, err := doc.checkPermissionsUsingQueryClient(client,
				"arn:aws:iam::123456789012:user/TestUser", nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ec2:RunInstances"))
			Expect(result).To(BeFalse())
		})

		It("returns error when SimulatePrincipalPolicy fails", func() {
			doc := &PolicyDocument{
				Statement: []PolicyStatement{
					{Effect: "Allow", Action: "s3:GetObject"},
				},
			}

			mockIamAPI.EXPECT().SimulatePrincipalPolicy(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil, fmt.Errorf("access denied"))

			result, err := doc.checkPermissionsUsingQueryClient(client,
				"arn:aws:iam::123456789012:user/TestUser", nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("error simulating policy"))
			Expect(result).To(BeFalse())
		})

		It("passes region context entry when SimulateParams has a region", func() {
			doc := &PolicyDocument{
				Statement: []PolicyStatement{
					{Effect: "Allow", Action: "s3:GetObject"},
				},
			}

			mockIamAPI.EXPECT().SimulatePrincipalPolicy(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ interface{}, input *iam.SimulatePrincipalPolicyInput,
					_ ...interface{}) (*iam.SimulatePrincipalPolicyOutput, error) {
					Expect(input.ContextEntries).To(HaveLen(1))
					Expect(*input.ContextEntries[0].ContextKeyName).To(Equal("aws:RequestedRegion"))
					Expect(input.ContextEntries[0].ContextKeyValues).To(ConsistOf("us-east-1"))
					return &iam.SimulatePrincipalPolicyOutput{
						EvaluationResults: []iamtypes.EvaluationResult{
							{EvalActionName: awsSdk.String("s3:GetObject"), EvalDecision: "allowed"},
						},
						IsTruncated: false,
					}, nil
				})

			params := &SimulateParams{Region: "us-east-1"}
			result, err := doc.checkPermissionsUsingQueryClient(client,
				"arn:aws:iam::123456789012:user/TestUser", params)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("does not add region context when SimulateParams is nil", func() {
			doc := &PolicyDocument{
				Statement: []PolicyStatement{
					{Effect: "Allow", Action: "s3:GetObject"},
				},
			}

			mockIamAPI.EXPECT().SimulatePrincipalPolicy(gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ interface{}, input *iam.SimulatePrincipalPolicyInput,
					_ ...interface{}) (*iam.SimulatePrincipalPolicyOutput, error) {
					Expect(input.ContextEntries).To(BeEmpty())
					return &iam.SimulatePrincipalPolicyOutput{
						EvaluationResults: []iamtypes.EvaluationResult{
							{EvalActionName: awsSdk.String("s3:GetObject"), EvalDecision: "allowed"},
						},
						IsTruncated: false,
					}, nil
				})

			result, err := doc.checkPermissionsUsingQueryClient(client,
				"arn:aws:iam::123456789012:user/TestUser", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeTrue())
		})

		It("returns true for a document with no allowed actions", func() {
			doc := &PolicyDocument{
				Statement: []PolicyStatement{
					{Effect: "Deny", Action: "s3:GetObject"},
				},
			}

			mockIamAPI.EXPECT().SimulatePrincipalPolicy(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(&iam.SimulatePrincipalPolicyOutput{
					EvaluationResults: []iamtypes.EvaluationResult{},
					IsTruncated:       false,
				}, nil)

			result, err := doc.checkPermissionsUsingQueryClient(client,
				"arn:aws:iam::123456789012:user/TestUser", nil)
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeTrue())
		})
	})
})
