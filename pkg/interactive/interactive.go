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

package interactive

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh/terminal"
)

const inputPrefix = "\033[0;36m?\033[m "

// Gets user input from the command line
func GetInput(q string) (a string, err error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s%s: ", inputPrefix, q)
	text, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	a = strings.Trim(text, "\n")
	return
}

func GetPassword(q string) (a string, err error) {
	fmt.Printf("%s%s: ", inputPrefix, q)
	text, err := terminal.ReadPassword(syscall.Stdin)
	fmt.Println("")
	if err != nil {
		return
	}
	a = string(text)
	return
}
