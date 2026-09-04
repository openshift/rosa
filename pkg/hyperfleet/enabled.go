package hyperfleet

// Enabled returns true when the CLI should dispatch to the Platform API (v2) path.
// Currently always returns false — the detection logic will be added when the
// hyperfleet integration lands.
func Enabled() bool {
	return false
}
