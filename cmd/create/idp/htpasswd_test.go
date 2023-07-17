/*
Copyright (c) 2022 Red Hat, Inc.

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

package idp

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("IDP Tests", func() {

	Describe("HTPasswd Tests", func() {

		wellformedContent :=
			"eleven:$apr1$hRY7OJWH$km1EYH.UIRjp6CzfZQz/g1" + "\n" +
				"vecna:$apr1$Q58SO804$B/fECNWfn5xkJXJLvu0mF"

		wellformedUserList := map[string]string{
			"eleven": "$apr1$hRY7OJWH$km1EYH.UIRjp6CzfZQz/g1",
			"vecna":  "$apr1$Q58SO804$B/fECNWfn5xkJXJLvu0mF",
		}

		malformedContent :=
			"MissingColon$apr1$hRY7OJWH$km1EYH.UIRjp6CzfZQz/g1" + "\n" +
				"MissingPasswordAfterColon:"

		DescribeTable("Htpasswd FileParser Tests",
			func(fileContent string, exceptedUserList map[string]string, errorExcepted bool) {

				fileName := "DoesNotExistYet"

				//Create Temp File with Input content
				if fileContent != "" {
					file, err := CreateTmpFile(fileContent)
					Expect(err).NotTo(HaveOccurred())

					fileName = file.Name()
					defer os.Remove(fileName)
				}

				//parse Temp File
				userList := make(map[string]string)
				err := parseHtpasswordFile(&userList, fileName)

				// Compare Results

				if errorExcepted {
					Expect(err).To(HaveOccurred())
				} else {
					Expect(err).NotTo(HaveOccurred())
				}

				fmt.Println("Expected", exceptedUserList)
				fmt.Println("Got", userList)
				fmt.Println("Equal", reflect.DeepEqual(userList, exceptedUserList))
				if exceptedUserList != nil {
					Expect(reflect.DeepEqual(userList, exceptedUserList)).To(BeTrue())
				}

			},
			Entry("Wellformed HTPassword File Test",
				wellformedContent, wellformedUserList, false),
			Entry("Malformed HTPasswd File Test",
				malformedContent, nil, true),
			Entry("Nonexistent File Test",
				"", nil, true),
		)
	})

})

func CreateTmpFile(content string) (*os.File, error) {
	// Create a temporary file
	file, err := ioutil.TempFile("", "temp-*.txt")
	if err != nil {
		return nil, err
	}

	// write to file
	_, err = file.WriteString(content)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	// Close the file
	err = file.Close()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return file, nil
}
