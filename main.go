package main

import (
	"fmt"
	"os"

	"github.com/mitchellh/cli"

	"lockerd/command/server"
	"lockerd/command/version"
)

func main() {
	os.Exit(realMain())
}

func realMain() int {
	// Expand version argument as a command override.
	args := os.Args[1:]
	for _, arg := range args {
		if arg == "--" {
			break
		}

		if arg == "-v" || arg == "--version" {
			args = []string{"version"}
			break
		}
	}

	// Set up CLI.
	ui := &cli.BasicUi{Writer: os.Stdout, ErrorWriter: os.Stderr}
	cli := &cli.CLI{
		Args: args,
		//Commands:     cmds,
		Commands: map[string]cli.CommandFactory{
			"server":  server.NewFactory(ui),
			"version": version.NewFactory(ui),
		},
		Autocomplete: true,
		Name:         "lockerd",
	}

	// Run the CLI.
	exitCode, err := cli.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing CLI: %s\n", err.Error())
		return 1
	}

	return exitCode
}
