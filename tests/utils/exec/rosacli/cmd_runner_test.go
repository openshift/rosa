// Copyright Red Hat
// SPDX-License-Identifier: Apache-2.0

package rosacli

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("redactedCMDString", func() {
	DescribeTable("redacts sensitive values while keeping the rest of the command visible",
		func(cmds []string, flags []string, expectedCmd string, expectedRedactOutput bool) {
			runner := NewRunner().Cmd(cmds...).CmdFlags(flags...)

			cmd, redactOutput := runner.redactedCMDString()

			Expect(cmd).To(Equal(expectedCmd))
			Expect(redactOutput).To(Equal(expectedRedactOutput))
		},
		Entry("positional key followed by its value",
			[]string{"config", "set"},
			[]string{"client_secret", "secret value"},
			"rosa config set client_secret <redacted>",
			false,
		),
		Entry("flag with an inline value",
			[]string{"create", "idp"},
			[]string{"-c", "cluster", "--client-secret=sekrit"},
			"rosa create idp -c cluster --client-secret=<redacted>",
			false,
		),
		Entry("flag with a separate value keeps trailing flags visible",
			[]string{"create", "idp"},
			[]string{"--client-secret", "sekrit", "--name", "github-idp"},
			"rosa create idp --client-secret <redacted> --name github-idp",
			false,
		),
		Entry("bare key is a read, so the output is withheld",
			[]string{"config", "get"},
			[]string{"access_token"},
			"rosa config get access_token",
			true,
		),
		Entry("bare key followed by a flag is still a read",
			[]string{"config", "get"},
			[]string{"refresh_token", "--debug"},
			"rosa config get refresh_token --debug",
			true,
		),
		Entry("multiple sensitive keys are all redacted",
			[]string{"config", "set"},
			[]string{"access_token", "tok", "--client-secret=sekrit"},
			"rosa config set access_token <redacted> --client-secret=<redacted>",
			false,
		),
		Entry("token_url is not a sensitive key",
			[]string{"config", "set"},
			[]string{"token_url", "https://example.com/token"},
			"rosa config set token_url https://example.com/token",
			false,
		),
		Entry("command with no sensitive key is returned unchanged",
			[]string{"list", "cluster"},
			[]string{"--region", "us-east-1"},
			"rosa list cluster --region us-east-1",
			false,
		),
	)
})

var _ = Describe("normalizeKey", func() {
	DescribeTable("resolves flag and positional spellings to the same key",
		func(elem string, expected string) {
			Expect(normalizeKey(elem)).To(Equal(expected))
		},
		Entry("positional key", "client_secret", "clientsecret"),
		Entry("long flag", "--client-secret", "clientsecret"),
		Entry("long flag with inline value", "--client-secret=sekrit", "clientsecret"),
		Entry("mixed case", "--Client-Secret", "clientsecret"),
		Entry("short flag", "-c", "c"),
		Entry("plain value", "us-east-1", "useast1"),
	)
})
