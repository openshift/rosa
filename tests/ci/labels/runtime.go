package labels

import (
	. "github.com/onsi/ginkgo/v2"
)

// Pre check
var E2ECommit = Label("e2e-commit")

// day1/day1-post and day2
var Day1 = Label("day1")
var Day1Prepare = Label("day1-prepare")
var Day1Negative = Label("day1-negative")
var Day1Post = Label("day1-post")
var Day2 = Label("day2")
var Upgrade = Label("upgrade")

// day3 : the test cases will destroy default resource
var Day3 = Label("day3")

// destroy
var Destroy = Label("destroy")
