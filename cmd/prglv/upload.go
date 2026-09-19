package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
)

func runUpload(args []string) error {
	c := spec("upload")
	flags, positional, err := parseFlags(c, args)
	if errors.Is(err, flag.ErrHelp) {
		printCommandHelp(os.Stdout, c)
		return nil
	}
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return fmt.Errorf("%w: upload needs exactly one <task> path", errUsage)
	}

	apiKey := resolveAPIKey(flags.strs["api-key"])
	if apiKey == "" {
		return errors.New("missing admin API key; set PRGLV_API_KEY or ADMIN_API_KEY, or pass --api-key")
	}

	zipBytes, archiveID, err := readTaskPackage(positional[0])
	if err != nil {
		return err
	}

	apiURL := resolveAPIURL(flags.strs["api-url"])
	createdID, err := uploadTask(context.Background(), apiURL, apiKey, zipBytes, flags.strs["override-id"])
	if err != nil {
		return err
	}

	if flags.bools["json"] {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"task_id": createdID})
	}
	fmt.Printf("ok: uploaded %s (%d bytes) from %s to %s\n", createdID, len(zipBytes), archiveID, apiURL)
	return nil
}
