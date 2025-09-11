package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
)

func SkipNotHosted() {
	message := fmt.Sprintln("The test profile is not hosted")
	Skip(message)
}
func SkipNotClassic() {
	message := fmt.Sprintln("The test profile is not classic")
	Skip(message)
}

func SkipTestOnFeature(feature string) {
	message := fmt.Sprintf("The test profile is not configured with the feature: %s", feature)
	Skip(message)
}

func SkipNotMultiArch() {
	message := fmt.Sprintln("The cluster can not handle multiple CPU architecture")
	Skip(message)
}

func SkipNotSTS() {
	message := fmt.Sprintln("The test profile is not sts")
	Skip(message)
}
