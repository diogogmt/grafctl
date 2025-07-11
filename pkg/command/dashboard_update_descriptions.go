package command

import (
	"context"
	"flag"
	"log"

	"github.com/peterbourgon/ff/v2/ffcli"
)

// DashboardUpdateDescriptionsConfig has the config for the dashboardUpdateDescriptions command
type DashboardUpdateDescriptionsConfig struct {
	*DashboardConfig

	UID       string
	Overwrite bool
	DryRun    bool
}

// DashboardUpdateDescriptionsCmd wraps the dashboardUpdateDescriptions config and a ffcli.Command
type DashboardUpdateDescriptionsCmd struct {
	Conf *DashboardUpdateDescriptionsConfig

	*ffcli.Command
}

// NewDashboardUpdateDescriptionsCmd creates a new DashboardUpdateDescriptionsCmd
func NewDashboardUpdateDescriptionsCmd(dashConf *DashboardConfig) *DashboardUpdateDescriptionsCmd {
	conf := DashboardUpdateDescriptionsConfig{
		DashboardConfig: dashConf,
	}
	cmd := DashboardUpdateDescriptionsCmd{
		Conf: &conf,
	}
	fs := flag.NewFlagSet("grafctl dashboard update-descriptions", flag.ExitOnError)
	cmd.RegisterFlags(fs)

	cmd.Command = &ffcli.Command{
		Name:        "update-descriptions",
		ShortUsage:  "grafctl dash update-descriptions",
		ShortHelp:   "update panel descriptions with proper query paths",
		FlagSet:     fs,
		Exec:        cmd.Exec,
		Subcommands: []*ffcli.Command{},
	}
	return &cmd
}

// RegisterFlags registers a set of flags for the dashboardUpdateDescriptions command
func (c *DashboardUpdateDescriptionsCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Conf.UID, "uid", "", "dashboard UID")
	fs.BoolVar(&c.Conf.Overwrite, "overwrite", false, "update all panels (not just invalid ones)")
	fs.BoolVar(&c.Conf.DryRun, "dry-run", false, "preview changes without updating dashboard")
}

// Exec executes the dashboard update-descriptions command
func (c *DashboardUpdateDescriptionsCmd) Exec(ctx context.Context, args []string) error {
	if c.Conf.UID == "" {
		log.Printf("missing -uid")
		c.FlagSet.Usage()
		return nil
	}

	if err := c.Conf.Client().UpdateDashboardDescriptions(ctx, c.Conf.UID, c.Conf.Overwrite, c.Conf.DryRun); err != nil {
		return err
	}

	return nil
}
