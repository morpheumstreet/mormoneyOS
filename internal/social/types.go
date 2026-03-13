package social

import "fmt"

const (
	MaxMessageLength   = 4096 // Align with Telegram; truncate for others
	MaxOutboundPerHour = 60
)

// ValidateOutbound checks size limits before send.
func ValidateOutbound(content string) error {
	if len(content) > MaxMessageLength {
		return fmt.Errorf("message too long: %d bytes (max %d)", len(content), MaxMessageLength)
	}
	return nil
}
