package aws_client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/iam"
)

func (client *AWSClient) DeleteOIDCProvider(providerArn string) error {
	input := &iam.DeleteOpenIDConnectProviderInput{
		OpenIDConnectProviderArn: &providerArn,
	}
	_, err := client.IamClient.DeleteOpenIDConnectProvider(context.TODO(), input)
	return err
}
