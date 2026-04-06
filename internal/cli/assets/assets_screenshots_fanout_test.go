package assets

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rudrankriyam/App-Store-Connect-CLI/internal/asc"
)

func TestUploadScreenshotsFanoutUsesLocaleDirectoriesForResolvedVersion(t *testing.T) {
	rootDir := t.TempDir()
	enDir := filepath.Join(rootDir, "en-US", "iphone")
	frDir := filepath.Join(rootDir, "fr-FR", "iphone")
	if err := os.MkdirAll(enDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	if err := os.MkdirAll(frDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	writeAssetsTestPNGWithSize(t, enDir, "01-home.png", 1242, 2688)
	writeAssetsTestPNGWithSize(t, frDir, "01-home.png", 1242, 2688)

	origTransport := http.DefaultTransport
	http.DefaultTransport = assetsUploadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/version-1/appStoreVersionLocalizations":
			return assetsJSONResponse(http.StatusOK, `{"data":[{"type":"appStoreVersionLocalizations","id":"loc-en","attributes":{"locale":"en-US"}},{"type":"appStoreVersionLocalizations","id":"loc-fr","attributes":{"locale":"fr-FR"}},{"type":"appStoreVersionLocalizations","id":"loc-es","attributes":{"locale":"es-ES"}}],"links":{}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersionLocalizations/loc-en/appScreenshotSets":
			return assetsJSONResponse(http.StatusOK, `{"data":[],"links":{}}`)
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersionLocalizations/loc-fr/appScreenshotSets":
			return assetsJSONResponse(http.StatusOK, `{"data":[],"links":{}}`)
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})
	t.Cleanup(func() {
		http.DefaultTransport = origTransport
	})

	result, err := uploadScreenshotsFanout(context.Background(), screenshotUploadFanoutConfig{
		Client:      newAssetsUploadTestClient(t),
		AppID:       "123456789",
		Version:     "1.2.3",
		VersionID:   "version-1",
		Platform:    "IOS",
		RootPath:    rootDir,
		DisplayType: asc.CanonicalScreenshotDisplayTypeForAPI("APP_IPHONE_65"),
		DryRun:      true,
	})
	if err != nil {
		t.Fatalf("uploadScreenshotsFanout() error: %v", err)
	}

	if result.AppID != "123456789" {
		t.Fatalf("expected app ID 123456789, got %q", result.AppID)
	}
	if result.VersionID != "version-1" {
		t.Fatalf("expected version ID version-1, got %q", result.VersionID)
	}
	if len(result.Localizations) != 2 {
		t.Fatalf("expected 2 localization results, got %d", len(result.Localizations))
	}
	if result.Localizations[0].Locale != "en-US" || result.Localizations[0].VersionLocalizationID != "loc-en" {
		t.Fatalf("unexpected first localization result: %#v", result.Localizations[0])
	}
	if result.Localizations[1].Locale != "fr-FR" || result.Localizations[1].VersionLocalizationID != "loc-fr" {
		t.Fatalf("unexpected second localization result: %#v", result.Localizations[1])
	}
	for _, localization := range result.Localizations {
		if len(localization.Results) != 1 {
			t.Fatalf("expected one dry-run file result for %s, got %d", localization.Locale, len(localization.Results))
		}
		if localization.Results[0].State != "would-upload" {
			t.Fatalf("expected would-upload for %s, got %#v", localization.Locale, localization.Results[0])
		}
		if !strings.Contains(localization.Results[0].FilePath, localization.Locale) {
			t.Fatalf("expected file path to include locale %q, got %q", localization.Locale, localization.Results[0].FilePath)
		}
	}
}

func TestUploadScreenshotsFanoutErrorsWhenLocalLocaleHasNoRemoteMatch(t *testing.T) {
	rootDir := t.TempDir()
	jaDir := filepath.Join(rootDir, "ja", "iphone")
	if err := os.MkdirAll(jaDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}
	writeAssetsTestPNGWithSize(t, jaDir, "01-home.png", 1242, 2688)

	origTransport := http.DefaultTransport
	http.DefaultTransport = assetsUploadRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/appStoreVersions/version-1/appStoreVersionLocalizations":
			return assetsJSONResponse(http.StatusOK, `{"data":[{"type":"appStoreVersionLocalizations","id":"loc-en","attributes":{"locale":"en-US"}}],"links":{}}`)
		default:
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
			return nil, nil
		}
	})
	t.Cleanup(func() {
		http.DefaultTransport = origTransport
	})

	_, err := uploadScreenshotsFanout(context.Background(), screenshotUploadFanoutConfig{
		Client:      newAssetsUploadTestClient(t),
		AppID:       "123456789",
		Version:     "1.2.3",
		VersionID:   "version-1",
		Platform:    "IOS",
		RootPath:    rootDir,
		DisplayType: asc.CanonicalScreenshotDisplayTypeForAPI("APP_IPHONE_65"),
		DryRun:      true,
	})
	if err == nil {
		t.Fatal("expected missing remote localization error")
	}
	if !strings.Contains(err.Error(), `no matching App Store version localizations found for locales: ja`) {
		t.Fatalf("expected missing locale error, got %v", err)
	}
}
