package breakglasscredential

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/breakglasscredential"
	"github.com/openshift/rosa/pkg/externalauthprovider"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var breakGlassCredentialArgs *breakglasscredential.BreakGlassCredentialArgs

var Cmd = makeCmd()

func makeCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "break-glass-credential",
		Aliases: []string{"break-glass-credentials", "breakglasscredential", "breakglasscredentials"},
		Short:   "Create a break glass credential for a cluster.",
		Long:    "Create a break glass credential for a hosted control plane cluster with external authentication enabled.",
		Example: `  # Interactively create a break glass credential to a cluster named "mycluster"
  rosa create break-glass-credential --cluster=mycluster --interactive`,
		Run:  run,
		Args: cobra.NoArgs,
	}
}

func init() {
	ocm.AddClusterFlag(Cmd)
	interactive.AddFlag(Cmd.Flags())
	breakGlassCredentialArgs = breakglasscredential.AddBreakGlassCredentialFlags(Cmd)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()
	err := runWithRuntime(r, cmd, argv)
	if err != nil {
		r.Reporter.Errorf(err.Error())
		os.Exit(1)
	}
}

func runWithRuntime(r *rosa.Runtime, cmd *cobra.Command, argv []string) error {
	clusterKey := r.GetClusterKey()
	cluster := r.FetchCluster()

	externalAuthService := externalauthprovider.NewExternalAuthService(r.OCMClient)
	err := externalAuthService.IsExternalAuthProviderSupported(cluster, clusterKey)
	if err != nil {
		return err
	}

	if interactive.Enabled() {
		r.Reporter.Infof("Enabling interactive mode")
	}
	r.Reporter.Debugf("Creating a break glass credential for cluster '%s'", clusterKey)

	args, err := breakglasscredential.GetBreakGlassCredentialOptions(
		cmd.Flags(), breakGlassCredentialArgs)
	if err != nil {
		return fmt.Errorf("failed to create a break glass credential for cluster '%s': %s",
			clusterKey, err)
	}

	credentialResponse, err := breakglasscredential.CreateBreakGlass(cluster, clusterKey, args, r)
	if err != nil {
		return err
	}

	kubeconfig, err := r.OCMClient.PollKubeconfig(
		cluster.ID(), credentialResponse.ID(), ocm.DefaultKubeConfigPollInterval, ocm.DefaultKubeConfigTimeout)
	if err != nil {
		return fmt.Errorf("An error occurred while polling for kubeconfig: %v", err)
	}

	r.Reporter.Infof("Successfully created a break glass credential for cluster '%s'.",
		clusterKey)
	r.Reporter.Infof(
		"To retrieve only the kubeconfig for this credential "+
			"use: 'rosa describe break-glass-credential %s -c %s --kubeconfig'",
		credentialResponse.ID(), clusterKey)
	fmt.Print(kubeconfig)

	return nil
}
