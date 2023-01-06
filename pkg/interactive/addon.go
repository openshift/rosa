package interactive

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

func GetAddonArgument(param *cmv1.AddOnParameter, input Input, dflt string) (string, error) {
	// Set default value based on existing parameter, otherwise use parameter default

	// add a prompt to question name to indicate if the boolean param is required and check validation
	if param.ValueType() == "boolean" && param.Validation() == "^true$" && param.Required() {
		input.Question = fmt.Sprintf("%s (required)", param.Name())
		input.Validators = []Validator{
			RegExpBoolean(param.Validation()),
		}
	}

	var err error
	var value string

	if len(input.Options) > 0 {
		value, err = GetOption(input)
		for _, paramOption := range param.Options() {
			if strings.Compare(paramOption.Name(), value) == 0 {
				value = paramOption.Value()
			}
		}

		if err != nil {
			return "", fmt.Errorf("expected a valid value for '%s': %v", param.ID(), err)
		}
		return value, nil
	}
	switch param.ValueType() {
	case "boolean":
		var boolVal bool
		input.Default, _ = strconv.ParseBool(dflt)
		boolVal, err = GetBool(input)
		if boolVal {
			value = "true"
		} else {
			value = "false"
		}
	case "cidr":
		var cidrVal net.IPNet
		if dflt != "" {
			_, defaultIDR, _ := net.ParseCIDR(dflt)
			input.Default = *defaultIDR
		}
		cidrVal, err = GetIPNet(input)
		value = cidrVal.String()
		if value == "<nil>" {
			value = ""
		}
	case "number", "resource":
		var numVal int
		input.Default, _ = strconv.Atoi(dflt)
		numVal, err = GetInt(input)
		value = fmt.Sprintf("%d", numVal)

	case "string":
		input.Default = dflt
		value, err = GetString(input)
	}

	if err != nil {
		return "", fmt.Errorf("expected a valid value for '%s': %v", param.ID(), err)
	}
	return value, nil
}
