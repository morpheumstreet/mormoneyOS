package tunnel

// BoreProvider exposes localhost via bore (https://github.com/ekzhang/bore).
// Free, no auth. Command: bore local {port} --to bore.pub
func BoreProvider() TunnelProvider {
	return NewCommandTunnelProvider(
		"bore",
		"bore local {port} --to bore.pub",
		"https://",
		"bore",
		nil,
	)
}
