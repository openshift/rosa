package helper

import (
	"fmt"
	"strings"

	//nolint:staticcheck
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck
	. "github.com/onsi/gomega"

	//nolint:staticcheck
	. "github.com/openshift/rosa/tests/utils/log"
)

// e is return value of Wait.Poll
// msg is the reason why time out
// the function assert return value of Wait.Poll, and expect NO error
// if e is Nil, just pass and nothing happen.
// if e is not Nil, will not print the default error message "timed out waiting for the condition"
//     because it causes RP AA not to analysis result exactly.
// if e is "timed out waiting for the condition" or "context deadline exceeded", it is replaced by msg.
// if e is not "timed out waiting for the condition", it print e and then case fails.

func AssertWaitPollNoErr(e error, msg string) {
	if e == nil {
		return
	}
	var err error
	if strings.Compare(e.Error(), "timed out waiting for the condition") == 0 ||
		strings.Compare(e.Error(), "context deadline exceeded") == 0 {
		err = fmt.Errorf("case: %v\nerror: %s", CurrentSpecReport().FullText(), msg)
	} else {
		err = fmt.Errorf("case: %v\nerror: %s", CurrentSpecReport().FullText(), e.Error())
	}
	Expect(err).NotTo(HaveOccurred())

}

// e is return value of Wait.Poll
// msg is the reason why not get
// the function assert return value of Wait.Poll, and expect error raised.
// if e is not Nil, just pass and nothing happen.
// if e is  Nil, will print expected error info and then case fails.

func AssertWaitPollWithErr(e error, msg string) {
	if e != nil {
		Logger.Infof("the error: %v", e)
		return
	}
	err := fmt.Errorf("case: %v\nexpected error not got because of %v", CurrentSpecReport().FullText(), msg)
	Expect(err).NotTo(HaveOccurred())

}

// Gingko helper function to:
// 1. check that an error occurred
// 2. check that the error message returned contains a substring
// 3. the error message check is case-insensitive
func ExpectErrorWithMessage(err error, msg string) {
	GinkgoHelper()
	Expect(err).To(HaveOccurred())
	errMsg := err.Error()
	Expect(strings.ToLower(errMsg)).Should(ContainSubstring(strings.ToLower(msg)))
}

// Gingko helper function to:
// 1. check that an error occurred
// 2. check that the error message returned matches a regex pattern
// 3. the error message check is case-insensitive
func ExpectErrorWithPattern(err error, pattern string) {
	GinkgoHelper()
	Expect(err).To(HaveOccurred())
	errMsg := err.Error()
	Expect(strings.ToLower(errMsg)).Should(MatchRegexp("(?i)%s", pattern))
}
