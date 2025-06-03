package ingress

import (
	"fmt"

	"github.com/openshift/rosa/pkg/reporter"
)

type DescribeIngressUserOptions struct {
	ingress string
}

type DescribeIngressOptions struct {
	reporter reporter.Logger
	args     DescribeIngressUserOptions
}

func NewDescribeIngressUserOptions() DescribeIngressUserOptions {
	return DescribeIngressUserOptions{ingress: ""}
}

func NewDescribeIngressOptions() *DescribeIngressOptions {
	return &DescribeIngressOptions{
		reporter: reporter.CreateReporter(),
		args:     NewDescribeIngressUserOptions(),
	}
}

func (i *DescribeIngressOptions) Bind(args DescribeIngressUserOptions) error {
	if args.ingress == "" {
		return fmt.Errorf("you need to specify an ingress ID/alias")
	}
	ingressKey := args.ingress
	if !ingressKeyRE.MatchString(ingressKey) {
		return fmt.Errorf(
			"Ingress identifier '%s' isn't valid: it must contain between three and five lowercase letters or digits",
			ingressKey,
		)
	}
	i.args.ingress = args.ingress
	return nil
}
