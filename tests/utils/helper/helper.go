package helper

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"

	"github.com/Masterminds/semver"

	"github.com/openshift/rosa/tests/utils/constants"
)

func ReadENVWithDefaultValue(envName string, fallback string) string {
	if os.Getenv(envName) != "" {
		return os.Getenv(envName)
	}
	return fallback
}

func RandomInt(max int) int {
	val, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(err)
	}
	return int(val.Int64())
}

func GetConsoleUrlBasedOnEnv(ocmApi string) string {
	switch ocmApi {
	case constants.StageEnv:
		return constants.StageURL
	case constants.ProductionEnv:
		return constants.ProductionURL
	default:
		return ""
	}
}

func GetMajorMinorFromVersion(version string) (major int64, minor int64, majorMinorVersion string, err error) {
	var semverVersion *semver.Version
	if semverVersion, err = semver.NewVersion(version); err != nil {
		return
	}
	major = semverVersion.Major()
	minor = semverVersion.Minor()
	majorMinorVersion = fmt.Sprintf("%d.%d", major, minor)
	return
}
