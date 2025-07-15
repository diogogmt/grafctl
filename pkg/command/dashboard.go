package command

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v2/ffcli"
)

// DashboardConfig has the config for the dashboard command and a reference to the root command config
type DashboardConfig struct {
	*RootConfig
}

// DashboardCmd wraps the dashboard config and a ffcli.Command
type DashboardCmd struct {
	Conf *DashboardConfig

	*ffcli.Command
}

// NewDashboardCmd creates a new DashboardCmd
func NewDashboardCmd(rootConf *RootConfig) *DashboardCmd {
	conf := DashboardConfig{
		RootConfig: rootConf,
	}
	cmd := DashboardCmd{
		Conf: &conf,
	}
	fs := flag.NewFlagSet("grafctl dashboard", flag.ExitOnError)
	cmd.RegisterFlags(fs)

	cmd.Command = &ffcli.Command{
		Name:       "dash",
		ShortUsage: "grafctl dash",
		ShortHelp:  "Manage grafana dashboards",
		FlagSet:    fs,
		Exec:       cmd.Exec,
		Subcommands: []*ffcli.Command{
			NewDashboardLsCmd(&conf).Command,
			NewDashboardInspectCmd(&conf).Command,
			NewDashboardSyncCmd(&conf).Command,
			NewDashboardUpdatePanelsDescriptionsCmd(&conf).Command,
		},
	}
	return &cmd
}

// RegisterFlags registers a set of flags for the dashboard command
func (c *DashboardCmd) RegisterFlags(fs *flag.FlagSet) {
}

// Exec executes the dashboard command
func (c *DashboardCmd) Exec(ctx context.Context, args []string) error {
	// TODO(dm): list all dashboards by default
	c.FlagSet.Usage()
	return nil
}
