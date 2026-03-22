package shared

import (
	"flag"
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func TestRenameLeafVerbKeepsDeprecatedAliasFlagSetPath(t *testing.T) {
	fs := flag.NewFlagSet("apps get", flag.ContinueOnError)
	id := ""
	fs.StringVar(&id, "id", "", "App ID")

	cmd := &ffcli.Command{
		Name:       "get",
		ShortUsage: "asc apps get --id APP_ID",
		ShortHelp:  "Get an app.",
		FlagSet:    fs,
	}

	alias, _ := renameLeafVerb(cmd, "asc apps get", "view")
	if alias == nil {
		t.Fatal("expected deprecated alias command")
	}
	if cmd.FlagSet == nil {
		t.Fatal("expected canonical flagset to remain available")
	}
	if alias.FlagSet == nil {
		t.Fatal("expected deprecated alias flagset to be cloned")
	}

	if got := cmd.FlagSet.Name(); got != "apps view" {
		t.Fatalf("expected canonical flagset name %q, got %q", "apps view", got)
	}
	if got := alias.FlagSet.Name(); got != "apps get" {
		t.Fatalf("expected alias flagset name %q, got %q", "apps get", got)
	}
	if alias.FlagSet == cmd.FlagSet {
		t.Fatal("expected deprecated alias to clone the canonical flagset")
	}

	if err := alias.FlagSet.Parse([]string{"--id", "app-123"}); err != nil {
		t.Fatalf("expected alias parse to succeed, got %v", err)
	}
	if id != "app-123" {
		t.Fatalf("expected alias parse to update shared flag value, got %q", id)
	}

	fs = flag.NewFlagSet("apps get", flag.ContinueOnError)
	fs.StringVar(&id, "id", "", "App ID")
	cmd = &ffcli.Command{
		Name:       "get",
		ShortUsage: "asc apps get --id APP_ID",
		ShortHelp:  "Get an app.",
		FlagSet:    fs,
	}
	alias, _ = renameLeafVerb(cmd, "asc apps get", "view")
	if alias == nil || alias.FlagSet == nil {
		t.Fatal("expected fresh deprecated alias flagset for usage test")
	}
	if got := alias.FlagSet.Name(); got != "apps get" {
		t.Fatalf("expected fresh alias flagset name %q, got %q", "apps get", got)
	}
}
