package common

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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
