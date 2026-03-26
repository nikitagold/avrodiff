package cmd

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/nikitagold/avrodiff/diff"
	"github.com/nikitagold/avrodiff/model"
	"github.com/nikitagold/avrodiff/output"
)

func Execute() error {
	fs := flag.NewFlagSet("avrodiff", flag.ContinueOnError)
	fs.Usage = func() {
		_, _ = fmt.Fprintln(os.Stderr, "usage: avrodiff --base <file> --head <file> [flags]")
		fs.PrintDefaults()
	}

	basePath := fs.String("base", "", "path to base schema (file or git ref, e.g. main:avro/user.avsc)")
	headPath := fs.String("head", "", "path to head schema (file or git ref)")
	format := fs.String("format", "text", "output format: text or json")
	compatMode := fs.String("compat-mode", "full", "compatibility mode: backward, forward, full, none")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}
	if *basePath == "" || *headPath == "" {
		fs.Usage()
		return errors.New("--base and --head are required")
	}
	if *format != "text" && *format != "json" {
		return fmt.Errorf("unknown format %q: must be text or json", *format)
	}

	mode := parseCompatMode(*compatMode)

	base, err := readSchema(*basePath)
	if err != nil {
		return fmt.Errorf("parse base: %w", err)
	}
	head, err := readSchema(*headPath)
	if err != nil {
		return fmt.Errorf("parse head: %w", err)
	}

	result := diff.DiffSchemas(base, head, mode)

	if *format == "json" {
		if err := output.PrintJSON(os.Stdout, *basePath, result); err != nil {
			return fmt.Errorf("json output: %w", err)
		}
	} else {
		output.PrintText(os.Stdout, *basePath, result)
	}

	switch result.Level {
	case model.LevelMajor:
		os.Exit(1)
	case model.LevelMinor:
		os.Exit(2)
	case model.LevelPatch:
		os.Exit(3)
	}
	return nil
}

func parseCompatMode(s string) model.CompatMode {
	switch s {
	case "backward":
		return model.ModeBackward
	case "forward":
		return model.ModeForward
	case "none":
		return model.ModeNone
	default:
		return model.ModeFull
	}
}
