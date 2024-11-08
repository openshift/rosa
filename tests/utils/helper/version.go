package helper

import "github.com/Masterminds/semver"

func FindUpgradeVersions(versionList []string, clusterVersion string) (
	yStreamVersions []string, zStreamVersions []string, err error) {
	clusterBaseVersionSemVer, err := semver.NewVersion(clusterVersion)
	if err != nil {
		return yStreamVersions, zStreamVersions, err
	}

	for _, version := range versionList {
		baseVersionSemVer, err := semver.NewVersion(version)
		if err != nil {
			return yStreamVersions, zStreamVersions, err
		}
		if baseVersionSemVer.Minor() == clusterBaseVersionSemVer.Minor() {
			zStreamVersions = append(zStreamVersions, version)
		}

		if baseVersionSemVer.Minor() > clusterBaseVersionSemVer.Minor() {
			yStreamVersions = append(yStreamVersions, version)
		}
	}
	return yStreamVersions, zStreamVersions, err
}
