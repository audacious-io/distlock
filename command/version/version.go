package version

import (
	"github.com/mitchellh/cli"

	"lockerd/version"
)

func NewFactory(ui cli.Ui) cli.CommandFactory {
	return func() (cli.Command, error) {
		return &cmd{ui: ui}, nil
	}
}

type cmd struct {
	ui cli.Ui
}

func (c *cmd) Run(_ []string) int {
	c.ui.Output("lockerd " + version.HumanVersion())
	return 0
}

func (c *cmd) Synopsis() string {
	return "Display lockerd version"
}

func (c *cmd) Help() string {
	return ""
}
