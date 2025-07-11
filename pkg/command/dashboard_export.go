package command

import (
	"context"
	"flag"
	"log"

	"github.com/peterbourgon/ff/v2/ffcli"
)

// DashboardExportConfig has the config for the dashboardExport command and a reference to the root command config
type DashboardExportConfig struct {
	*DashboardConfig

	UID        string
	QueriesDir string
	Overwrite  bool
}

// DashboardExportCmd wraps the dashboardExport config and a ffcli.Command
type DashboardExportCmd struct {
	Conf *DashboardExportConfig

	*ffcli.Command
}

// NewDashboardExportCmd creates a new DashboardExportCmd
func NewDashboardExportCmd(dashConf *DashboardConfig) *DashboardExportCmd {
	conf := DashboardExportConfig{
		DashboardConfig: dashConf,
	}
	cmd := DashboardExportCmd{
		Conf: &conf,
	}
	fs := flag.NewFlagSet("grafctl dashboard export", flag.ExitOnError)
	cmd.RegisterFlags(fs)

	cmd.Command = &ffcli.Command{
		Name:        "export",
		ShortUsage:  "grafctl dash export",
		ShortHelp:   "export panel queries from grafana dashboard to filesystem",
		FlagSet:     fs,
		Exec:        cmd.Exec,
		Subcommands: []*ffcli.Command{},
	}
	return &cmd
}

// RegisterFlags registers a set of flags for the dashboardExport command
func (c *DashboardExportCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Conf.UID, "uid", "", "dashboard UID")
	fs.StringVar(&c.Conf.QueriesDir, "queries", "", "base directory to save queries")
	fs.BoolVar(&c.Conf.Overwrite, "overwrite", false, "overwrite existing query files")
}

// Exec executes the dashboard export command
func (c *DashboardExportCmd) Exec(ctx context.Context, args []string) error {
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

	if err := c.Conf.Client().ExportDashboard(ctx, c.Conf.UID, c.Conf.QueriesDir, c.Conf.Overwrite); err != nil {
		return err
	}

	return nil
}
