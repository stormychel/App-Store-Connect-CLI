package cmdtest

import (
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestScreenshotsUploadAppScopedModeRequiresVersionSelector(t *testing.T) {
	stdout, stderr, runErr := runRootCommand(t, []string{
		"screenshots", "upload",
		"--app", "123456789",
		"--path", "./screenshots",
		"--device-type", "IPHONE_65",
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", runErr)
	}
	if !strings.Contains(stderr, "Error: --version or --version-id is required with --app") {
		t.Fatalf("expected missing app-scoped version selector error, got %q", stderr)
	}
}

func TestScreenshotsUploadRejectsMixingDirectAndAppScopedSelectors(t *testing.T) {
	stdout, stderr, runErr := runRootCommand(t, []string{
		"screenshots", "upload",
		"--version-localization", "LOC_ID",
		"--app", "123456789",
		"--version", "1.2.3",
		"--path", "./screenshots",
		"--device-type", "IPHONE_65",
	})

	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatalf("expected flag.ErrHelp, got %v", runErr)
	}
	if !strings.Contains(stderr, "Error: --version-localization cannot be combined with --app, --version, --version-id, or --platform") {
		t.Fatalf("expected direct/app-scoped selector conflict error, got %q", stderr)
	}
}
