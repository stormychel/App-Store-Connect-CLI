package xcode

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
	localxcode "github.com/rudrankriyam/App-Store-Connect-CLI/internal/xcode"
)

func TestXcodeExportWaitRequiresDirectUpload(t *testing.T) {
	restore := overrideXcodeCommandTestHooks(t)
	defer restore()

	isDirectUploadExportOptionsFn = func(string) bool { return false }

	cmd := XcodeExportCommand()
	cmd.FlagSet.SetOutput(io.Discard)
	if err := cmd.FlagSet.Parse([]string{
		"--archive-path", "Demo.xcarchive",
		"--export-options", "ExportOptions.plist",
		"--ipa-path", "Demo.ipa",
		"--wait",
	}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	var runErr error
	_, stderr := captureCommandOutput(t, func() error {
		runErr = cmd.Exec(context.Background(), nil)
		return runErr
	})
	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatal("expected flag.ErrHelp when --wait is used without direct upload mode")
	}
	if !strings.Contains(stderr, "Error: --wait requires ExportOptions.plist with destination=upload") {
		t.Fatalf("expected direct upload usage error, got %q", stderr)
	}
}

func TestXcodeExportWaitRequiresPositivePollInterval(t *testing.T) {
	restore := overrideXcodeCommandTestHooks(t)
	defer restore()

	isDirectUploadExportOptionsFn = func(string) bool { return true }

	cmd := XcodeExportCommand()
	cmd.FlagSet.SetOutput(io.Discard)
	if err := cmd.FlagSet.Parse([]string{
		"--archive-path", "Demo.xcarchive",
		"--export-options", "ExportOptions.plist",
		"--ipa-path", "Demo.ipa",
		"--wait",
		"--poll-interval", "0s",
	}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	var runErr error
	_, stderr := captureCommandOutput(t, func() error {
		runErr = cmd.Exec(context.Background(), nil)
		return runErr
	})
	if !errors.Is(runErr, flag.ErrHelp) {
		t.Fatal("expected flag.ErrHelp for non-positive poll interval")
	}
	if !strings.Contains(stderr, "Error: --poll-interval must be greater than 0") {
		t.Fatalf("expected poll interval usage error, got %q", stderr)
	}
}

func TestXcodeExportAllowsPollIntervalWithoutWait(t *testing.T) {
	restore := overrideXcodeCommandTestHooks(t)
	defer restore()

	runExport = func(context.Context, localxcode.ExportOptions) (*localxcode.ExportResult, error) {
		return &localxcode.ExportResult{
			ArchivePath: "/tmp/Demo.xcarchive",
			IPAPath:     "/tmp/Demo.ipa",
			BundleID:    "com.example.demo",
			Version:     "1.2.3",
			BuildNumber: "42",
		}, nil
	}

	cmd := XcodeExportCommand()
	cmd.FlagSet.SetOutput(io.Discard)
	if err := cmd.FlagSet.Parse([]string{
		"--archive-path", "Demo.xcarchive",
		"--export-options", "ExportOptions.plist",
		"--ipa-path", "Demo.ipa",
		"--poll-interval", "0s",
		"--output", "json",
	}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	var runErr error
	stdout, stderr := captureCommandOutput(t, func() error {
		runErr = cmd.Exec(context.Background(), nil)
		return runErr
	})
	if runErr != nil {
		t.Fatalf("Exec() error: %v", runErr)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Fatal("expected JSON output")
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("expected no stderr output without --wait, got %q", stderr)
	}
}

func TestXcodeExportWaitPollsForUploadedBuild(t *testing.T) {
	restore := overrideXcodeCommandTestHooks(t)
	defer restore()

	isDirectUploadExportOptionsFn = func(string) bool { return true }
	runExport = func(context.Context, localxcode.ExportOptions) (*localxcode.ExportResult, error) {
		return &localxcode.ExportResult{
			ArchivePath: "/tmp/Demo.xcarchive",
			IPAPath:     "",
			BundleID:    "com.example.demo",
			Version:     "1.2.3",
			BuildNumber: "42",
		}, nil
	}
	inferArchivePlatformFn = func(string) (string, error) { return "IOS", nil }
	getASCClientFn = func() (*asc.Client, error) { return &asc.Client{}, nil }
	resolveAppIDWithExactLookupFn = func(_ context.Context, _ *asc.Client, app string) (string, error) {
		if app != "com.example.demo" {
			t.Fatalf("expected bundle ID app lookup, got %q", app)
		}
		return "123456789", nil
	}
	waitForBuildByNumberOrUploadFailureFn = func(_ context.Context, _ *asc.Client, appID, uploadID, version, buildNumber, platform string, pollInterval time.Duration) (*asc.BuildResponse, error) {
		if appID != "123456789" {
			t.Fatalf("expected resolved app ID, got %q", appID)
		}
		if uploadID != "" {
			t.Fatalf("expected no upload ID for xcode export wait, got %q", uploadID)
		}
		if version != "1.2.3" || buildNumber != "42" || platform != "IOS" {
			t.Fatalf("unexpected wait lookup params: version=%q build=%q platform=%q", version, buildNumber, platform)
		}
		if pollInterval != 5*time.Second {
			t.Fatalf("expected 5s poll interval, got %s", pollInterval)
		}
		return &asc.BuildResponse{
			Data: asc.Resource[asc.BuildAttributes]{
				ID: "build-123",
				Attributes: asc.BuildAttributes{
					Version:         "42",
					ProcessingState: asc.BuildProcessingStateValid,
				},
			},
		}, nil
	}
	waitForBuildProcessingFn = func(_ context.Context, _ *asc.Client, buildID string, pollInterval time.Duration) (*asc.BuildResponse, error) {
		if buildID != "build-123" {
			t.Fatalf("expected build-123, got %q", buildID)
		}
		if pollInterval != 5*time.Second {
			t.Fatalf("expected 5s poll interval, got %s", pollInterval)
		}
		return &asc.BuildResponse{
			Data: asc.Resource[asc.BuildAttributes]{
				ID: "build-123",
				Attributes: asc.BuildAttributes{
					Version:         "42",
					ProcessingState: asc.BuildProcessingStateValid,
				},
			},
		}, nil
	}

	cmd := XcodeExportCommand()
	cmd.FlagSet.SetOutput(io.Discard)
	if err := cmd.FlagSet.Parse([]string{
		"--archive-path", "Demo.xcarchive",
		"--export-options", "ExportOptions.plist",
		"--ipa-path", "Demo.ipa",
		"--wait",
		"--poll-interval", "5s",
		"--output", "json",
	}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	var runErr error
	stdout, stderr := captureCommandOutput(t, func() error {
		runErr = cmd.Exec(context.Background(), nil)
		return runErr
	})
	if runErr != nil {
		t.Fatalf("Exec() error: %v", runErr)
	}

	if strings.TrimSpace(stdout) == "" {
		t.Fatal("expected JSON output")
	}
	var payload struct {
		ArchivePath     string `json:"archive_path"`
		IPAPath         string `json:"ipa_path"`
		BuildID         string `json:"build_id"`
		ProcessingState string `json:"processing_state"`
		BundleID        string `json:"bundle_id"`
		Version         string `json:"version"`
		BuildNumber     string `json:"build_number"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error: %v\nstdout=%s", err, stdout)
	}
	if payload.BuildID != "build-123" {
		t.Fatalf("expected build_id build-123, got %q", payload.BuildID)
	}
	if payload.ProcessingState != asc.BuildProcessingStateValid {
		t.Fatalf("expected processing state VALID, got %q", payload.ProcessingState)
	}
	if !strings.Contains(stderr, "Waiting for build 42 (1.2.3) to appear in App Store Connect...") {
		t.Fatalf("expected discovery wait message, got %q", stderr)
	}
	if !strings.Contains(stderr, "Build build-123 discovered; waiting for processing...") {
		t.Fatalf("expected processing wait message, got %q", stderr)
	}
}

func TestXcodeExportWaitRejectsNilProcessedBuildResponse(t *testing.T) {
	restore := overrideXcodeCommandTestHooks(t)
	defer restore()

	isDirectUploadExportOptionsFn = func(string) bool { return true }
	runExport = func(context.Context, localxcode.ExportOptions) (*localxcode.ExportResult, error) {
		return &localxcode.ExportResult{
			ArchivePath: "/tmp/Demo.xcarchive",
			BundleID:    "com.example.demo",
			Version:     "1.2.3",
			BuildNumber: "42",
		}, nil
	}
	inferArchivePlatformFn = func(string) (string, error) { return "IOS", nil }
	getASCClientFn = func() (*asc.Client, error) { return &asc.Client{}, nil }
	resolveAppIDWithExactLookupFn = func(context.Context, *asc.Client, string) (string, error) {
		return "123456789", nil
	}
	waitForBuildByNumberOrUploadFailureFn = func(context.Context, *asc.Client, string, string, string, string, string, time.Duration) (*asc.BuildResponse, error) {
		return &asc.BuildResponse{
			Data: asc.Resource[asc.BuildAttributes]{
				ID: "build-123",
			},
		}, nil
	}
	waitForBuildProcessingFn = func(context.Context, *asc.Client, string, time.Duration) (*asc.BuildResponse, error) {
		return nil, nil
	}

	cmd := XcodeExportCommand()
	cmd.FlagSet.SetOutput(io.Discard)
	if err := cmd.FlagSet.Parse([]string{
		"--archive-path", "Demo.xcarchive",
		"--export-options", "ExportOptions.plist",
		"--ipa-path", "Demo.ipa",
		"--wait",
	}); err != nil {
		t.Fatalf("failed to parse flags: %v", err)
	}

	var runErr error
	_, _ = captureCommandOutput(t, func() error {
		runErr = cmd.Exec(context.Background(), nil)
		return runErr
	})
	if runErr == nil {
		t.Fatal("expected error for nil processed build response")
	}
	if !strings.Contains(runErr.Error(), "failed to resolve processed build state for build \"build-123\"") {
		t.Fatalf("expected nil processed build error, got %v", runErr)
	}
}

func overrideXcodeCommandTestHooks(t *testing.T) func() {
	t.Helper()

	originalRunArchive := runArchive
	originalRunExport := runExport
	originalIsDirectUpload := isDirectUploadExportOptionsFn
	originalInferArchivePlatform := inferArchivePlatformFn
	originalGetASCClient := getASCClientFn
	originalResolveAppID := resolveAppIDWithExactLookupFn
	originalWaitForDiscovery := waitForBuildByNumberOrUploadFailureFn
	originalWaitForProcessing := waitForBuildProcessingFn
	originalWaitTimeout := resolveXcodeExportWaitTimeoutFn

	return func() {
		runArchive = originalRunArchive
		runExport = originalRunExport
		isDirectUploadExportOptionsFn = originalIsDirectUpload
		inferArchivePlatformFn = originalInferArchivePlatform
		getASCClientFn = originalGetASCClient
		resolveAppIDWithExactLookupFn = originalResolveAppID
		waitForBuildByNumberOrUploadFailureFn = originalWaitForDiscovery
		waitForBuildProcessingFn = originalWaitForProcessing
		resolveXcodeExportWaitTimeoutFn = originalWaitTimeout
	}
}

func captureCommandOutput(t *testing.T, fn func() error) (string, string) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stderr pipe: %v", err)
	}

	os.Stdout = wOut
	os.Stderr = wErr

	outC := make(chan string)
	errC := make(chan string)

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rOut)
		_ = rOut.Close()
		outC <- buf.String()
	}()

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rErr)
		_ = rErr.Close()
		errC <- buf.String()
	}()

	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		_ = wOut.Close()
		_ = wErr.Close()
	}()

	_ = fn()

	_ = wOut.Close()
	_ = wErr.Close()

	stdout := <-outC
	stderr := <-errC

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	return stdout, stderr
}
