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
	"fmt"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/iam"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const (
	OIDCClientIDOpenShift = "openshift"
	OIDCClientIDSTSAWS    = "sts.amazonaws.com"
)

type OIDC struct {
	ARN       string `json:"ARN,omitempty"`
	ClusterID string `json:"ClusterID,omitempty"`
	InUse     string `json:"InUse,omitempty"`
}

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

func (c *awsClient) HasOpenIDConnectProvider(issuerURL string, accountID string) (bool, error) {
	parsedIssuerURL, err := url.ParseRequestURI(issuerURL)
	if err != nil {
		return false, err
	}
	providerURL := fmt.Sprintf("%s%s", parsedIssuerURL.Host, parsedIssuerURL.Path)
	oidcProviderARN := fmt.Sprintf("arn:aws:iam::%s:oidc-provider/%s", accountID, providerURL)
	output, err := c.iamClient.GetOpenIDConnectProvider(&iam.GetOpenIDConnectProviderInput{
		OpenIDConnectProviderArn: aws.String(oidcProviderARN),
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
	}
	if aws.StringValue(output.Url) != providerURL {
		return false, fmt.Errorf("The OIDC provider exists but is misconfigured")
	}
	return true, nil
}

func (c *awsClient) ListOpenIDConnectProviders(clusterID string, clusters []*cmv1.Cluster) ([]OIDC, error) {
	// get all open id connect providers
	oidcProviderOutput, err := c.iamClient.ListOpenIDConnectProviders(&iam.ListOpenIDConnectProvidersInput{})
	if err != nil {
		return nil, err
	}

	var oidcProviders []OIDC
	for _, oidcp := range oidcProviderOutput.OpenIDConnectProviderList {
		// filter out rosa oidc providers
		if !strings.Contains(aws.StringValue(oidcp.Arn), "rh-oidc") {
			continue
		}

		// split arn to get cluster id
		splitOIDC := strings.Split(aws.StringValue(oidcp.Arn), "amazonaws.com/")
		oidcClusterID := splitOIDC[1]

		// filter out oidc provider if cluster id is passed
		if clusterID != "" && oidcClusterID != clusterID {
			continue
		}

		// declare oidc object, defaulting InUse to `no`
		oidc := OIDC{
			ARN:       aws.StringValue(oidcp.Arn),
			ClusterID: oidcClusterID,
			InUse:     "no",
		}

		// set InUse to `yes` if there is a cluster in OCM with the corresponding ID
		for _, cluster := range clusters {
			if cluster.ID() == oidcClusterID {
				oidc.InUse = "yes"
				break
			}
		}
		oidcProviders = append(oidcProviders, oidc)
	}
	return oidcProviders, nil
}
