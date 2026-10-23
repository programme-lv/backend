package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
)

func runManifest(args []string) error {
	c := spec("manifest")
	flags, positional, err := parseFlags(c, args)
	if errors.Is(err, flag.ErrHelp) {
		printCommandHelp(os.Stdout, c)
		return nil
	}
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return fmt.Errorf("%w: manifest needs exactly one <task-id>", errUsage)
	}
	apiKey := resolveAPIKey(flags.strs["api-key"])
	if apiKey == "" {
		return errors.New("missing admin API key; set PRGLV_API_KEY or ADMIN_API_KEY, or pass --api-key")
	}
	manifest, err := getTaskManifest(
		context.Background(), resolveAPIURL(flags.strs["api-url"]), apiKey, positional[0],
	)
	if err != nil {
		return err
	}
	if flags.bools["json"] {
		return json.NewEncoder(os.Stdout).Encode(manifest)
	}
	fmt.Printf("task %s: revision %d, tests version %d, %d tests\n",
		manifest.TaskID, manifest.Revision, manifest.TestsVersion, manifest.TestsCount)
	fmt.Printf("content etag: %s\n", manifest.ContentETag)
	for _, section := range []string{
		"statement", "examples", "solutions", "tests", "readme", "checker", "interactor",
	} {
		if digest := manifest.Sections[section]; digest != "" {
			fmt.Printf("%s: %s\n", section, digest)
		}
	}
	return nil
}
