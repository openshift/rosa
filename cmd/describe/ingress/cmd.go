package ingress

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/output"
	"github.com/openshift/rosa/pkg/rosa"
)

// Regular expression to used to make sure that the identifier given by the
// user is safe and that it there is no risk of SQL injection:
var ingressKeyRE = regexp.MustCompile(`^[a-z0-9]{3,5}$`)

var Cmd = &cobra.Command{
	Use:     "ingress",
	Short:   "Show details of the specified ingress within cluster",
	Example: `rosa describe ingress <ingress_id> -c mycluster`,
	Run:     run,
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
	output.AddFlag(Cmd)
}

func run(_ *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithOCM()
	defer r.Cleanup()

	ingressKey := argv[0]
	if !ingressKeyRE.MatchString(ingressKey) {
		r.Reporter.Errorf(
			"Ingress  identifier '%s' isn't valid: it must contain only letters or digits",
			ingressKey,
		)
		os.Exit(1)
	}

	cluster := r.FetchCluster()

	ingress, err := r.OCMClient.GetIngress(cluster.ID(), ingressKey)
	if err != nil {
		r.Reporter.Errorf("Failed to fetch ingress: %v", err)
		os.Exit(1)
	}
	if output.HasFlag() {
		var b bytes.Buffer
		err := cmv1.MarshalIngress(ingress, &b)
		if err != nil {
			r.Reporter.Errorf("Failed to generate output for ingress '%s': %v", ingress.ID(), err)
			os.Exit(1)
		}
		ret := make(map[string]interface{})
		err = json.Unmarshal(b.Bytes(), &ret)
		if err != nil {
			r.Reporter.Errorf("Failed to generate output for ingress '%s': %v", ingress.ID(), err)
			os.Exit(1)
		}
		err = output.Print(ret)
		if err != nil {
			r.Reporter.Errorf("Failed to output ingress '%s': %v", ingress.ID(), err)
			os.Exit(1)
		}
		return
	}
	entries := generateEntriesOutput(cluster, ingress)
	ingressOutput := ""
	keys := helper.MapKeys(entries)
	sort.Strings(keys)
	minWidth := getMinWidth(keys)
	for _, key := range keys {
		ingressOutput += fmt.Sprintf("%s: %s\n", key, strings.Repeat(" ", minWidth-len(key))+entries[key])
	}
	fmt.Print(ingressOutput)
}

// Min width is defined as the length of the longest string
func getMinWidth(keys []string) int {
	minWidth := 0
	for _, key := range keys {
		if len(key) > minWidth {
			minWidth = len(key)
		}
	}
	return minWidth
}

func generateEntriesOutput(cluster *cmv1.Cluster, ingress *cmv1.Ingress) map[string]string {
	private := false
	if ingress.Listening() == cmv1.ListeningMethodInternal {
		private = true
	}
	entries := map[string]string{
		"ID":         ingress.ID(),
		"Cluster ID": cluster.ID(),
		"Default":    strconv.FormatBool(ingress.Default()),
		"Private":    strconv.FormatBool(private),
		"LB-Type":    string(ingress.LoadBalancerType()),
	}
	// These are only available for ingress v2
	wildcardPolicy := string(ingress.RouteWildcardPolicy())
	if wildcardPolicy != "" {
		entries["Wildcard Policy"] = string(ingress.RouteWildcardPolicy())
	}
	namespaceOwnershipPolicy := string(ingress.RouteNamespaceOwnershipPolicy())
	if namespaceOwnershipPolicy != "" {
		entries["Namespace Ownership Policy"] = namespaceOwnershipPolicy
	}
	routeSelectors := ""
	if len(ingress.RouteSelectors()) > 0 {
		routeSelectors = fmt.Sprintf("%v", ingress.RouteSelectors())
	}
	if routeSelectors != "" {
		entries["Route Selectors"] = routeSelectors
	}
	excludedNamespaces := helper.SliceToSortedString(ingress.ExcludedNamespaces())
	if excludedNamespaces != "" {
		entries["Excluded Namespaces"] = excludedNamespaces
	}
	componentRoutes := ""
	for component, value := range ingress.ComponentRoutes() {
		keys := helper.MapKeys(entries)
		minWidth := getMinWidth(keys)
		depth := 4
		componentRouteEntries := map[string]string{
			"Hostname":       value.Hostname(),
			"TLS Secret Ref": value.TlsSecretRef(),
		}
		componentRoutes += fmt.Sprintf("%s: \n", strings.Repeat(" ", depth)+component)
		depth *= 2
		for key, entry := range componentRouteEntries {
			componentRoutes += fmt.Sprintf(
				"%s: %s\n",
				strings.Repeat(" ", depth)+key,
				strings.Repeat(" ", minWidth-len(key)-depth)+entry,
			)
		}
	}
	if componentRoutes != "" {
		componentRoutes = fmt.Sprintf("\n%s", componentRoutes)
		//remove extra \n at the end
		componentRoutes = componentRoutes[:len(componentRoutes)-1]
		entries["Component Routes"] = componentRoutes
	}
	return entries
}
