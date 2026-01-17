package main

var Version = "dev" // Set by the build system to the release version

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/UnitVectorY-Labs/ghprmerge/internal/config"
	"github.com/UnitVectorY-Labs/ghprmerge/internal/github"
	"github.com/UnitVectorY-Labs/ghprmerge/internal/merger"
	"github.com/UnitVectorY-Labs/ghprmerge/internal/output"
)

func main() {
	if err := run(); err != nil {
		// Don't print error for help request
		if errors.Is(err, config.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Parse configuration
	cfg, err := config.ParseFlags(os.Args[1:])
	if err != nil {
		return err
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return err
	}

	// Create GitHub client
	client := github.NewRealClient(cfg.Token)

	// Create merger
	m := merger.New(client, cfg)

	// Run merger
	ctx := context.Background()
	result, err := m.Run(ctx)
	if err != nil {
		return err
	}

	// Output results
	writer := output.NewWriter(os.Stdout, cfg.JSON)
	return writer.WriteResult(result)

