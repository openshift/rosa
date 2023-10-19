package validations

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/aws"
	semver "github.com/hashicorp/go-version"
	"github.com/openshift-online/ocm-common/pkg"
)

func GetRoleName(prefix string, role string) string {
	name := fmt.Sprintf("%s-%s-Role", prefix, role)
	if len(name) > pkg.MaxByteSize {
		name = name[0:pkg.MaxByteSize]
	}
	return name
}

func IsManagedRole(roleTags []*iam.Tag) bool {
    for _, tag := range roleTags {
        if aws.StringValue(tag.Key) == ManagedPolicies && aws.StringValue(tag.Value) == "true" {
            return true
        }
    }

    return false
}

func HasCompatibleVersionTags(iamTags []*iam.Tag, version string) (bool, error) {
	if len(iamTags) == 0 {
		return false, nil
	}

	wantedVersion, err := semver.NewVersion(version)
	if err != nil {
		return false, err
	}
	
	for _, tag := range iamTags {
		if aws.StringValue(tag.Key) == OpenShiftVersion {
			if version == aws.StringValue(tag.Value) {
				return true, nil
			}
			
			currentVersion, err := semver.NewVersion(aws.StringValue(tag.Value))
			if err != nil {
				return false, err
			}
			return currentVersion.GreaterThanOrEqual(wantedVersion), nil
		}
	}
	return false, nil
}

func IamResourceHasTag(iamTags []*iam.Tag, tagKey string, tagValue string) bool {
	for _, tag := range iamTags {
		if aws.StringValue(tag.Key) == tagKey && aws.StringValue(tag.Value) == tagValue {
			return true
		}
	}

	return false
}
