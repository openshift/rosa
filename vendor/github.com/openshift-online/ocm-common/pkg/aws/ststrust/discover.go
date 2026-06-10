package ststrust

// DiscoverSTSExternalID returns a single external ID when discovery is unambiguous across
// installer and support trust policies. Empty policies are skipped. Returns ("", nil) when
// no ID should be sent to OCM (zero IDs or ambiguous multiple candidates without a unique intersection).
func DiscoverSTSExternalID(installerPolicy, supportPolicy string) (string, error) {
	installerIDs, err := CollectSTSExternalIDsFromTrustPolicy(installerPolicy)
	if err != nil {
		return "", err
	}
	supportIDs, err := CollectSTSExternalIDsFromTrustPolicy(supportPolicy)
	if err != nil {
		return "", err
	}
	union := setUnion(installerIDs, supportIDs)
	if len(union) == 0 {
		return "", nil
	}
	if len(union) == 1 {
		return union[0], nil
	}
	intersection := setIntersection(installerIDs, supportIDs)
	if len(intersection) == 1 {
		return intersection[0], nil
	}
	return "", nil
}
