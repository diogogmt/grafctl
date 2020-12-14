package command

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/grafana-tools/sdk"
	"github.com/peterbourgon/ff/v2/ffcli"
)

// DashboardBackupConfig has the config for the dashboardBackup command and a reference to the root command config
type DashboardBackupConfig struct {
	*DashboardConfig

	Provider string
	Out      string
}

// DashboardBackupCmd wraps the dashboardBackup config and a ffcli.Command
type DashboardBackupCmd struct {
	Conf *DashboardBackupConfig

	*ffcli.Command
}

type DashboardDump struct {
	Dashboard sdk.Board           `json:"dashboard"`
	Meta      sdk.BoardProperties `json:"meta"`
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
	fs.StringVar(&c.Conf.Provider, "provider", "local", "object storage provider, eg; local/gcs")
	fs.StringVar(&c.Conf.Out, "out", "", "location where to store the backup; either the path to a local dir or the remote bucket")
}

// Exec executes the dashboardBackup command
func (c *DashboardBackupCmd) Exec(ctx context.Context, args []string) error {
	var gcsBucket *storage.BucketHandle
	switch c.Conf.Provider {
	case "gcs":
		if c.Conf.Out == "" {
			return fmt.Errorf("missing bucket location")
		}
		gcsClient, err := storage.NewClient(ctx)
		if err != nil {
			return err
		}
		gcsBucket = gcsClient.Bucket(c.Conf.Out)
		if _, err := gcsBucket.Attrs(context.Background()); err != nil {
			return fmt.Errorf("error getting bucket %q attributes: %s", c.Conf.Out, err)
		}

	case "local":
		// nop
	default:
		return fmt.Errorf("provider %q not supported", c.Conf.Provider)
	}

	foundBoards, err := c.Conf.Client().SearchDashboards(ctx, "", false)
	if err != nil {
		return err
	}

	var b bytes.Buffer
	tarWriter := tar.NewWriter(&b)
	defer tarWriter.Close()
	for _, foundBoard := range foundBoards {
		board, boardMeta, err := c.Conf.Client().GetDashboardByUID(ctx, foundBoard.UID)
		if err != nil {
			return err
		}

		boardBy, err := json.Marshal(&DashboardDump{
			Dashboard: board,
			Meta:      boardMeta,
		})
		if err != nil {
			return err
		}
		header := &tar.Header{
			Name:    fmt.Sprintf("%s-%s.json", board.UID, strings.ReplaceAll(board.Title, " ", "_")),
			Mode:    int64(0644),
			ModTime: time.Now(),
			Size:    int64(len(boardBy)),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		if _, err := tarWriter.Write(boardBy); err != nil {
			return err
		}
	}

	u, err := url.Parse(c.Conf.APIURL)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	backupName := fmt.Sprintf("%s-%s-%d.tar", strings.ReplaceAll(u.Host, ".", "_"), now.Format("2006-01-02"), now.UnixNano())

	switch c.Conf.Provider {
	case "gcs":
		ctx, cancel := context.WithTimeout(ctx, time.Second*15)
		defer cancel()
		objectWriter := gcsBucket.Object(backupName).NewWriter(ctx)
		if _, err = io.Copy(objectWriter, bytes.NewReader(b.Bytes())); err != nil {
			return err
		}
		if err := objectWriter.Close(); err != nil {
			return err
		}
		fmt.Printf("gs://%s/%s\n", c.Conf.Out, backupName)
	case "local":
		p := filepath.Join(c.Conf.Out, backupName)
		if err := ioutil.WriteFile(p, b.Bytes(), 0644); err != nil {
			return err
		}
		fmt.Printf("%s\n", p)
	default:
		return fmt.Errorf("provider %q not supported", c.Conf.Provider)
	}

	return nil
}
