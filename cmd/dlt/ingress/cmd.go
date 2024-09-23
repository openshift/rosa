/*
Copyright (c) 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ingress

import (
	"fmt"
	"os"
	"regexp"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/interactive/confirm"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

// Regular expression to used to make sure that the identifier given by the
// user is safe and that it there is no risk of SQL injection:
var ingressKeyRE = regexp.MustCompile(`^[a-z0-9]{3,5}$`)

var Cmd = &cobra.Command{
	Use:     "ingress ID",
	Aliases: []string{"ingresses", "route", "routes"},
	Short:   "Delete cluster ingress",
	Long:    "Delete the additional non-default application router for a cluster.",
	Example: `  # Delete ingress with ID a1b2 from a cluster named 'mycluster'
  rosa delete ingress --cluster=mycluster a1b2

  # Delete secondary ingress using the sub-domain name
  rosa delete ingress --cluster=mycluster apps2`,
	Run: run,
	Args: func(_ *cobra.Command, argv []string) error {
		if len(argv) != 1 {
			return fmt.Errorf(
				"Expected exactly one command line parameter containing the id of the ingress",
			)
		}
		return nil
	},
}

func init() {
	ocm.AddClusterFlag(Cmd)
}

func run(_ *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	ingressID := argv[0]
	if !ingressKeyRE.MatchString(ingressID) {
		r.Reporter.Errorf(
			"Ingress identifier '%s' isn't valid: it must contain between three and five lowercase letters or digits",
			ingressID,
		)
		os.Exit(1)
	}

	clusterKey := r.GetClusterKey()

	cluster := r.FetchCluster()
	// Try to find the ingress:
	r.Reporter.Debugf("Loading ingresses for cluster '%s'", clusterKey)
	ingresses, err := r.OCMClient.GetIngresses(cluster.ID())
	if err != nil {
		r.Reporter.Errorf("Failed to get ingresses for cluster '%s': %v", clusterKey, err)
		os.Exit(1)
	}

	var ingress *cmv1.Ingress
	for _, item := range ingresses {
		if ingressID == "apps" && item.Default() {
			ingress = item
		}
		if ingressID == "apps2" && !item.Default() {
			ingress = item
		}
		if item.ID() == ingressID {
			ingress = item
		}
	}
	if ingress == nil {
		r.Reporter.Errorf("Ingress '%s' does not exist on cluster '%s'", ingressID, clusterKey)
		os.Exit(1)
	}

	if confirm.Confirm("delete ingress %s on cluster %s", ingressID, clusterKey) {
		r.Reporter.Debugf("Deleting ingress '%s' on cluster '%s'", ingress.ID(), clusterKey)
		err = r.OCMClient.DeleteIngress(cluster.ID(), ingress.ID())
		if err != nil {
			r.Reporter.Errorf("Failed to delete ingress '%s' on cluster '%s': %s",
				ingress.ID(), clusterKey, err)
			os.Exit(1)
		}
		r.Reporter.Infof("Successfully deleted ingress '%s' from cluster '%s'", ingressID, clusterKey)
	}
}
