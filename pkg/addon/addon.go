package addon

import (
	"fmt"
	"regexp"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

func PrintDescription(addOn *cmv1.AddOn) {
	fmt.Printf("ADD-ON\n"+
		"ID:               %s\n"+
		"Name:             %s\n"+
		"Description:      %s\n"+
		"Documentation:    %s\n"+
		"Operator:         %s\n"+
		"Target namespace: %s\n"+
		"Install mode:     %s\n",
		addOn.ID(),
		addOn.Name(),
		wrapText(addOn.Description()),
		addOn.DocsLink(),
		addOn.OperatorName(),
		addOn.TargetNamespace(),
		addOn.InstallMode(),
	)
	fmt.Println()
}

func PrintCredentialRequests(requests []*cmv1.CredentialRequest) {
	if len(requests) > 0 {
		fmt.Printf("CREDENTIALS REQUESTS\n")
		for _, cr := range requests {
			fmt.Printf(""+
				"- Service account:  %s\n"+
				"  Secret name:      %s\n"+
				"  Secret namespace: %s\n",
				cr.ServiceAccount(),
				cr.Name(),
				cr.Namespace(),
			)
			if len(cr.PolicyPermissions()) > 0 {
				fmt.Printf("  Policy permissions:\n")
				for _, p := range cr.PolicyPermissions() {
					fmt.Printf("  - %s\n", p)
				}
			}
		}
	}
	fmt.Println()
}

func PrintParameters(params *cmv1.AddOnParameterList) {
	if params.Len() > 0 {
		fmt.Printf("ADD-ON PARAMETERS\n")
		params.Each(func(param *cmv1.AddOnParameter) bool {
			if !param.Enabled() {
				return true
			}
			fmt.Printf(""+
				"- ID:             %s\n"+
				"  Name:           %s\n"+
				"  Description:    %s\n"+
				"  Type:           %s\n"+
				"  Required:       %s\n"+
				"  Editable:       %s\n",
				param.ID(),
				param.Name(),
				wrapText(param.Description()),
				param.ValueType(),
				printBool(param.Required()),
				printBool(param.Editable()),
			)
			if param.DefaultValue() != "" {
				fmt.Printf("  Default Value:  %s\n", param.DefaultValue())
			}
			if param.Validation() != "" {
				fmt.Printf("  Validation:     /%s/\n", param.Validation())
			}
			fmt.Println()
			return true
		})
	}
}

func printBool(val bool) string {
	if val {
		return "yes"
	}
	return "no"
}

func wrapText(text string) string {
	return strings.TrimSpace(
		regexp.MustCompile(`(.{1,80})( +|$\n?)|(.{1,80})`).
			ReplaceAllString(text, "$1$3\n                  "),
	)
}
