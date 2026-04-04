package cmdtest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestLocalizationsSupportedLocalesSuccess(t *testing.T) {
	setupAuth(t)

	originalTransport := http.DefaultTransport
	t.Cleanup(func() {
		http.DefaultTransport = originalTransport
	})

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet || req.URL.Path != "/v1/appStoreVersions/ver-1/appStoreVersionLocalizations" {
			t.Fatalf("unexpected request: %s %s", req.Method, req.URL.String())
		}

		body := `{"data":[
			{"type":"appStoreVersionLocalizations","id":"loc-en","attributes":{"locale":"en-US","keywords":"baseline,copy"}},
			{"type":"appStoreVersionLocalizations","id":"loc-bn","attributes":{"locale":"bn-BD","keywords":"bangla,copy"}}
		],"links":{}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)

	stdout, stderr := captureOutput(t, func() {
		if err := root.Parse([]string{
			"localizations", "supported-locales",
			"--version", "ver-1",
			"--output", "json",
		}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var out struct {
		VersionID       string `json:"versionId"`
		TotalSupported  int    `json:"totalSupported"`
		ConfiguredCount int    `json:"configuredCount"`
		Locales         []struct {
			Locale         string `json:"locale"`
			Name           string `json:"name"`
			Configured     bool   `json:"configured"`
			LocalizationID string `json:"localizationId,omitempty"`
		} `json:"locales"`
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("stdout should be valid json: %v\nstdout=%q", err, stdout)
	}

	if out.VersionID != "ver-1" {
		t.Fatalf("expected versionId ver-1, got %q", out.VersionID)
	}
	if out.TotalSupported != 50 {
		t.Fatalf("expected totalSupported 50, got %d", out.TotalSupported)
	}
	if out.ConfiguredCount != 2 {
		t.Fatalf("expected configuredCount 2, got %d", out.ConfiguredCount)
	}

	var seenBangla, seenSlovenian bool
	for _, item := range out.Locales {
		switch item.Locale {
		case "bn-BD":
			seenBangla = true
			if !item.Configured {
				t.Fatalf("expected bn-BD to be configured, got %+v", item)
			}
			if item.LocalizationID != "loc-bn" {
				t.Fatalf("expected bn-BD localization id loc-bn, got %+v", item)
			}
		case "sl-SI":
			seenSlovenian = true
			if item.Name == "" {
				t.Fatalf("expected sl-SI to include a display name, got %+v", item)
			}
			if item.Configured {
				t.Fatalf("expected sl-SI to be unconfigured, got %+v", item)
			}
		}
	}

	if !seenBangla {
		t.Fatalf("expected locales to include bn-BD, got %+v", out.Locales)
	}
	if !seenSlovenian {
		t.Fatalf("expected locales to include sl-SI, got %+v", out.Locales)
	}
}
