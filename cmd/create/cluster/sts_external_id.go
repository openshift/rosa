package cluster

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/aws"
)

var errMismatchedSTSExternalIDTrustPolicies = errors.New(
	"installer and support role trust policies define STS external IDs with no value in common",
)

func validateChangedSTSExternalIDFlag(externalID string) error {
	return aws.ValidateSTSExternalIDFormat(externalID)
}

func resolveSTSExternalIDForClusterCreate(
	awsClient aws.Client,
	externalID, installerRoleARN, supportRoleARN string,
) (aws.STSExternalIDClusterResolution, error) {
	if installerRoleARN == "" || supportRoleARN == "" {
		return aws.STSExternalIDClusterResolution{ExternalID: externalID}, nil
	}
	resolution, err := aws.ResolveSTSExternalIDForClusterDetails(
		externalID, installerRoleARN, supportRoleARN, awsClient,
	)
	if err != nil {
		return aws.STSExternalIDClusterResolution{}, fmt.Errorf("failed to resolve STS external ID: %w", err)
	}
	return resolution, nil
}

func checkSTSExternalIDResolution(
	resolution aws.STSExternalIDClusterResolution,
	externalIDFlagChanged bool,
) error {
	if resolution.MismatchedTrustPolicies && !externalIDFlagChanged {
		return errMismatchedSTSExternalIDTrustPolicies
	}
	return nil
}

func shouldWarnAmbiguousSTSExternalID(
	resolution aws.STSExternalIDClusterResolution,
	externalIDFlagChanged bool,
) bool {
	return resolution.Ambiguous && !externalIDFlagChanged
}

func resolveEnteredSTSExternalIDForCluster(
	awsClient aws.Client,
	externalID, installerRoleARN, supportRoleARN string,
) (string, error) {
	if err := aws.ValidateSTSExternalIDFormat(externalID); err != nil {
		return "", err
	}
	return aws.ResolveSTSExternalIDForCluster(externalID, installerRoleARN, supportRoleARN, awsClient)
}

func externalIDFlagChanged(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("external-id")
}
