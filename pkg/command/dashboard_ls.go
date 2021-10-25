package command

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/diogogmt/grafctl/pkg/grafsdk"
	"github.com/olekukonko/tablewriter"
	"github.com/peterbourgon/ff/v2/ffcli"
)

// DashboardLsConfig has the config for the dashboardLs command and a reference to the root command config
type DashboardLsConfig struct {
	*DashboardConfig

	UID string
}

// DashboardLsCmd wraps the dashboardLs config and a ffcli.Command
type DashboardLsCmd struct {
	Conf *DashboardLsConfig

	*ffcli.Command
}

// NewDashboardLsCmd creates a new DashboardLsCmd
func NewDashboardLsCmd(dashConf *DashboardConfig) *DashboardLsCmd {
	conf := DashboardLsConfig{
		DashboardConfig: dashConf,
	}
	cmd := DashboardLsCmd{
		Conf: &conf,
	}
	fs := flag.NewFlagSet("grafctl dashboard ls", flag.ExitOnError)
	cmd.RegisterFlags(fs)

	cmd.Command = &ffcli.Command{
		Name:        "ls",
		ShortUsage:  "grafctl dash ls",
		ShortHelp:   "List grafana dashboards",
		FlagSet:     fs,
		Exec:        cmd.Exec,
		Subcommands: []*ffcli.Command{},
	}
	return &cmd
}

// RegisterFlags registers a set of flags for the dashboardLs command
func (c *DashboardLsCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Conf.UID, "uid", "", "dashboard UID")
}

// Exec executes the dashboard ls command
func (c *DashboardLsCmd) Exec(ctx context.Context, args []string) error {
	dashboards, err := c.Conf.Client().Search(ctx, grafsdk.DashTypeSearchOption())
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"UID", "Folder", "Title", "URL"})

	for _, dashboard := range dashboards {
		table.Append([]string{dashboard.UID, dashboard.FolderTitle, dashboard.Title, fmt.Sprintf("%s/%s", c.Conf.APIURL, dashboard.URL)})
	}
	table.Render()

	return nil
}
