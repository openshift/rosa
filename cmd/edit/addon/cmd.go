/*
Copyright (c) 2021 Red Hat, Inc.

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

package addon

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	asv1 "github.com/openshift-online/ocm-sdk-go/addonsmgmt/v1"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/spf13/cobra"

	"github.com/openshift/rosa/pkg/arguments"
	"github.com/openshift/rosa/pkg/helper"
	"github.com/openshift/rosa/pkg/interactive"
	"github.com/openshift/rosa/pkg/ocm"
	"github.com/openshift/rosa/pkg/rosa"
)

var Cmd = &cobra.Command{
	Use:     "addon ID",
	Aliases: []string{"addons", "add-on", "add-ons"},
	Short:   "Edit add-on installation parameters on cluster",
	Long:    "Edit the parameters on installed Red Hat managed add-ons on a cluster",
	Example: `  # Edit the parameters of the Red Hat OpenShift logging operator add-on installation
  rosa edit addon --cluster=mycluster cluster-logging-operator`,
	Run:                run,
	DisableFlagParsing: true,
	Args: func(cmd *cobra.Command, argv []string) error {

		err := arguments.PreprocessUnknownFlagsWithId(cmd, argv)
		if err != nil {
			return fmt.Errorf("Expected exactly one command line parameter containing the id of the add-on."+
				" Error: %w", err)
		}

		err = arguments.ParseUnknownFlags(cmd, argv)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	ocm.AddClusterFlag(Cmd)
}

func run(cmd *cobra.Command, argv []string) {
	r := rosa.NewRuntime().WithAWS().WithOCM()
	defer r.Cleanup()

	// Parse out CLI flags, then override positional arguments
	_ = cmd.Flags().Parse(argv)
	argv = cmd.Flags().Args()
	addOnID := argv[0]

	clusterKey := r.GetClusterKey()

	cluster := r.FetchCluster()
	if cluster.State() != cmv1.ClusterStateReady {
		r.Reporter.Errorf("Cluster '%s' is not yet ready", clusterKey)
		os.Exit(1)
	}

	addonParameters, err := r.OCMClient.GetAddOnParameters(cluster.ID(), addOnID)
	if err != nil {
		r.Reporter.Errorf("Failed to get add-on '%s' parameters: %v", addOnID, err)
		os.Exit(1)
	}

	addOnInstallation, err := r.OCMClient.GetAddOnInstallation(cluster.ID(), addOnID)
	if err != nil {
		r.Reporter.Errorf("Failed to get add-on '%s' installation: %v", addOnID, err)
		os.Exit(1)
	}

	if addonParameters.Len() == 0 {
		r.Reporter.Errorf("Add-on '%s' has no parameters to edit", addOnID)
		os.Exit(1)
	}

	// Determine if all required parameters have already been set as flags and ensure
	// that interactive mode is enabled if they have not. If there are no parameters
	// set as flags, then we also ensure that interactive mode is enabled so that the
	// user gets prompted.
	if arguments.HasUnknownFlags() {
		addonParameters.Each(func(param *asv1.AddonParameter) bool {
			flag := cmd.Flags().Lookup(param.ID())
			if flag != nil && !param.Editable() {
				r.Reporter.Errorf("Parameter '%s' on addon '%s' cannot be modified", param.ID(), addOnID)
				os.Exit(1)
			}
			return true
		})
	} else {
		interactive.Enable()
	}

	var addonArguments []ocm.AddOnParam
	addonParameters.Each(func(param *asv1.AddonParameter) bool {
		// Find the installation parameter corresponding to the addon parameter
		var addOnInstallationParam *asv1.AddonInstallationParameter
		addOnInstallation.Parameters().Each(func(p *asv1.AddonInstallationParameter) bool {
			if p.Id() == param.ID() {
				addOnInstallationParam = p
				return false
			}
			return true
		})

		// If the parameter already exists in the cluster and is not editable, hide it
		if addOnInstallationParam != nil && !param.Editable() {
			return true
		}

		var val string
		var options []string
		var values []string

		parameterOptions, _ := param.GetOptions()

		for _, opt := range parameterOptions {
			options = append(options, opt.Name())
			values = append(values, opt.Value())
		}

		//Retrieve default value and set it first
		dflt := param.DefaultValue()
		if addOnInstallationParam != nil {
			dflt = addOnInstallationParam.Value()
		}
		val = dflt

		// If value is already set in the CLI, ignore interactive prompt
		flag := cmd.Flags().Lookup(param.ID())
		if flag != nil {
			val = flag.Value.String()
		}
		if interactive.Enabled() {
			val, err = interactive.GetAddonArgument(*param, dflt)
			if err != nil {
				r.Reporter.Errorf("%s", err)
				os.Exit(1)
			}
		}
		val = strings.Trim(val, " ")
		if val != "" && param.Validation() != "" {
			isValid, err := regexp.MatchString(param.Validation(), val)
			if err != nil || !isValid {
				r.Reporter.Errorf("Expected %v to match /%s/", val, param.Validation())
				os.Exit(1)
			}
		}

		if len(options) > 0 && !helper.Contains(values, val) {
			r.Reporter.Errorf("Expected %v to match one of the options /%v/", val, values)
			os.Exit(1)
		}
		addonArguments = append(addonArguments, ocm.AddOnParam{Key: param.ID(), Val: val})

		return true
	})

	r.Reporter.Debugf("Updating add-on parameters for '%s' on cluster '%s'", addOnID, clusterKey)
	err = r.OCMClient.UpdateAddOnInstallation(cluster.ID(), addOnID, addonArguments)
	if err != nil {
		r.Reporter.Errorf("Failed to update add-on installation '%s' for cluster '%s': %v", addOnID, clusterKey, err)
		os.Exit(1)
	}
	r.Reporter.Infof("Add-on '%s' is now updating. To check the status run 'rosa list addons -c %s'", addOnID, clusterKey)
}
