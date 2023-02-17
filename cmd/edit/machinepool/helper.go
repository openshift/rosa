package machinepool

import (
	"fmt"
	"os"
	"strings"

	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/openshift/rosa/pkg/interactive"
	rprtr "github.com/openshift/rosa/pkg/reporter"
	"github.com/openshift/rosa/pkg/rosa"
	"github.com/spf13/cobra"
)

func getTaints(cmd *cobra.Command, r *rosa.Runtime, inputTaints []*cmv1.Taint) []*cmv1.TaintBuilder {
	taintBuilders := []*cmv1.TaintBuilder{}
	taints := args.taints
	var err error
	if interactive.Enabled() {
		if taints == "" {
			for _, taint := range inputTaints {
				if taint == nil {
					continue
				}
				if taints != "" {
					taints += ","
				}
				taints += fmt.Sprintf("%s=%s:%s", taint.Key(), taint.Value(), taint.Effect())
			}
		}
		taints, err = interactive.GetString(interactive.Input{
			Question: "Taints",
			Help:     cmd.Flags().Lookup("taints").Usage,
			Default:  taints,
		})
		if err != nil {
			r.Reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
			os.Exit(1)
		}
	}
	taints = strings.Trim(taints, " ")
	if taints != "" {
		for _, taint := range strings.Split(taints, ",") {
			if !strings.Contains(taint, "=") || !strings.Contains(taint, ":") {
				r.Reporter.Errorf("Expected key=value:scheduleType format for taints")
				os.Exit(1)
			}
			tokens := strings.FieldsFunc(taint, Split)
			taintBuilders = append(taintBuilders, cmv1.NewTaint().Key(tokens[0]).Value(tokens[1]).Effect(tokens[2]))
		}
	}
	return taintBuilders
}

func getLabels(cmd *cobra.Command,
	reporter *rprtr.Object,
	existingLabels map[string]string) map[string]string {
	var err error
	labels := args.labels
	labelMap := make(map[string]string)
	if interactive.Enabled() {
		if labels == "" {
			for lk, lv := range existingLabels {
				if labels != "" {
					labels += ","
				}
				labels += fmt.Sprintf("%s=%s", lk, lv)
			}
		}
		labels, err = interactive.GetString(interactive.Input{
			Question: "Labels",
			Help:     cmd.Flags().Lookup("labels").Usage,
			Default:  labels,
		})
		if err != nil {
			reporter.Errorf("Expected a valid comma-separated list of attributes: %s", err)
			os.Exit(1)
		}
	}

	labels = strings.Trim(labels, " ")
	if labels != "" {
		for _, label := range strings.Split(labels, ",") {
			if !strings.Contains(label, "=") {
				reporter.Errorf("Expected key=value format for labels")
				os.Exit(1)
			}
			tokens := strings.Split(label, "=")
			labelMap[strings.TrimSpace(tokens[0])] = strings.TrimSpace(tokens[1])
		}
	}
	return labelMap
}
