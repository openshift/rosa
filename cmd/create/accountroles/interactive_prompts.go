/*
Copyright (c) 2026 Red Hat, Inc.

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

package accountroles

import (
	"github.com/openshift/rosa/pkg/interactive"
)

const (
	questionCreateClassic  = "Create Classic account roles"
	questionCreateHostedCP = "Create Hosted CP account roles"
)

// promptClassicAndHostedCP applies interactive rules for classic vs hosted-cp
// account-role creation (Polarion OCP-43071 steps 3–4).
//
//   - Classic is asked only when neither --classic nor --hosted-cp was set.
//   - Hosted CP is asked only when --hosted-cp was not set and --classic was not
//     changed on the command line.
func promptClassicAndHostedCP(
	p interactive.Prompter,
	classicHelp string,
	hostedCPHelp string,
	createClassic bool,
	createHostedCP bool,
	isClassicValueSet bool,
	isHostedCPValueSet bool,
	classicFlagChanged bool,
	sharedVpcRolesDefault bool,
) (bool, bool, bool, bool, error) {
	var err error

	if interactive.Enabled() && !isClassicValueSet && !isHostedCPValueSet {
		createClassic, err = p.GetBool(interactive.Input{
			Question: questionCreateClassic,
			Help:     classicHelp,
			Default:  true,
			Required: false,
		})
		if err != nil {
			return createClassic, createHostedCP, isClassicValueSet, isHostedCPValueSet, err
		}
		isClassicValueSet = true
	}

	if interactive.Enabled() && !isHostedCPValueSet && !classicFlagChanged {
		createHostedCP, err = p.GetBool(interactive.Input{
			Question: questionCreateHostedCP,
			Help:     hostedCPHelp,
			Default:  sharedVpcRolesDefault || !createClassic,
			Required: false,
		})
		if err != nil {
			return createClassic, createHostedCP, isClassicValueSet, isHostedCPValueSet, err
		}
		isHostedCPValueSet = true
	}

	return createClassic, createHostedCP, isClassicValueSet, isHostedCPValueSet, nil
}
