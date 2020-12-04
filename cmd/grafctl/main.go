package main

import (
	"context"
	"fmt"
	"os"

	"github.com/diogogmt/grafctl/pkg/command"
)

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "run: %s", err)
	}
}

func run(args []string) error {
	rootCmd := command.NewRootCmd()

	if err := rootCmd.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error during Parse: %v\n", err)
		os.Exit(1)
	}

	if err := rootCmd.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	return nil
}
