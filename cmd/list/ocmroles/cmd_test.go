/*
Copyright (c) 2024 Red Hat, Inc.

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
package ocmroles

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openshift-online/ocm-sdk-go/testing"

	"github.com/openshift/rosa/pkg/aws"
	mock "github.com/openshift/rosa/pkg/aws"
	. "github.com/openshift/rosa/pkg/test"
)

func TestListOCMRoles(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "rosa list ocm-roles")
}

var _ = Describe("rosa list ocm-roles", func() {
	Context("printOCMRoles", func() {
		It("Prints header and rows with all columns", func() {
			roles := []aws.Role{
				{
					RoleName:      "my-OCM-Role",
					RoleARN:       "arn:aws:iam::111111111111:role/my-OCM-Role",
					Linked:        "Yes",
					Admin:         "Yes",
					ManagedPolicy: true,
					NoConsole:     "No",
				},
			}
			var buf bytes.Buffer
			printOCMRoles(&buf, roles)
			lines := strings.Split(buf.String(), "\n")
			Expect(len(lines)).To(BeNumerically(">=", 2))

			header := strings.Split(lines[0], "\t")
			Expect(header).To(HaveLen(6))
			Expect(header[4]).To(Equal("AWS Managed"))
			Expect(header[5]).To(Equal("CONSOLE ACCESS"))

			cols := strings.Split(lines[1], "\t")
			Expect(cols).To(HaveLen(6))
			Expect(cols[0]).To(Equal("my-OCM-Role"))
			Expect(cols[1]).To(Equal("arn:aws:iam::111111111111:role/my-OCM-Role"))
			Expect(cols[2]).To(Equal("Yes"))
			Expect(cols[3]).To(Equal("Yes"))
			Expect(cols[4]).To(Equal("Yes"))
			Expect(cols[5]).To(Equal("Yes"))
		})

		It("Shows No in CONSOLE ACCESS when NoConsole is Yes", func() {
			roles := []aws.Role{
				{
					RoleName:  "no-console-OCM-Role",
					RoleARN:   "arn:aws:iam::111111111111:role/no-console-OCM-Role",
					Linked:    "No",
					Admin:     "No",
					NoConsole: "Yes",
				},
			}
			var buf bytes.Buffer
			printOCMRoles(&buf, roles)
			lines := strings.Split(buf.String(), "\n")
			Expect(len(lines)).To(BeNumerically(">=", 2))
			cols := strings.Split(lines[1], "\t")
			Expect(cols).To(HaveLen(6))
			Expect(cols[4]).To(Equal("No"))
			Expect(cols[5]).To(Equal("No"))
		})

		It("Shows Yes in CONSOLE ACCESS when NoConsole is No", func() {
			roles := []aws.Role{
				{
					RoleName:  "console-OCM-Role",
					RoleARN:   "arn:aws:iam::111111111111:role/console-OCM-Role",
					Linked:    "No",
					Admin:     "No",
					NoConsole: "No",
				},
			}
			var buf bytes.Buffer
			printOCMRoles(&buf, roles)
			lines := strings.Split(buf.String(), "\n")
			Expect(len(lines)).To(BeNumerically(">=", 2))
			cols := strings.Split(lines[1], "\t")
			Expect(cols).To(HaveLen(6))
			Expect(cols[4]).To(Equal("No"))
			Expect(cols[5]).To(Equal("Yes"))
		})

		It("Shows AWS Managed correctly", func() {
			roles := []aws.Role{
				{
					RoleName:      "managed-OCM-Role",
					RoleARN:       "arn:aws:iam::111111111111:role/managed-OCM-Role",
					Linked:        "No",
					Admin:         "No",
					ManagedPolicy: true,
					NoConsole:     "No",
				},
				{
					RoleName:      "unmanaged-OCM-Role",
					RoleARN:       "arn:aws:iam::111111111111:role/unmanaged-OCM-Role",
					Linked:        "No",
					Admin:         "No",
					ManagedPolicy: false,
					NoConsole:     "Yes",
				},
			}
			var buf bytes.Buffer
			printOCMRoles(&buf, roles)
			lines := strings.Split(buf.String(), "\n")
			Expect(len(lines)).To(BeNumerically(">=", 3))

			managedCols := strings.Split(lines[1], "\t")
			Expect(managedCols).To(HaveLen(6))
			Expect(managedCols[4]).To(Equal("Yes"))
			Expect(managedCols[5]).To(Equal("Yes"))

			unmanagedCols := strings.Split(lines[2], "\t")
			Expect(unmanagedCols).To(HaveLen(6))
			Expect(unmanagedCols[4]).To(Equal("No"))
			Expect(unmanagedCols[5]).To(Equal("No"))
		})

		It("Shows NoConsole overriding Admin in display", func() {
			roles := []aws.Role{
				{
					RoleName:  "both-tags-OCM-Role",
					RoleARN:   "arn:aws:iam::111111111111:role/both-tags-OCM-Role",
					Linked:    "No",
					Admin:     "No",
					NoConsole: "Yes",
				},
			}
			var buf bytes.Buffer
			printOCMRoles(&buf, roles)
			lines := strings.Split(buf.String(), "\n")
			Expect(len(lines)).To(BeNumerically(">=", 2))
			cols := strings.Split(lines[1], "\t")
			Expect(cols).To(HaveLen(6))
			Expect(cols[0]).To(Equal("both-tags-OCM-Role"))
			Expect(cols[3]).To(Equal("No"))
			Expect(cols[5]).To(Equal("No"))
		})

		It("Prints only the header for empty roles", func() {
			var buf bytes.Buffer
			printOCMRoles(&buf, []aws.Role{})
			output := buf.String()
			Expect(output).To(ContainSubstring("ROLE NAME"))
			Expect(output).To(ContainSubstring("CONSOLE ACCESS"))
			lines := bytes.Split(buf.Bytes(), []byte("\n"))
			Expect(len(lines)).To(Equal(2))
		})
	})

	Context("listOCMRoles", func() {
		var (
			t          *TestingRuntime
			mockClient *mock.MockClient
		)

		BeforeEach(func() {
			t = NewTestRuntime()
			mockCtrl := gomock.NewController(GinkgoT())
			mockClient = mock.NewMockClient(mockCtrl)
			t.RosaRuntime.AWSClient = mockClient
		})

		It("Returns empty slice when no roles exist", func() {
			mockClient.EXPECT().ListOCMRoles().Return([]aws.Role{}, nil)
			roles, err := listOCMRoles(t.RosaRuntime)
			Expect(err).ToNot(HaveOccurred())
			Expect(roles).To(BeEmpty())
		})

		It("Returns error when AWS ListOCMRoles fails", func() {
			mockClient.EXPECT().ListOCMRoles().Return(nil, fmt.Errorf("aws error"))
			roles, err := listOCMRoles(t.RosaRuntime)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("aws error"))
			Expect(roles).To(BeNil())
		})

		It("Sets linked status and preserves NoConsole field", func() {
			awsRoles := []aws.Role{
				{
					RoleName:  "linked-OCM-Role",
					RoleARN:   "arn:aws:iam::111111111111:role/linked-OCM-Role",
					Admin:     "Yes",
					NoConsole: "Yes",
				},
				{
					RoleName:  "unlinked-OCM-Role",
					RoleARN:   "arn:aws:iam::222222222222:role/unlinked-OCM-Role",
					Admin:     "No",
					NoConsole: "No",
				},
			}
			mockClient.EXPECT().ListOCMRoles().Return(awsRoles, nil)

			// GET /api/accounts_mgmt/v1/current_account
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, `{
					"kind": "Account",
					"organization": {
						"id": "org123",
						"kind": "Organization"
					}
				}`),
			)
			// GET /api/accounts_mgmt/v1/organizations/org123/labels/sts_ocm_role
			t.ApiServer.AppendHandlers(
				RespondWithJSON(http.StatusOK, `{
					"kind": "Label",
					"key": "sts_ocm_role",
					"value": "arn:aws:iam::111111111111:role/linked-OCM-Role"
				}`),
			)

			roles, err := listOCMRoles(t.RosaRuntime)
			Expect(err).ToNot(HaveOccurred())
			Expect(roles).To(HaveLen(2))
			// Linked roles sort first
			Expect(roles[0].RoleName).To(Equal("linked-OCM-Role"))
			Expect(roles[0].Linked).To(Equal("Yes"))
			Expect(roles[0].NoConsole).To(Equal("Yes"))
			Expect(roles[1].RoleName).To(Equal("unlinked-OCM-Role"))
			Expect(roles[1].Linked).To(Equal("No"))
			Expect(roles[1].NoConsole).To(Equal("No"))
		})
	})
})
