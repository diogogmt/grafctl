package command

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v2/ffcli"
)

// RootConfig has the config for the root command
type RootConfig struct {
	APIURL  string
	APIKey  string
	Verbose bool
}

func (c *RootConfig) Client() *Client {
	return NewClient(c.APIURL, c.APIKey)
}

// RootCmd wraps the  config and a ffcli.Command
type RootCmd struct {
	Conf *RootConfig

	*ffcli.Command
}

// RootConfigOption defines the signature for functional options to be applied to the root command
type RootConfigOption = func(c *RootConfig)

// NewRootCmd creates a new RootCmd
func NewRootCmd(opts ...RootConfigOption) *RootCmd {
	fs := flag.NewFlagSet("grafctl", flag.ExitOnError)

	conf := RootConfig{}
	for _, opt := range opts {
		opt(&conf)
	}

	cmd := RootCmd{
		Conf: &conf,
	}
	cmd.Command = &ffcli.Command{
		Name:       "grafctl",
		ShortUsage: "grafctl [flags] <subcommand>",
		ShortHelp:  "Grafana control",
		FlagSet:    fs,
		Exec:       cmd.Exec,
		Subcommands: []*ffcli.Command{
			NewDashboardCmd(&conf).Command,
			NewBackupCmd(&conf).Command,
			NewImportCmd(&conf).Command,
		},
	}

	cmd.RegisterFlags(fs)

	return &cmd
}

// RegisterFlags registers a set of flags for the root command
func (c *RootCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Conf.APIURL, "url", "", "grafana server API URL")
	fs.StringVar(&c.Conf.APIKey, "key", "", "grafana server API key")
	fs.BoolVar(&c.Conf.Verbose, "verbose", false, "log verbose output")
}

// Exec executes the root command
func (c *RootCmd) Exec(ctx context.Context, args []string) error {
	c.FlagSet.Usage()
	return nil
}
