package command

import (
	"context"
	"flag"
	"log"

	"github.com/peterbourgon/ff/v2/ffcli"
)

// DashboardUpdatePanelsDescriptionsConfig has the config for the dashboardUpdatePanelsDescriptions command
type DashboardUpdatePanelsDescriptionsConfig struct {
	*DashboardConfig

	UID       string
	Overwrite bool
	DryRun    bool
}

// DashboardUpdatePanelsDescriptionsCmd wraps the dashboardUpdatePanelsDescriptions config and a ffcli.Command
type DashboardUpdatePanelsDescriptionsCmd struct {
	Conf *DashboardUpdatePanelsDescriptionsConfig

	*ffcli.Command
}

// NewDashboardUpdatePanelsDescriptionsCmd creates a new DashboardUpdatePanelsDescriptionsCmd
func NewDashboardUpdatePanelsDescriptionsCmd(dashConf *DashboardConfig) *DashboardUpdatePanelsDescriptionsCmd {
	conf := DashboardUpdatePanelsDescriptionsConfig{
		DashboardConfig: dashConf,
	}
	cmd := DashboardUpdatePanelsDescriptionsCmd{
		Conf: &conf,
	}
	fs := flag.NewFlagSet("grafctl dashboard update-panels-descriptions", flag.ExitOnError)
	cmd.RegisterFlags(fs)

	cmd.Command = &ffcli.Command{
		Name:        "update-panels-descriptions",
		ShortUsage:  "grafctl dash update-panels-descriptions",
		ShortHelp:   "update panel descriptions with proper query paths",
		FlagSet:     fs,
		Exec:        cmd.Exec,
		Subcommands: []*ffcli.Command{},
	}
	return &cmd
}

// RegisterFlags registers a set of flags for the dashboardUpdatePanelsDescriptions command
func (c *DashboardUpdatePanelsDescriptionsCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Conf.UID, "uid", "", "dashboard UID")
	fs.BoolVar(&c.Conf.Overwrite, "overwrite", false, "update all panels (not just invalid ones)")
	fs.BoolVar(&c.Conf.DryRun, "dry-run", false, "preview changes without updating dashboard")
}

// Exec executes the dashboard update-descriptions command
func (c *DashboardUpdatePanelsDescriptionsCmd) Exec(ctx context.Context, args []string) error {
	if c.Conf.UID == "" {
		log.Printf("missing -uid")
		c.FlagSet.Usage()
		return nil
	}

	if err := c.Conf.Client().UpdateDashboardPanelsDescription(ctx, c.Conf.UID, c.Conf.Overwrite, c.Conf.DryRun); err != nil {
		return err
	}

	return nil
}
