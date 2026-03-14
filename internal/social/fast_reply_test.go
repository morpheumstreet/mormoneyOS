package social

import "testing"

func TestClassifyFastReply(t *testing.T) {
	tests := []struct {
		content string
		fast    bool
		norm    string
	}{
		{"/status", true, "/status"},
		{"/help", true, "/help"},
		{"/balance", true, "/balance"},
		{"ping", true, "/ping"},
		{"!ping", true, "/ping"},
		{"PING", true, "/ping"},
		{"status", true, "/status"},
		{"credits?", true, "/balance"},
		{"credits", true, "/balance"},
		{"help", true, "/help"},
		{"?", true, "/help"},
		{"uptime", true, "/status"},
		{"!cmd balance", true, "/balance"},
		{"!cmd status", true, "/status"},
		{"@mybot status", true, "/status"},
		{"hello", false, ""},
		{"how are you?", false, ""},
		{"", false, ""},
		{"/foo", true, "/foo"}, // unknown command, still fast path (handler may return false)
	}
	for _, tt := range tests {
		fast, norm := ClassifyFastReply(tt.content)
		if fast != tt.fast || norm != tt.norm {
			t.Errorf("ClassifyFastReply(%q) = (%v, %q), want (%v, %q)", tt.content, fast, norm, tt.fast, tt.norm)
		}
	}
}
