package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
)

func runExport(args []string) error {
	c := spec("export")
	flags, positional, err := parseFlags(c, args)
	if errors.Is(err, flag.ErrHelp) {
		printCommandHelp(os.Stdout, c)
		return nil
	}
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return fmt.Errorf("%w: export needs exactly one <task-id>", errUsage)
	}

	apiKey := resolveAPIKey(flags.strs["api-key"])
	if apiKey == "" {
		return errors.New("missing admin API key; set PRGLV_API_KEY or ADMIN_API_KEY, or pass --api-key")
	}

	taskID := positional[0]
	apiURL := resolveAPIURL(flags.strs["api-url"])
	data, err := exportTask(context.Background(), apiURL, apiKey, taskID)
	if err != nil {
		return err
	}

	out := flags.strs["out"]
	if out == "" {
		out = taskID + ".zip"
	}
	if out == "-" {
		_, err := os.Stdout.Write(data)
		return err
	}

	if !flags.bools["force"] {
		if _, statErr := os.Stat(out); statErr == nil {
			return fmt.Errorf("%s already exists; pass --force to overwrite", out)
		}
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	fmt.Printf("ok: wrote %s (%d bytes)\n", out, len(data))
	return nil
}
