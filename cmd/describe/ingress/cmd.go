package ingress

import (
	"context"
	"fmt"
	"regexp"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/ingress"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

const (
	use     = "ingress"
	short   = "Show details of the specified ingress within cluster"
	example = `rosa describe ingress <ingress_id> -c mycluster`
)

// Regular expression to used to make sure that the identifier given by the
// user is safe and that it there is no risk of SQL injection:
var ingressKeyRE = regexp.MustCompile(`^[a-z0-9]{3,5}$`)

func NewDescribeIngressCommand() *cobra.Command {
	options := NewDescribeIngressUserOptions()
	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Example: example,
		Run:     rosa.DefaultRunner(rosa.RuntimeWithOCM(), DescribeIngressRunner(options)),
		Args:    cobra.MaximumNArgs(1),
	}

	flags := cmd.Flags()
	flags.StringVar(
		&options.ingress,
		"ingress",
		"",
		"Ingress of the cluster to target",
	)

	ocm.AddClusterFlag(cmd)
	output.AddFlag(cmd)
	return cmd
}

func DescribeIngressRunner(userOptions DescribeIngressUserOptions) rosa.CommandRunner {
	return func(_ context.Context, runtime *rosa.Runtime, cmd *cobra.Command, argv []string) error {
		options := NewDescribeIngressOptions()
		if len(argv) == 1 && !cmd.Flag("ingress").Changed {
			userOptions.ingress = argv[0]
		} else {
			err := cmd.ParseFlags(argv)
			if err != nil {
				return fmt.Errorf("unable to parse flags: %v", err)
			}
			userOptions.ingress = cmd.Flag("ingress").Value.String()
		}
		err := options.Bind(userOptions)
		if err != nil {
			return err
		}
		clusterKey := runtime.GetClusterKey()
		cluster := runtime.FetchCluster()
		if cluster.State() != cmv1.ClusterStateReady {
			return fmt.Errorf("cluster '%s' is not yet ready", clusterKey)
		}
		service := ingress.NewIngressService()
		return service.DescribeIngress(runtime, cluster, options.args.ingress)
	}
}
