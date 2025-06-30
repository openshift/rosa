package kms_key

import (
	"encoding/json"
	"fmt"

	"github.com/openshift-online/ocm-common/pkg/aws/aws_client"
	"github.com/openshift-online/ocm-common/pkg/log"
)

func CreateOCMTestKMSKey(region string, multiRegion bool, testClient string) (string, error) {
	log.LogInfo("Preparing OCM testing kms key")
	client, err := aws_client.CreateAWSClient("", region)
	if err != nil {
		return "", err
	}
	accountRoleArns := []string{fmt.Sprintf("arn:%s:iam::%s:root", client.GetAWSPartition(), client.AccountID)}
	testKMSKeyPolicy := KMSKeyPolicy{
		Statement: []Statement{
			{
				Principal: Principal{
					Aws: accountRoleArns,
				},
				Sid:      "Enable IAM User Permissions",
				Effect:   "Allow",
				Action:   "kms:*",
				Resource: "*",
			},
		},
	}

	keyJson, err := json.Marshal(testKMSKeyPolicy)
	if err != nil {
		return "", err
	}
	keyString := string(keyJson)
	tagKey, tagValue, keyDescription := "Purpose", fmt.Sprintf("%s automation test", testClient), fmt.Sprintf("BYOK Test Key for client %s automation", testClient)

	_, kmsKeyArn, err := client.CreateKMSKeys(tagKey, tagValue, keyDescription, keyString, multiRegion)
	if err != nil {
		return "", err
	}
	return kmsKeyArn, nil
}

func ScheduleKeyDeletion(keyArn string, region string) error {
	client, err := aws_client.CreateAWSClient("", region)
	if err != nil {
		log.LogError(err.Error())
		return err
	}
	_, err = client.ScheduleKeyDeletion(keyArn, 7)
	return err
}

// Reference doc to https://docs.openshift.com/rosa/rosa_install_access_delete_clusters/rosa-sts-creating-a-cluster-with-customizations.html
func ConfigKMSKeyPolicyForSTS(key string, region string, HCP bool, accountRoles []string, operatorRoleArn map[string]string) error {

	if len(accountRoles) == 0 && len(operatorRoleArn) == 0 {
		return nil
	}
	log.LogInfo("Configuring sts roles to key polilies of %s", key)
	client, err := aws_client.CreateAWSClient("", region)
	if err != nil {
		return err
	}
	KMSPolicyResponse, err := client.GetKMSPolicy(key, "")
	if err != nil {
		return err
	}
	var KMSPolicy KMSKeyPolicy
	err = json.Unmarshal([]byte(*KMSPolicyResponse.Policy), &KMSPolicy)
	if err != nil {
		return err
	}
	var additionalStatements []Statement = []Statement{}
	if !HCP {
		roles := append(accountRoles, operatorRoleArn["ebs-cloud-credentials"])
		s1 := Statement{
			Principal: Principal{
				Aws: roles,
			},
			Sid:    "Allow ROSA use of the key",
			Effect: "Allow",
			Action: []string{
				"kms:Encrypt",
				"kms:Decrypt",
				"kms:ReEncrypt*",
				"kms:GenerateDataKey*",
				"kms:DescribeKey"},
			Resource: "*",
		}

		s2 := Statement{
			Principal: Principal{
				Aws: roles,
			},
			Sid:    "Allow attachment of persistent resources",
			Effect: "Allow",
			Action: []string{
				"kms:CreateGrant",
				"kms:ListGrants",
				"kms:RevokeGrant"},
			Resource: "*",
			Condition: Condition{
				Bool: map[string]interface{}{
					"kms:GrantIsForAWSResource": "true",
				},
			},
		}
		additionalStatements = append(additionalStatements, s1, s2)
	} else {
		s1 := Statement{
			Principal: Principal{
				Aws: accountRoles,
			},
			Sid:    "Installer Permissions",
			Effect: "Allow",
			Action: []string{
				"kms:CreateGrant",
				"kms:DescribeKey",
				"kms:GenerateDataKeyWithoutPlaintext"},
			Resource: "*",
		}

		s2 := Statement{
			Principal: Principal{
				Aws: operatorRoleArn["kube-controller-manager"],
			},
			Sid:    "ROSA KubeControllerManager Permissions",
			Effect: "Allow",
			Action: []string{
				"kms:DescribeKey",
			},
			Resource: "*",
		}
		s3 := Statement{
			Principal: Principal{
				Aws: operatorRoleArn["kms-provider"],
			},
			Sid:    "ROSA KMS Provider Permissions",
			Effect: "Allow",
			Action: []string{
				"kms:Encrypt",
				"kms:Decrypt",
				"kms:DescribeKey",
			},
			Resource: "*",
		}
		s4 := Statement{
			Principal: Principal{
				Aws: operatorRoleArn["capa-controller-manager"],
			},
			Sid:    "ROSA NodeManager Permissions",
			Effect: "Allow",
			Action: []string{
				"kms:DescribeKey",
				"kms:GenerateDataKeyWithoutPlaintext",
				"kms:CreateGrant",
			},
			Resource: "*",
		}
		additionalStatements = append(additionalStatements, s1, s2, s3, s4)
	}
	KMSPolicy.Statement = append(KMSPolicy.Statement, additionalStatements...)
	keyJson, err := json.Marshal(KMSPolicy)
	if err != nil {
		return err
	}
	keyString := string(keyJson)

	_, err = client.PutKMSPolicy(key, "", keyString)
	return err
}

func AddTagToKMS(key string, region string, tags map[string]string) error {
	client, err := aws_client.CreateAWSClient("", region)
	if err != nil {
		return err
	}
	for k, v := range tags {
		_, err := client.TagKeys(key, k, v)
		if err != nil {
			return err
		}

	}
	return nil
}
