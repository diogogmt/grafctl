package command

import (
	"context"
	"flag"
	"log"

	"github.com/peterbourgon/ff/v2/ffcli"
)

// DashboardSyncConfig has the config for the dashboardSync command and a reference to the root command config
type DashboardSyncConfig struct {
	*DashboardConfig

	UID        string
	QueriesDir string
}

// DashboardSyncCmd wraps the dashboardSync config and a ffcli.Command
type DashboardSyncCmd struct {
	Conf *DashboardSyncConfig

	*ffcli.Command
}

// NewDashboardSyncCmd creates a new DashboardSyncCmd
func NewDashboardSyncCmd(dashConf *DashboardConfig) *DashboardSyncCmd {
	conf := DashboardSyncConfig{
		DashboardConfig: dashConf,
	}
	cmd := DashboardSyncCmd{
		Conf: &conf,
	}
	fs := flag.NewFlagSet("grafctl dashboard ls", flag.ExitOnError)
	cmd.RegisterFlags(fs)

	cmd.Command = &ffcli.Command{
		Name:        "sync",
		ShortUsage:  "grafctl dash sync",
		ShortHelp:   "sync grafana dashboards",
		FlagSet:     fs,
		Exec:        cmd.Exec,
		Subcommands: []*ffcli.Command{},
	}
	return &cmd
}

// RegisterFlags registers a set of flags for the dashboardSync command
func (c *DashboardSyncCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Conf.UID, "uid", "", "dashboard UID")
	fs.StringVar(&c.Conf.QueriesDir, "queries", "", "base directory to build queries catalog")
}

// Exec executes the dashboard sync command
func (c *DashboardSyncCmd) Exec(ctx context.Context, args []string) error {
	if c.Conf.UID == "" {
		log.Printf("missing -uid")
		c.FlagSet.Usage()
		return nil
	}
	if c.Conf.QueriesDir == "" {
		log.Printf("missing -queries")
		c.FlagSet.Usage()
		return nil
	}

	if err := c.Conf.Client().SyncDashboard(ctx, c.Conf.UID, c.Conf.QueriesDir); err != nil {
		return err
	}

	return nil
}
