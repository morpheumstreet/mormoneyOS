package tunnel

// LocaltunnelProvider exposes localhost via localtunnel (https://localtunnel.github.io/www/).
// Free, no auth. Uses npx localtunnel --port {port}; requires Node.js/npx in PATH.
func LocaltunnelProvider() TunnelProvider {
	return NewCommandTunnelProvider(
		"localtunnel",
		"npx localtunnel --port {port}",
		"https://",
		"npx",
		nil,
	)
}
