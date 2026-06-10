package accountroles

// RequiresSTSExternalIDInTrustPolicy reports whether account role creation should embed
// sts:ExternalId in the trust policy when an external ID is provided by the user.
func RequiresSTSExternalIDInTrustPolicy(roleKey string) bool {
	switch roleKey {
	case InstallerAccountRole, SupportAccountRole:
		return true
	default:
		return false
	}
}
