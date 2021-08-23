package command

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v2/ffcli"
)

// BackupConfig has the config for the dashboardBackup command and a reference to the root command config
type BackupConfig struct {
	*RootConfig

	Provider string
	Out      string
}

// BackupCmd wraps the dashboardBackup config and a ffcli.Command
type BackupCmd struct {
	Conf *BackupConfig

	*ffcli.Command
}

// NewBackupCmd creates a new BackupCmd
func NewBackupCmd(rootConf *RootConfig) *BackupCmd {
	conf := BackupConfig{
		RootConfig: rootConf,
	}
	cmd := BackupCmd{
		Conf: &conf,
	}
	fs := flag.NewFlagSet("grafctl backup", flag.ExitOnError)
	cmd.RegisterFlags(fs)

	cmd.Command = &ffcli.Command{
		Name:        "backup",
		ShortUsage:  "grafctl backup",
		ShortHelp:   "Backup grafana dashboards and datasources",
		FlagSet:     fs,
		Exec:        cmd.Exec,
		Subcommands: []*ffcli.Command{},
	}
	return &cmd
}

// RegisterFlags registers a set of flags for the dashboardBackup command
func (c *BackupCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Conf.Provider, "provider", "local", "object storage provider, eg; local/gcs")
	fs.StringVar(&c.Conf.Out, "out", "", "location where to store the backup; either the path to a local dir or the remote bucket")
}

// Exec executes the dashboardBackup command
func (c *BackupCmd) Exec(ctx context.Context, args []string) error {
	if err := c.Conf.Client().BackupGrafana(ctx, BackupProvider(c.Conf.Provider), c.Conf.Out); err != nil {
		return err
	}
	return nil
}
