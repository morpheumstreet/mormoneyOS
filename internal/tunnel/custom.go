package tunnel

// CustomProvider runs a user-supplied command with {port} and {host} placeholders.
func CustomProvider(startCommand, urlPattern, binary string) TunnelProvider {
	bin := binary
	if bin == "" {
		// Use first word of command as binary for IsAvailable check
		parts := splitFields(startCommand)
		if len(parts) > 0 {
			bin = parts[0]
		}
	}
	if urlPattern == "" {
		urlPattern = "https://"
	}
	return NewCommandTunnelProvider(
		"custom",
		startCommand,
		urlPattern,
		bin,
		nil,
	)
}

func splitFields(s string) []string {
	var out []string
	var buf []rune
	for _, r := range s {
		if r == ' ' || r == '\t' {
			if len(buf) > 0 {
				out = append(out, string(buf))
				buf = nil
			}
		} else {
			buf = append(buf, r)
		}
	}
	if len(buf) > 0 {
		out = append(out, string(buf))
	}
	return out
}
