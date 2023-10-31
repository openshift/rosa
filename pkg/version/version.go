package version

import (
	gversion "github.com/hashicorp/go-version"
	"github.com/openshift/rosa/cmd/verify/rosa"
	"github.com/openshift/rosa/pkg/info"
	"github.com/openshift/rosa/pkg/reporter"
	"slices"
)

func SkipCommands() []string {
	return []string{"versions", "rosa-client"}
}

func ShouldRunCheck(commandName string) bool {
	return !slices.Contains(SkipCommands(), commandName)
}

func Check() {
	rprtr := reporter.CreateReporterOrExit()

	currVersion, err := gversion.NewVersion(info.Version)
	if err != nil {
		rprtr.Warnf("Could not verify the current version of ROSA.")
		rprtr.Warnf("You might be running on an outdated version. Make sure you are using the current version of ROSA.")
		return
	}

	latestVersionFromMirror, err := rosa.RetrieveLatestVersionFromMirror()
	if err != nil {
		rprtr.Warnf("There was a problem retrieving the latest version of ROSA.")
		rprtr.Warnf("You might be running on an outdated version. Make sure you are using the current version of ROSA.")
		return
	}

	if currVersion.LessThan(latestVersionFromMirror) {
		rprtr.Warnf("The current version (%s) is not up to date with latest released version (%s).",
			currVersion.Original(),
			latestVersionFromMirror.Original(),
		)

		rprtr.Warnf("It is recommended that you update to the latest version.")
	}
}
