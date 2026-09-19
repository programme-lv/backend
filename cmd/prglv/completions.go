package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

func runCompletions(args []string) error {
	c := spec("completions")
	_, positional, err := parseFlags(c, args)
	if errors.Is(err, flag.ErrHelp) {
		printCommandHelp(os.Stdout, c)
		return nil
	}
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return fmt.Errorf("%w: completions needs exactly one <shell>", errUsage)
	}

	switch positional[0] {
	case "fish":
		fmt.Print(fishCompletions())
	case "bash":
		fmt.Print(bashCompletions())
	case "zsh":
		fmt.Print(zshCompletions())
	default:
		return fmt.Errorf("%w: unsupported shell %q (use fish, bash, or zsh)", errUsage, positional[0])
	}
	return nil
}

// fishCompletions renders a fish completion script from commandSpecs.
func fishCompletions() string {
	var b strings.Builder
	b.WriteString("# prglv fish completions. Install with:\n")
	b.WriteString("#   prglv completions fish > ~/.config/fish/completions/prglv.fish\n\n")
	b.WriteString("complete -c prglv -f\n")
	for _, c := range commandSpecs {
		fmt.Fprintf(&b, "complete -c prglv -n __fish_use_subcommand -a %s -d %s\n",
			c.name, fishQuote(c.summary))
	}
	for _, c := range commandSpecs {
		cond := fishQuote("__fish_seen_subcommand_from " + c.name)
		switch c.name {
		case "completions":
			fmt.Fprintf(&b, "complete -c prglv -n %s -a 'fish bash zsh' -d 'Shell'\n", cond)
		case "upload":
			fmt.Fprintf(&b, "complete -c prglv -n %s -F -d 'TaskZip directory or .zip'\n", cond)
		case "export":
			fmt.Fprintf(&b, "complete -c prglv -n %s -d 'Task ID'\n", cond)
		}
		for _, f := range c.flags {
			if f.kind == "bool" {
				fmt.Fprintf(&b, "complete -c prglv -n %s -l %s -d %s\n",
					cond, f.name, fishQuote(f.desc))
				continue
			}
			files := ""
			if f.files {
				files = " -F"
			}
			fmt.Fprintf(&b, "complete -c prglv -n %s -l %s -r%s -d %s\n",
				cond, f.name, files, fishQuote(f.desc))
		}
	}
	return b.String()
}

// fishQuote wraps s in single quotes for fish.
func fishQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `\'`) + "'"
}

// bashCompletions renders a bash completion script from commandSpecs.
func bashCompletions() string {
	names := make([]string, len(commandSpecs))
	for i, c := range commandSpecs {
		names[i] = c.name
	}

	var b strings.Builder
	b.WriteString("# prglv bash completions. Install with:\n")
	b.WriteString("#   prglv completions bash > /etc/bash_completion.d/prglv\n\n")
	b.WriteString("_prglv() {\n")
	b.WriteString("    local cur cmd i\n")
	b.WriteString("    cur=\"${COMP_WORDS[COMP_CWORD]}\"\n")
	b.WriteString("    cmd=\"\"\n")
	b.WriteString("    for ((i=1; i<COMP_CWORD; i++)); do\n")
	b.WriteString("        case \"${COMP_WORDS[i]}\" in\n")
	b.WriteString("            -*) ;;\n")
	b.WriteString("            *) cmd=\"${COMP_WORDS[i]}\"; break ;;\n")
	b.WriteString("        esac\n")
	b.WriteString("    done\n")
	fmt.Fprintf(&b, "    if [[ -z \"$cmd\" ]]; then\n        COMPREPLY=( $(compgen -W '%s' -- \"$cur\") )\n        return\n    fi\n",
		strings.Join(names, " "))
	b.WriteString("    case \"$cmd\" in\n")
	for _, c := range commandSpecs {
		fmt.Fprintf(&b, "        %s)\n", c.name)
		switch {
		case c.name == "completions":
			b.WriteString("            COMPREPLY=( $(compgen -W 'fish bash zsh' -- \"$cur\") )\n")
		case len(c.flags) > 0:
			flags := make([]string, len(c.flags))
			for i, f := range c.flags {
				flags[i] = "--" + f.name
			}
			fmt.Fprintf(&b, "            COMPREPLY=( $(compgen -W '%s' -- \"$cur\") )\n",
				strings.Join(flags, " "))
			if c.name == "upload" {
				b.WriteString("            [[ \"$cur\" == -* ]] || COMPREPLY+=( $(compgen -f -- \"$cur\") )\n")
			}
		default:
			b.WriteString("            COMPREPLY=()\n")
		}
		b.WriteString("            ;;\n")
	}
	b.WriteString("        *)\n            COMPREPLY=()\n            ;;\n")
	b.WriteString("    esac\n")
	b.WriteString("}\n")
	b.WriteString("complete -F _prglv prglv\n")
	return b.String()
}

// zshCompletions renders a zsh completion script from commandSpecs.
func zshCompletions() string {
	var b strings.Builder
	b.WriteString("#compdef prglv\n")
	b.WriteString("# prglv zsh completions. Install with:\n")
	b.WriteString("#   prglv completions zsh > \"${fpath[1]}/_prglv\"\n\n")
	b.WriteString("_prglv() {\n")
	b.WriteString("    local -a commands\n")
	b.WriteString("    commands=(\n")
	for _, c := range commandSpecs {
		fmt.Fprintf(&b, "        '%s:%s'\n", c.name, zshQuote(c.summary))
	}
	b.WriteString("    )\n")
	b.WriteString("    if (( CURRENT == 2 )); then\n")
	b.WriteString("        _describe 'command' commands\n")
	b.WriteString("        return\n")
	b.WriteString("    fi\n")
	b.WriteString("    case ${words[2]} in\n")
	for _, c := range commandSpecs {
		fmt.Fprintf(&b, "        %s)\n", c.name)
		switch c.name {
		case "completions":
			b.WriteString("            _values 'shell' fish bash zsh\n")
		case "upload", "export":
			fmt.Fprintf(&b, "            _arguments %s\n", zshArguments(c))
		default:
			b.WriteString("            _arguments\n")
		}
		b.WriteString("            ;;\n")
	}
	b.WriteString("    esac\n")
	b.WriteString("}\n")
	b.WriteString("_prglv \"$@\"\n")
	return b.String()
}

// zshArguments renders the _arguments specs for a command with positionals.
func zshArguments(c commandSpec) string {
	parts := make([]string, 0, len(c.flags)+1)
	for _, f := range c.flags {
		switch {
		case f.kind == "bool":
			parts = append(parts, fmt.Sprintf("'--%s[%s]'", f.name, zshQuote(f.desc)))
		case f.files:
			parts = append(parts, fmt.Sprintf("'--%s[%s]:%s:_files'", f.name, zshQuote(f.desc), f.value))
		default:
			parts = append(parts, fmt.Sprintf("'--%s[%s]:%s:'", f.name, zshQuote(f.desc), f.value))
		}
	}
	if c.name == "upload" {
		parts = append(parts, "'*:task:_files'")
	} else {
		parts = append(parts, "'*:task id:'")
	}
	return strings.Join(parts, " ")
}

// zshQuote escapes a description for use inside a single-quoted zsh string.
func zshQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `'`, `'\''`)
	s = strings.ReplaceAll(s, `[`, `\[`)
	s = strings.ReplaceAll(s, `]`, `\]`)
	return s
}
