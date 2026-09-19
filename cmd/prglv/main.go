// Command prglv is the programme.lv administration CLI.
//
// It talks to a running backend over the admin HTTP API. It packages a
// TaskZip directory or .zip and uploads it as a new task, and downloads
// existing tasks as TaskZip archives.
//
// The backend base URL comes from --api-url, then PRGLV_API_URL, then
// API_PUBLIC_BASE_URL, defaulting to http://localhost:8080. The admin key
// comes from --api-key, then PRGLV_API_KEY, then ADMIN_API_KEY.
//
// Generate shell completions with "prglv completions <shell>".
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	version       = "0.1.0"
	defaultAPIURL = "http://localhost:8080"
)

// errUsage marks an error caused by bad command-line input.
var errUsage = errors.New("usage")

// flagSpec describes one command flag.
// The same specs drive both help output and generated shell completions.
type flagSpec struct {
	name  string // long name, without the leading --
	kind  string // "string" or "bool"
	value string // metavariable shown for string flags
	desc  string
	files bool // request file completion for a string flag value
}

// commandSpec describes one subcommand.
type commandSpec struct {
	name    string
	summary string
	args    string // positional usage, e.g. "<task>"
	flags   []flagSpec
}

var commandSpecs = []commandSpec{
	{
		name:    "upload",
		summary: "Package a TaskZip directory or .zip and upload it as a new task",
		args:    "<task>",
		flags: []flagSpec{
			{name: "api-url", kind: "string", value: "URL", desc: "Backend base URL"},
			{name: "api-key", kind: "string", value: "KEY", desc: "Admin API key"},
			{name: "override-id", kind: "string", value: "ID", desc: "Store the task under this ID instead of the archive ID"},
			{name: "json", kind: "bool", desc: "Print the result as JSON"},
		},
	},
	{
		name:    "export",
		summary: "Download a task as a TaskZip .zip archive",
		args:    "<task-id>",
		flags: []flagSpec{
			{name: "api-url", kind: "string", value: "URL", desc: "Backend base URL"},
			{name: "api-key", kind: "string", value: "KEY", desc: "Admin API key"},
			{name: "out", kind: "string", value: "PATH", desc: "Output path, or - for stdout", files: true},
			{name: "force", kind: "bool", desc: "Overwrite an existing output file"},
		},
	},
	{
		name:    "completions",
		summary: "Print a shell completion script (fish, bash, zsh)",
		args:    "<shell>",
	},
	{
		name:    "version",
		summary: "Print the prglv version",
	},
	{
		name:    "help",
		summary: "Show this help",
	},
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "prglv:", err)
		if errors.Is(err, errUsage) {
			fmt.Fprintln(os.Stderr, "run 'prglv help' for usage")
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage(os.Stdout)
		return nil
	}
	switch args[0] {
	case "upload":
		return runUpload(args[1:])
	case "export":
		return runExport(args[1:])
	case "completions":
		return runCompletions(args[1:])
	case "version", "--version", "-v":
		fmt.Println("prglv", version)
		return nil
	case "help", "--help", "-h":
		printUsage(os.Stdout)
		return nil
	default:
		return fmt.Errorf("%w: unknown command %q", errUsage, args[0])
	}
}

// spec returns the commandSpec named name.
func spec(name string) commandSpec {
	for _, c := range commandSpecs {
		if c.name == name {
			return c
		}
	}
	panic("unknown command spec " + name)
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, "prglv %s - programme.lv administration CLI\n\n", version)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  prglv <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	for _, c := range commandSpecs {
		fmt.Fprintf(w, "  %-12s %s\n", c.name, c.summary)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Environment:")
	fmt.Fprintf(w, "  PRGLV_API_URL   Backend base URL (default %s)\n", defaultAPIURL)
	fmt.Fprintln(w, "  PRGLV_API_KEY   Admin API key (fallback ADMIN_API_KEY)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Run 'prglv <command> --help' for command flags.")
}

func printCommandHelp(w io.Writer, c commandSpec) {
	usage := c.name
	if c.args != "" {
		usage += " " + c.args
	}
	fmt.Fprintf(w, "Usage: prglv %s [flags]\n\n%s\n", usage, c.summary)
	if len(c.flags) == 0 {
		return
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	for _, f := range c.flags {
		left := "--" + f.name
		if f.kind == "string" {
			left += " " + f.value
		}
		fmt.Fprintf(w, "  %-22s %s\n", left, f.desc)
	}
}

// parsedFlags holds the flag values of one command invocation.
type parsedFlags struct {
	strs  map[string]string
	bools map[string]bool
}

// parseFlags parses args against c.flags.
// It returns flag.ErrHelp unwrapped when the user asked for help.
func parseFlags(c commandSpec, args []string) (*parsedFlags, []string, error) {
	set := flag.NewFlagSet("prglv "+c.name, flag.ContinueOnError)
	set.SetOutput(io.Discard)

	p := &parsedFlags{strs: map[string]string{}, bools: map[string]bool{}}
	strPtrs := make(map[string]*string, len(c.flags))
	boolPtrs := make(map[string]*bool, len(c.flags))
	for _, f := range c.flags {
		if f.kind == "bool" {
			boolPtrs[f.name] = set.Bool(f.name, false, f.desc)
		} else {
			strPtrs[f.name] = set.String(f.name, "", f.desc)
		}
	}

	flagArgs, positional := splitArgs(c, args)
	if err := set.Parse(flagArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil, nil, flag.ErrHelp
		}
		return nil, nil, fmt.Errorf("%w: %v", errUsage, err)
	}
	for name, ptr := range strPtrs {
		p.strs[name] = *ptr
	}
	for name, ptr := range boolPtrs {
		p.bools[name] = *ptr
	}
	return p, positional, nil
}

// splitArgs separates flags from positional arguments so that flags may
// appear before or after them. The stdlib flag package stops parsing at the
// first positional argument, which would otherwise swallow trailing flags.
// A bare "--" ends flag parsing; everything after it is positional.
func splitArgs(c commandSpec, args []string) (flags, positional []string) {
	takesValue := make(map[string]bool, len(c.flags))
	for _, f := range c.flags {
		if f.kind != "bool" {
			takesValue["--"+f.name] = true
		}
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return flags, append(positional, args[i+1:]...)
		}
		if len(arg) > 1 && arg[0] == '-' {
			flags = append(flags, arg)
			if takesValue[arg] && !strings.Contains(arg, "=") && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		positional = append(positional, arg)
	}
	return flags, positional
}
