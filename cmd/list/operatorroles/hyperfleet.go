package operatorroles

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/briandowns/spinner"

	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/hyperfleet"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

// hfEnabled, hfExitFn, and hfListOperatorRoles are package-level
// vars so tests can stub the hyperfleet dispatch path.
var (
	hfEnabled           = hyperfleet.Enabled
	hfExitFn            = func(code int) { os.Exit(code) }
	hfListOperatorRoles = func() {
		r := rosa.NewRuntime().WithHyperFleet().WithAWSOnly()
		defer r.Cleanup()
		runHyperfleetList(r)
	}
)

// runHyperfleetList lists operator roles directly from AWS without OCM dependency.
func runHyperfleetList(r *rosa.Runtime) {
	var spin *spinner.Spinner
	if r.Reporter.IsTerminal() {
		spin = spinner.New(spinner.CharSets[9], 100*time.Millisecond)
	}
	if spin != nil {
		r.Reporter.Infof("Fetching operator roles")
		spin.Start()
	}

	// List operator roles from AWS
	// In hyperfleet mode, we use AWS client directly without OCM validation
	operatorsMap, err := r.AWSClient.ListOperatorRoles(args.version, "", args.prefix)
	prefixes := helper.MapKeys(operatorsMap)
	helper.SortStringRespectLength(prefixes)

	if spin != nil {
		spin.Stop()
	}

	if err != nil {
		r.Reporter.Errorf("Failed to get operator roles: %v", err)
		hfExitFn(1)
		return
	}

	if len(operatorsMap) == 0 {
		noOperatorRolesOutput := "No operator roles available"
		if args.version != "" {
			noOperatorRolesOutput = fmt.Sprintf("%s in version '%s'", noOperatorRolesOutput, args.version)
		}
		if args.prefix != "" {
			if _, ok := operatorsMap[args.prefix]; !ok {
				r.Reporter.Infof("No operator roles available for prefix '%s'", args.prefix)
				hfExitFn(0)
				return
			}
		}
		r.Reporter.Infof(noOperatorRolesOutput)
		hfExitFn(0)
		return
	}

	if output.HasFlag() {
		var resource interface{} = operatorsMap
		if args.prefix != "" {
			resource = operatorsMap[args.prefix]
		}
		err = output.Print(resource)
		if err != nil {
			r.Reporter.Errorf("%s", err)
			hfExitFn(1)
			return
		}
		hfExitFn(0)
		return
	}

	// Create the writer that will be used to print the tabulated results:
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if args.prefix == "" {
		fmt.Fprintf(writer, "ROLE PREFIX\tAMOUNT IN BUNDLE\n")
		for _, key := range prefixes {
			fmt.Fprintf(
				writer,
				"%s\t%d\n",
				key,
				len(operatorsMap[key]),
			)
		}
		writer.Flush()
		r.Reporter.Infof("\nUse --prefix=<prefix> to see detailed information about a specific operator role prefix")
		hfExitFn(0)
		return
	}

	if args.prefix != "" {
		if _, ok := operatorsMap[args.prefix]; !ok {
			noOperatorRolesPrefixOutput := fmt.Sprintf("No operator roles available for prefix '%s'", args.prefix)
			if args.version != "" {
				noOperatorRolesPrefixOutput =
					fmt.Sprintf("%s in version '%s'", noOperatorRolesPrefixOutput, args.version)
			}
			r.Reporter.Infof(noOperatorRolesPrefixOutput)
			hfExitFn(0)
			return
		}

		fmt.Fprintf(writer, "OPERATOR NAME\tOPERATOR NAMESPACE\tROLE NAME\t"+
			"ROLE ARN\tVERSION\tPOLICIES\tAWS Managed\n")
		for _, operatorRole := range operatorsMap[args.prefix] {
			awsManaged := "No"
			if operatorRole.ManagedPolicy {
				awsManaged = "Yes"
			}
			fmt.Fprintf(
				writer,
				"%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
				operatorRole.OperatorName,
				operatorRole.OperatorNamespace,
				operatorRole.RoleName,
				operatorRole.RoleARN,
				operatorRole.Version,
				operatorRole.AttachedPolicies,
				awsManaged,
			)
		}
		writer.Flush()
	}
}
