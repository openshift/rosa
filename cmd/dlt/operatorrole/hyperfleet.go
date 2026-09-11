package operatorrole

import (
	"fmt"
	"os"
	"time"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/briandowns/spinner"

	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/roles"
	"github.com/openshift/rosa/pkg/rosa"
)

// hfEnabled, hfExitFn, and hfDeleteOperatorRoles are package-level
// vars so tests can stub the hyperfleet dispatch path.
var (
	hfEnabled             = hyperfleet.Enabled
	hfExitFn              = func(code int) { os.Exit(code) }
	hfDeleteOperatorRoles = func() {
		r := rosa.NewRuntime().WithHyperFleet().WithAWSOnly()
		defer r.Cleanup()
		runHyperfleetDelete(r)
	}
)

// runHyperfleetDelete deletes operator roles directly from AWS without OCM dependency.
func runHyperfleetDelete(r *rosa.Runtime) {
	mode, err := interactive.GetMode()
	if err != nil {
		r.Reporter.Errorf("%s", err)
		hfExitFn(1)
		return
	}

	// In hyperfleet mode, only --prefix is supported (no cluster lookup via OCM)
	if args.prefix == "" {
		r.Reporter.Errorf("--prefix is required in hyperfleet mode")
		r.Reporter.Infof("Use 'rosa list operator-roles' to see available prefixes")
		hfExitFn(1)
		return
	}

	var spin *spinner.Spinner
	if r.Reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}

	fetchingReporterOutput := fmt.Sprintf("Fetching operator roles for prefix: %s", args.prefix)
	if spin != nil {
		r.Reporter.Infof("%s", fetchingReporterOutput)
		spin.Start()
	}

	var foundOperatorRoles []string
	for _, roleName := range hyperfleet.OperatorRoleNames(args.prefix) {
		exists, _, err := r.AWSClient.CheckRoleExists(roleName)
		if err != nil {
			if spin != nil {
				spin.Stop()
			}
			r.Reporter.Errorf("There was a problem retrieving the Operator Roles from AWS: %v", err)
			hfExitFn(1)
			return
		}
		if exists {
			foundOperatorRoles = append(foundOperatorRoles, roleName)
		}
	}
	if spin != nil {
		spin.Stop()
	}

	if len(foundOperatorRoles) == 0 {
		r.Reporter.Infof("There are no operator roles to delete for prefix '%s'", args.prefix)
		return
	}

	r.Reporter.Infof("Found %d operator role(s) with prefix '%s'", len(foundOperatorRoles), args.prefix)

	// Check if roles have managed policies
	_, roleARN, err := r.AWSClient.CheckRoleExists(foundOperatorRoles[0])
	if err != nil {
		r.Reporter.Errorf("Failed to get '%s' role ARN", foundOperatorRoles[0])
		hfExitFn(1)
		return
	}
	managedPolicies, err := r.AWSClient.HasManagedPolicies(roleARN)
	if err != nil {
		r.Reporter.Errorf("Failed to determine if cluster has managed policies: %v", err)
		hfExitFn(1)
		return
	}

	errOccurred := false
	switch mode {
	case interactive.ModeAuto:
		// Only ask user if they want to delete policies if they are deleting HcpSharedVpc roles
		deleteHcpSharedVpcPolicies := args.deleteHcpSharedVpcPolicies
		if roles.CheckIfRolesAreHcpSharedVpc(r, foundOperatorRoles) &&
			!deleteHcpSharedVpcPolicies {
			deleteHcpSharedVpcPolicies = confirm.Prompt(true, "Attempt to delete Hosted CP shared VPC policies?")
		}

		allSharedVpcPoliciesNotDeleted := make(map[string]bool)
		for _, role := range foundOperatorRoles {
			if !confirm.Prompt(true, "Delete the operator role '%s'?", role) {
				continue
			}
			r.Reporter.Infof("Deleting operator role '%s'", role)
			if spin != nil {
				spin.Start()
			}
			sharedVpcPoliciesNotDeleted, err := r.AWSClient.DeleteOperatorRole(role, managedPolicies,
				deleteHcpSharedVpcPolicies)
			for key, value := range sharedVpcPoliciesNotDeleted {
				allSharedVpcPoliciesNotDeleted[key] = value
			}

			if err != nil {
				if spin != nil {
					spin.Stop()
				}
				r.Reporter.Warnf("There was an error deleting the Operator Role or Policies: %s", err)
				errOccurred = true
				continue
			}
			if spin != nil {
				spin.Stop()
			}
		}
		for policyOutput, notDeleted := range allSharedVpcPoliciesNotDeleted {
			if notDeleted {
				r.Logger.Warnf("Unable to delete policy %s: Policy still attached to other resources",
					policyOutput)
			}
		}
		if !errOccurred {
			r.Reporter.Infof("Successfully deleted the operator roles")
		}

	case interactive.ModeManual:
		policyMap, arbitraryPolicyMap, err := r.AWSClient.GetOperatorRolePolicies(foundOperatorRoles)
		if err != nil {
			r.Reporter.Errorf("There was an error getting the policy: %v", err)
			hfExitFn(1)
			return
		}

		// Get HCP shared vpc policy details if the user is deleting roles related to HCP shared vpc
		policiesOutput := make([]*iam.GetPolicyOutput, 0)
		if roles.CheckIfRolesAreHcpSharedVpc(r, foundOperatorRoles) && args.deleteHcpSharedVpcPolicies {
			for _, role := range foundOperatorRoles {
				policies, err := r.AWSClient.GetPolicyDetailsFromRole(awssdk.String(role))
				policiesOutput = append(policiesOutput, policies...)
				if err != nil {
					r.Reporter.Warnf("There was an error getting details of policies attached to role '%s': %v",
						role, err)
				}
			}
		}

		commands := buildCommand(r, foundOperatorRoles, policyMap, arbitraryPolicyMap, managedPolicies, policiesOutput)
		if r.Reporter.IsTerminal() {
			r.Reporter.Infof("Run the following commands to delete the Operator roles and policies:\n")
		}
		fmt.Println(commands)

	default:
		r.Reporter.Errorf("Invalid mode. Allowed values are %s", interactive.Modes)
		hfExitFn(1)
		return
	}
}
