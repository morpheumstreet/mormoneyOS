package social

import "strings"

// markdownV2Special are characters that must be escaped for Telegram MarkdownV2.
// See https://core.telegram.org/bots/api#formatting-options
const markdownV2Special = "_*[]()~`>#+-=|{}.!"

// EscapeMarkdownV2 escapes special symbols for Telegram MarkdownV2 parse mode.
// Use for dynamic content (numbers, addresses, user input) before inserting into
// a MarkdownV2 message. Formatting markers (*bold*, _italic_) should not be escaped.
func EscapeMarkdownV2(s string) string {
	var b strings.Builder
	for _, r := range s {
		if strings.ContainsRune(markdownV2Special, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
