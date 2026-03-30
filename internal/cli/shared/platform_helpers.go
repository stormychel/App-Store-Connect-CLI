package shared

import (
	"fmt"
	"strings"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
)

var platformValues = map[string]asc.Platform{
	"IOS":       asc.PlatformIOS,
	"MAC_OS":    asc.PlatformMacOS,
	"TV_OS":     asc.PlatformTVOS,
	"VISION_OS": asc.PlatformVisionOS,
}

// NormalizePlatform validates and normalizes a platform string.
func NormalizePlatform(value string) (asc.Platform, error) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	if normalized == "" {
		return "", fmt.Errorf("--platform is required")
	}
	platform, ok := platformValues[normalized]
	if !ok {
		return "", fmt.Errorf("--platform must be one of: %s", strings.Join(platformList(), ", "))
	}
	return platform, nil
}

// PlatformList returns the allowed platform values.
func PlatformList() []string {
	return platformList()
}

func platformList() []string {
	return []string{"IOS", "MAC_OS", "TV_OS", "VISION_OS"}
}

// platformDisplayNames maps API enum values to human-readable names for table/markdown output.
var platformDisplayNames = map[string]string{
	"IOS":       "iOS",
	"MAC_OS":    "macOS",
	"TV_OS":     "tvOS",
	"VISION_OS": "visionOS",
}

// DisplayPlatform returns a human-readable platform name for table/markdown output.
// Unknown values pass through unchanged.
func DisplayPlatform(raw string) string {
	if display, ok := platformDisplayNames[raw]; ok {
		return display
	}
	return raw
}
