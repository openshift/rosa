/*
Copyright (c) 2021 Red Hat, Inc.

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
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
)

const (
	OIDCClientIDOpenShift = "openshift"
	OIDCClientIDSTSAWS    = "sts.amazonaws.com"
)

func (c *awsClient) CreateOpenIDConnectProvider(providerURL string, thumbprint string) (string, error) {
	output, err := c.iamClient.CreateOpenIDConnectProvider(&iam.CreateOpenIDConnectProviderInput{
		ClientIDList: []*string{
			aws.String(OIDCClientIDOpenShift),
			aws.String(OIDCClientIDSTSAWS),
		},
		ThumbprintList: []*string{aws.String(thumbprint)},
		Url:            aws.String(providerURL),
	})
	if err != nil {
		return "", err
	}

	return aws.StringValue(output.OpenIDConnectProviderArn), nil
}
