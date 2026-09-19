package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestFishCompletionsCoverCommandsAndFlags(t *testing.T) {
	script := fishCompletions()
	for _, want := range []string{
		"__fish_use_subcommand",
		"-a upload",
		"-a export",
		"-a completions",
		"-l api-url",
		"-l override-id",
		"-l out",
		"-l force",
		"fish bash zsh",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("fish completions are missing %q", want)
		}
	}
}

func TestBashCompletionsCoverCommandsAndFlags(t *testing.T) {
	script := bashCompletions()
	for _, want := range []string{
		"complete -F _prglv prglv",
		"upload export completions version help",
		"--api-url",
		"--override-id",
		"--out",
	} {
		if !strings.Contains(script, want) {
			t.Errorf("bash completions are missing %q", want)
		}
	}
}

func TestBashCompletionsPassSyntaxCheck(t *testing.T) {
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash is not installed")
	}
	cmd := exec.Command(bash, "-n")
	cmd.Stdin = strings.NewReader(bashCompletions())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bash -n: %v\n%s", err, out)
	}
}

func TestFishCompletionsPassSyntaxCheck(t *testing.T) {
	fish, err := exec.LookPath("fish")
	if err != nil {
		t.Skip("fish is not installed")
	}
	cmd := exec.Command(fish, "-n")
	cmd.Stdin = strings.NewReader(fishCompletions())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fish -n: %v\n%s", err, out)
	}
}
