package command

import (
	"context"
	"flag"
	"log"

	"github.com/peterbourgon/ff/v2/ffcli"
)

// DashboardExportQueriesConfig has the config for the dashboardExportQueries command and a reference to the root command config
type DashboardExportQueriesConfig struct {
	*DashboardConfig

	UID        string
	QueriesDir string
	Overwrite  bool
}

// DashboardExportQueriesCmd wraps the dashboardExportQueries config and a ffcli.Command
type DashboardExportQueriesCmd struct {
	Conf *DashboardExportQueriesConfig

	*ffcli.Command
}

// NewDashboardExportQueriesCmd creates a new DashboardExportQueriesCmd
func NewDashboardExportQueriesCmd(dashConf *DashboardConfig) *DashboardExportQueriesCmd {
	conf := DashboardExportQueriesConfig{
		DashboardConfig: dashConf,
	}
	cmd := DashboardExportQueriesCmd{
		Conf: &conf,
	}
	fs := flag.NewFlagSet("grafctl dashboard export-queries", flag.ExitOnError)
	cmd.RegisterFlags(fs)

	cmd.Command = &ffcli.Command{
		Name:        "export-queries",
		ShortUsage:  "grafctl dash export-queries",
		ShortHelp:   "export panel queries from grafana dashboard to filesystem",
		FlagSet:     fs,
		Exec:        cmd.Exec,
		Subcommands: []*ffcli.Command{},
	}
	return &cmd
}

// RegisterFlags registers a set of flags for the dashboardExportQueries command
func (c *DashboardExportQueriesCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Conf.UID, "uid", "", "dashboard UID")
	fs.StringVar(&c.Conf.QueriesDir, "out", "", "base directory to save queries")
	fs.BoolVar(&c.Conf.Overwrite, "overwrite", false, "overwrite existing query files")
}

// Exec executes the dashboard export-queries command
func (c *DashboardExportQueriesCmd) Exec(ctx context.Context, args []string) error {
	if c.Conf.UID == "" {
		log.Printf("missing -uid")
		c.FlagSet.Usage()
		return nil
	}
	if c.Conf.QueriesDir == "" {
		log.Printf("missing -out, defaulting to ./queries")
		c.Conf.QueriesDir = "./queries"
	}

	if err := c.Conf.Client().ExportDashboardQueries(ctx, c.Conf.UID, c.Conf.QueriesDir, c.Conf.Overwrite); err != nil {
		return err
	}

	return nil
}
