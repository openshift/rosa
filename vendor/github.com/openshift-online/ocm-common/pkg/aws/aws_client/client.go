package aws_client

import (
	"context"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/ram"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	"github.com/openshift-online/ocm-common/pkg/log"

	elb "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	"github.com/aws/aws-sdk-go-v2/service/route53"

	CON "github.com/openshift-online/ocm-common/pkg/aws/consts"
)

type AWSClient struct {
	Ec2Client            *ec2.Client
	Route53Client        *route53.Client
	StackFormationClient *cloudformation.Client
	ElbClient            *elb.Client
	StsClient            *sts.Client
	Region               string
	IamClient            *iam.Client
	ClientContext        context.Context
	AccountID            string
	Arn                  string
	KmsClient            *kms.Client
	CloudWatchLogsClient *cloudwatchlogs.Client
	AWSConfig            *aws.Config
	RamClient            *ram.Client
}

type AccessKeyMod struct {
	AccessKeyId     string `ini:"aws_access_key_id,omitempty"`
	SecretAccessKey string `ini:"aws_secret_access_key,omitempty"`
}

func CreateAWSClient(profileName string, region string, awsSharedCredentialFile ...string) (*AWSClient, error) {
	var cfg aws.Config
	var err error

	if len(awsSharedCredentialFile) > 0 {
		file := awsSharedCredentialFile[0]
		log.LogInfo("Got aws shared credential file path: %s ", file)
		cfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(region),
			config.WithSharedCredentialsFiles([]string{file}),
		)
	} else {
		if envAwsProfile() {
			file := os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
			log.LogInfo("Got file path: %s from env variable AWS_SHARED_CREDENTIALS_FILE\n", file)
			cfg, err = config.LoadDefaultConfig(context.TODO(),
				config.WithRegion(region),
				config.WithSharedCredentialsFiles([]string{file}),
			)
		} else {
			if envCredential() {
				log.LogInfo("Got AWS_ACCESS_KEY_ID env settings, going to build the config with the env")
				cfg, err = config.LoadDefaultConfig(context.TODO(),
					config.WithRegion(region),
					config.WithCredentialsProvider(
						credentials.NewStaticCredentialsProvider(
							os.Getenv("AWS_ACCESS_KEY_ID"),
							os.Getenv("AWS_SECRET_ACCESS_KEY"),
							"")),
				)
			} else {
				log.LogInfo("AWS_SHARED_CREDENTIALS_FILE not supplied")
				cfg, err = config.LoadDefaultConfig(context.TODO(),
					config.WithRegion(region),
					config.WithSharedConfigProfile(profileName),
				)
			}
		}
	}

	if err != nil {
		return nil, err
	}

	awsClient := &AWSClient{
		Ec2Client:            ec2.NewFromConfig(cfg),
		Route53Client:        route53.NewFromConfig(cfg),
		StackFormationClient: cloudformation.NewFromConfig(cfg),
		ElbClient:            elb.NewFromConfig(cfg),
		Region:               region,
		StsClient:            sts.NewFromConfig(cfg),
		IamClient:            iam.NewFromConfig(cfg),
		ClientContext:        context.TODO(),
		KmsClient:            kms.NewFromConfig(cfg),
		AWSConfig:            &cfg,
		RamClient:            ram.NewFromConfig(cfg),
		CloudWatchLogsClient: cloudwatchlogs.NewFromConfig(cfg),
	}
	out, err := awsClient.GetCallerIdentity()
	if err != nil {
		return nil, err
	}
	awsClient.AccountID = *out.Account
	awsClient.Arn = *out.Arn
	return awsClient, nil
}

func (client *AWSClient) GetCallerIdentity() (*sts.GetCallerIdentityOutput, error) {
	input := &sts.GetCallerIdentityInput{}
	out, err := client.StsClient.GetCallerIdentity(client.ClientContext, input)
	if err != nil {
		log.LogError("Error happened when calling GetCallerIdentity: %s", err)
		return nil, err
	}
	return out, nil
}

func (client *AWSClient) GetAWSPartition() string {
	defaultPartition := "aws"
	input := &sts.GetCallerIdentityInput{}
	out, err := client.StsClient.GetCallerIdentity(client.ClientContext, input)
	if err != nil {
		// Failed to get caller identity, return default partition
		return defaultPartition
	}
	segments := strings.Split(*out.Arn, ":")
	if len(segments) < 2 {
		// Failed to parse ARN, return default partition
		return defaultPartition
	}
	return segments[1]
}

func (client *AWSClient) EC2() *ec2.Client {
	return client.Ec2Client
}

func (client *AWSClient) Route53() *route53.Client {
	return client.Route53Client
}
func (client *AWSClient) CloudFormation() *cloudformation.Client {
	return client.StackFormationClient
}
func (client *AWSClient) ELB() *elb.Client {
	return client.ElbClient
}

func GrantValidAccessKeys(userName string) (*AccessKeyMod, error) {
	var cre aws.Credentials
	var keysMod *AccessKeyMod
	var err error
	retryTimes := 3
	for retryTimes > 0 {
		if cre.AccessKeyID != "" {
			break
		}
		client, err := CreateAWSClient(userName, CON.DefaultAWSRegion)
		if err != nil {
			return nil, err
		}

		cre, err = client.AWSConfig.Credentials.Retrieve(client.ClientContext)
		if err != nil {
			return nil, err
		}
		log.LogInfo(">>> Access key grant successfully")

		keysMod = &AccessKeyMod{
			AccessKeyId:     cre.AccessKeyID,
			SecretAccessKey: cre.SecretAccessKey,
		}
		retryTimes--
	}
	return keysMod, err
}
