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

package version

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zgalor/weberr"

	"github.com/openshift/rosa/pkg/info"
)

var Cmd = &cobra.Command{
	Use:   "version",
	Short: "Prints the version of the tool",
	Long:  "Prints the version number of the tool.",
	Run:   run,
}

const (
	releaseURL = "https://api.github.com/repos/openshift/rosa/releases/latest"
)

func run(cmd *cobra.Command, argv []string) {
	fmt.Fprintf(os.Stdout, "%s\n", info.Version)
	req, err := http.NewRequest("GET", releaseURL, nil)
	if err != nil {
		weberr.Errorf("Error setting up request for latest released rosa cli: %v", err)
		os.Exit(1)
	}
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		weberr.Errorf("Error while requesting latest released rosa cli: %v", err)
		os.Exit(1)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		weberr.Errorf("Error while requesting latest released rosa cli: %d %s", resp.StatusCode, resp.Status)
		os.Exit(1)
	}
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		weberr.Errorf("Error reading response body: %v", err)
	}
	result := make(map[string]interface{})
	json.Unmarshal(respBody, &result)

	if strings.Contains(result["tag_name"].(string), info.Version) {
		fmt.Fprintf(os.Stdout, "Your ROSA CLI is up to date.\n")
	} else {
		fmt.Fprintf(os.Stdout, "There is a newer release version, please consider updating: %s\n", result["html_url"].(string))
	}
}
