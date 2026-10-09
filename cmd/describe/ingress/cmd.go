package ingress

import (
	"context"
	"fmt"
	"regexp"
	"sort"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift-online/rosa-hyperfleet-api/clientset/platform"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/hyperfleet"
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
		Args: func(cmd *cobra.Command, argv []string) error {
			if hyperfleet.Enabled() && len(argv) != 0 {
				return fmt.Errorf("HyperFleet describe ingress does not accept an ingress ID")
			}
			return cobra.MaximumNArgs(1)(cmd, argv)
		},
		Run: func(c *cobra.Command, argv []string) {
			if hyperfleet.Enabled() {
				rosa.DefaultRunner(rosa.RuntimeWithHyperFleet(), DescribeHyperfleetIngressRunner())(c, argv)
				return
			}
			rosa.DefaultRunner(rosa.RuntimeWithOCM(), DescribeIngressRunner(options))(c, argv)
		},
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

func DescribeHyperfleetIngressRunner() rosa.CommandRunner {
	return func(ctx context.Context, runtime *rosa.Runtime, _ *cobra.Command, _ []string) error {
		clusterUID, err := hyperfleet.ResolveClusterUID(ctx, runtime.HyperFleetClient, runtime.GetClusterKey())
		if err != nil {
			return err
		}
		cluster, err := runtime.HyperFleetClient.HyperfleetV1alpha1().Clusters().Get(ctx, clusterUID, platform.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get cluster: %w", err)
		}
		var routes []string
		if cluster.Spec.HostedCluster.Configuration != nil && cluster.Spec.HostedCluster.Configuration.Ingress != nil {
			for _, route := range cluster.Spec.HostedCluster.Configuration.Ingress.ComponentRoutes {
				routes = append(routes, fmt.Sprintf("%s: hostname=%s;tlsSecretRef=%s", route.Name, route.Hostname, route.ServingCertKeyPairSecret.Name))
			}
		}
		sort.Strings(routes)
		if output.HasFlag() {
			return output.Print(routes)
		}
		if len(routes) == 0 {
			fmt.Printf("No custom component routes configured for cluster '%s'\n", runtime.GetClusterKey())
			return nil
		}
		fmt.Printf("Custom component routes for cluster '%s':\n", runtime.GetClusterKey())
		for _, route := range routes {
			fmt.Printf("%s\n", route)
		}
		return nil
	}
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
