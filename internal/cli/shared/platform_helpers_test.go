package shared

import "testing"

func TestDisplayPlatform(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"IOS", "iOS"},
		{"MAC_OS", "macOS"},
		{"TV_OS", "tvOS"},
		{"VISION_OS", "visionOS"},
		{"UNKNOWN_PLATFORM", "UNKNOWN_PLATFORM"},
		{"", ""},
		{"ios", "ios"}, // lowercase not mapped — pass through
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := DisplayPlatform(tt.input)
			if got != tt.want {
				t.Errorf("DisplayPlatform(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
