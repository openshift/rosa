/*
Copyright (c) 2020 Red Hat, Inc.

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

package aws

import (
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/cloudformation/cloudformationiface"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/aws/aws-sdk-go/service/organizations/organizationsiface"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
)

type Client interface {
	ValidateCredentials() (bool, error)
	EnsureAdminUser() (bool, error)
	ValidateSCP() (bool, error)
	EnsurePermissions() (bool, error)
}

type awsClient struct {
	cloudformationClient cloudformationiface.CloudFormationAPI
	iamClient            iamiface.IAMAPI
	orgClient            organizationsiface.OrganizationsAPI
	stsClient            stsiface.STSAPI
}

func NewClient() (Client, error) {
	// Create the AWS session:
	sess, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	})
	if err != nil {
		return nil, err
	}

	// Check that the session is set:
	region := aws.StringValue(sess.Config.Region)
	if region == "" {
		return nil, errors.New("Region is not set")
	}

	// Check that the AWS credentials are available:
	_, err = sess.Config.Credentials.Get()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Can't find AWS credentials: %v", err))
	}

	return &awsClient{
		cloudformationClient: cloudformation.New(sess),
		iamClient:            iam.New(sess),
		orgClient:            organizations.New(sess),
		stsClient:            sts.New(sess),
	}, nil
}

// Checks if given credentials are valid.
func (c *awsClient) ValidateCredentials() (bool, error) {
	_, err := c.stsClient.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return false, err
	}
	return true, nil
}

// Ensure osdCcsAdmin account
func (c *awsClient) EnsureAdminUser() (bool, error) {
	username := aws.String("osdCcsAdmin")

	exists, err := c.userExists(username)
	if !exists {
		user, err := c.iamClient.CreateUser(&iam.CreateUserInput{
			UserName: username,
		})
		if err != nil {
			return false, err
		}
		fmt.Fprintf(os.Stdout, "[DEBUG] EnsureAdminUser::CreateUser\n%+v\n", user)
	}
	if err != nil {
		return false, err
	}

	hasPolicy, err := c.userHasPolicy(username, aws.String("AdministratorAccess"))
	if !hasPolicy {
		policy, err := c.iamClient.AttachUserPolicy(&iam.AttachUserPolicyInput{
			PolicyArn: aws.String("arn:aws:iam::aws:policy/AdministratorAccess"),
			UserName:  username,
		})
		if err != nil {
			return false, err
		}
		fmt.Fprintf(os.Stdout, "[DEBUG] EnsureAdminUser::AttachUserPolicy\n%+v\n", policy)
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (c *awsClient) userExists(username *string) (bool, error) {
	_, err := c.iamClient.GetUser(&iam.GetUserInput{
		UserName: username,
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				return false, nil
			default:
				return false, err
			}
		}
		return false, err
	}
	return true, nil
}

func (c *awsClient) userHasPolicy(username *string, policy *string) (bool, error) {
	_, err := c.iamClient.GetUserPolicy(&iam.GetUserPolicyInput{
		PolicyName: policy,
		UserName:   username,
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				return false, nil
			default:
				return false, err
			}
		}
		return false, err
	}
	return true, nil
}

// Validate SCP...
func (c *awsClient) ValidateSCP() (bool, error) {
	policyType := aws.String("SERVICE_CONTROL_POLICY")
	policies, err := c.orgClient.ListPolicies(&organizations.ListPoliciesInput{
		Filter: policyType,
	})
	if err != nil {
		return false, err
	}
	fmt.Fprintf(os.Stdout, "[DEBUG] ValidateSCP::ListPolicies\n%+v\n", policies)
	return true, nil
}

// Ensure correct permissions on account
func (c *awsClient) EnsurePermissions() (bool, error) {
	// stack, err := c.cloudformationClient.CreateStack(&cloudformation.CreateStackInput{})
	// if err != nil {
	// 	return false, err
	// }
	// fmt.Fprintf(os.Stdout, "[DEBUG] EnsurePermissions::CreateStack\n%+v\n", stack)
	return true, nil
}
