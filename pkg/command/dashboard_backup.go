package command

import (
	"context"
	"encoding/json"
	"flag"
	"log"

	"github.com/peterbourgon/ff/v2/ffcli"
)

// DashboardBackupConfig has the config for the dashboardBackup command and a reference to the root command config
type DashboardBackupConfig struct {
	*DashboardConfig

	UID    string
	All    bool
	Format bool
}

// DashboardBackupCmd wraps the dashboardBackup config and a ffcli.Command
type DashboardBackupCmd struct {
	Conf *DashboardBackupConfig

	*ffcli.Command
}

// NewDashboardBackupCmd creates a new DashboardBackupCmd
func NewDashboardBackupCmd(dashConf *DashboardConfig) *DashboardBackupCmd {
	conf := DashboardBackupConfig{
		DashboardConfig: dashConf,
	}
	cmd := DashboardBackupCmd{
		Conf: &conf,
	}
	fs := flag.NewFlagSet("grafctl dashboard backup", flag.ExitOnError)
	cmd.RegisterFlags(fs)

	cmd.Command = &ffcli.Command{
		Name:        "backup",
		ShortUsage:  "grafc dash backup",
		ShortHelp:   "Backup grafana dashboards",
		FlagSet:     fs,
		Exec:        cmd.Exec,
		Subcommands: []*ffcli.Command{},
	}
	return &cmd
}

// RegisterFlags registers a set of flags for the dashboardBackup command
func (c *DashboardBackupCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Conf.UID, "uid", "", "dashboard UID")
	fs.BoolVar(&c.Conf.All, "bool", false, "all dashboards")
}

// Exec executes the dashboardBackup command
// TODO(dm): support backing up all dashboards to an object sorage, eg GCS
func (c *DashboardBackupCmd) Exec(ctx context.Context, args []string) error {
	if c.Conf.All {
		// TODO(dm): backup all dashboards
		return nil
	}

	if c.Conf.UID == "" {
		log.Printf("missing -uid")
		c.FlagSet.Usage()
		return nil
	}

	board, _, err := c.Conf.Client().GetDashboardByUID(ctx, c.Conf.UID)
	if err != nil {
		return err
	}
	boardBy, err := json.Marshal(board)
	if err != nil {
		return err
	}

	if c.Conf.Format {
		// TODO(dm): format json
	} else {
		log.Printf("%+v", string(boardBy))
	}
	return nil
}
