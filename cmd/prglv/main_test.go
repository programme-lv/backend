package main

import (
	"errors"
	"flag"
	"testing"
)

func TestParseFlagsUpload(t *testing.T) {
	p, positional, err := parseFlags(spec("upload"), []string{
		"--api-url", "http://example", "--json", "task.zip",
	})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if p.strs["api-url"] != "http://example" {
		t.Fatalf("api-url = %q", p.strs["api-url"])
	}
	if !p.bools["json"] {
		t.Fatal("json flag was not set")
	}
	if len(positional) != 1 || positional[0] != "task.zip" {
		t.Fatalf("positional = %v", positional)
	}
}

func TestParseFlagsAllowsTrailingFlags(t *testing.T) {
	p, positional, err := parseFlags(spec("upload"), []string{
		"task.zip", "--api-url", "http://example", "--json",
	})
	if err != nil {
		t.Fatalf("parseFlags: %v", err)
	}
	if p.strs["api-url"] != "http://example" || !p.bools["json"] {
		t.Fatalf("flags = %#v", p)
	}
	if len(positional) != 1 || positional[0] != "task.zip" {
		t.Fatalf("positional = %v", positional)
	}
}

func TestParseFlagsHelp(t *testing.T) {
	_, _, err := parseFlags(spec("upload"), []string{"--help"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("err = %v, want flag.ErrHelp", err)
	}
}

func TestParseFlagsUnknown(t *testing.T) {
	_, _, err := parseFlags(spec("upload"), []string{"--nope"})
	if !errors.Is(err, errUsage) {
		t.Fatalf("err = %v, want errUsage", err)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	if err := run([]string{"nope"}); !errors.Is(err, errUsage) {
		t.Fatalf("err = %v, want errUsage", err)
	}
}

func TestResolveAPIURLPrecedence(t *testing.T) {
	t.Setenv("PRGLV_API_URL", "http://env")
	t.Setenv("API_PUBLIC_BASE_URL", "http://public")

	if got := resolveAPIURL("http://flag"); got != "http://flag" {
		t.Fatalf("flag value ignored: %q", got)
	}
	if got := resolveAPIURL(""); got != "http://env" {
		t.Fatalf("env value ignored: %q", got)
	}
}

func TestResolveAPIKeyPrecedence(t *testing.T) {
	t.Setenv("PRGLV_API_KEY", "prglv-key")
	t.Setenv("ADMIN_API_KEY", "admin-key")

	if got := resolveAPIKey("flag-key"); got != "flag-key" {
		t.Fatalf("flag value ignored: %q", got)
	}
	if got := resolveAPIKey(""); got != "prglv-key" {
		t.Fatalf("PRGLV_API_KEY ignored: %q", got)
	}
}
