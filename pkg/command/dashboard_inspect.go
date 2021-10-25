package command

import (
	"context"
	"flag"
	"fmt"

	"github.com/peterbourgon/ff/v2/ffcli"
)

// DashboardInspectConfig has the config for the dashboardInspect command and a reference to the root command config
type DashboardInspectConfig struct {
	*DashboardConfig

	UID string
}

// DashboardInspectCmd wraps the dashboardInspect config and a ffcli.Command
type DashboardInspectCmd struct {
	Conf *DashboardInspectConfig

	*ffcli.Command
}

// NewDashboardInspectCmd creates a new DashboardInspectCmd
func NewDashboardInspectCmd(dashConf *DashboardConfig) *DashboardInspectCmd {
	conf := DashboardInspectConfig{
		DashboardConfig: dashConf,
	}
	cmd := DashboardInspectCmd{
		Conf: &conf,
	}
	fs := flag.NewFlagSet("grafctl dashboard ls", flag.ExitOnError)
	cmd.RegisterFlags(fs)

	cmd.Command = &ffcli.Command{
		Name:        "inspect",
		ShortUsage:  "grafctl dash inspect",
		ShortHelp:   "Inspect grafana dashboard",
		FlagSet:     fs,
		Exec:        cmd.Exec,
		Subcommands: []*ffcli.Command{},
	}
	return &cmd
}

// RegisterFlags registers a set of flags for the dashboardInspect command
func (c *DashboardInspectCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Conf.UID, "uid", "", "dashboard UID")
}

// Exec executes the dashboard ls command
func (c *DashboardInspectCmd) Exec(ctx context.Context, args []string) error {
	dashboard, err := c.Conf.Client().GetDashboardByUID(ctx, c.Conf.UID)
	if err != nil {
		return err
	}

	dashBy, err := dashboard.Dashboard.EncodePretty()
	if err != nil {
		return err
	}
	fmt.Printf("%s\n", string(dashBy))

	return nil
}
