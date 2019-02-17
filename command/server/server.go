package server

import (
	"flag"
	"io/ioutil"
	"net/http"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/mitchellh/cli"

	"lockerd/httpserver"
	"lockerd/locking"
	"lockerd/version"
)

func NewFactory(ui cli.Ui) cli.CommandFactory {
	return func() (cli.Command, error) {
		flags := flag.NewFlagSet("", flag.ContinueOnError)
		flags.SetOutput(ioutil.Discard)
		addr := flags.String("address", ":12000", "")

		return &cmd{
			ui:    ui,
			addr:  addr,
			flags: flags,
		}, nil
	}
}

type cmd struct {
	ui    cli.Ui
	addr  *string
	flags *flag.FlagSet
}

func (c *cmd) Run(args []string) int {
	// Parse arguments.
	if err := c.flags.Parse(args); err != nil {
		c.ui.Error(err.Error())
		c.ui.Error("")
		c.ui.Error(c.Help())
		return 2
	}

	// Set up the lock manager.
	manager := locking.NewManager(locking.Config{})

	// Set up the server.
	handler := httpserver.NewHandler(manager)
	server := &http.Server{
		Addr:    *c.addr,
		Handler: handler,
	}

	c.ui.Output("Starting lockerd " + version.HumanVersion() + " HTTP API server on " + *c.addr)

	if err := gracehttp.Serve(server); err != nil {
		c.ui.Error("Error starting HTTP server: " + err.Error())
	}

	return 0
}

func (c *cmd) Synopsis() string {
	return "Start the lockerd server"
}

func (c *cmd) Help() string {
	return `Usage: lockerd server [options]

  Starts the lockerd server.

Options:

  --address=:12000  Listening address.`
}
