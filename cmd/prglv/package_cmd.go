package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
)

func runPackage(args []string) error {
	c := spec("package")
	flags, positional, err := parseFlags(c, args)
	if errors.Is(err, flag.ErrHelp) {
		printCommandHelp(os.Stdout, c)
		return nil
	}
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return fmt.Errorf("%w: package needs exactly one <task> path", errUsage)
	}

	zipBytes, taskID, err := readTaskPackage(positional[0])
	if err != nil {
		return err
	}
	out := flags.strs["out"]
	if out == "" {
		out = taskID + ".zip"
	}
	if out == "-" {
		_, err := os.Stdout.Write(zipBytes)
		return err
	}
	if !flags.bools["force"] {
		if _, err := os.Stat(out); err == nil {
			return fmt.Errorf("%s already exists; pass --force to overwrite", out)
		}
	}
	if err := os.WriteFile(out, zipBytes, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}
	if flags.bools["json"] {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{
			"task_id": taskID, "path": out, "bytes": len(zipBytes),
		})
	}
	fmt.Printf("ok: packaged %s (%d bytes) to %s\n", taskID, len(zipBytes), out)
	return nil
}
