package command

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/grafana-tools/sdk"
	"github.com/peterbourgon/ff/v2/ffcli"
)

// ImportConfig has the config for the dashboardImport command and a reference to the root command config
type ImportConfig struct {
	*RootConfig

	Src string
}

// ImportCmd wraps the dashboardImport config and a ffcli.Command
type ImportCmd struct {
	Conf *ImportConfig

	*ffcli.Command
}

// NewImportCmd creates a new ImportCmd
func NewImportCmd(rootConf *RootConfig) *ImportCmd {
	conf := ImportConfig{
		RootConfig: rootConf,
	}
	cmd := ImportCmd{
		Conf: &conf,
	}
	fs := flag.NewFlagSet("grafctl import", flag.ExitOnError)
	cmd.RegisterFlags(fs)

	cmd.Command = &ffcli.Command{
		Name:        "import",
		ShortUsage:  "grafc import",
		ShortHelp:   "Import grafana dashboards and datasources",
		FlagSet:     fs,
		Exec:        cmd.Exec,
		Subcommands: []*ffcli.Command{},
	}
	return &cmd
}

// RegisterFlags registers a set of flags for the dashboardImport command
func (c *ImportCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Conf.Src, "src", "", "location where to read the backup from; either the path to a local backup or the remote bucket URL, eg; gs://grafana-backup-bucket/monitoring-2020-12-20.json")
}

// Exec executes the dashboardImport command
func (c *ImportCmd) Exec(ctx context.Context, args []string) error {
	if c.Conf.Src == "" {
		return fmt.Errorf("missing -src")
	}

	var bucketName string
	var objectName string
	if strings.HasPrefix(c.Conf.Src, "gs://") {
		parts := strings.Split(c.Conf.Src, "gs://")
		if len(parts) != 2 {
			return fmt.Errorf("invalid gcs url")
		}
		parts = strings.Split(parts[1], "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid gcs url")
		}
		bucketName = parts[0]
		objectName = parts[1]
	}

	log.Printf("reading backup from %q", c.Conf.Src)

	var backupReader io.ReadCloser
	if bucketName != "" {
		gcsClient, err := storage.NewClient(ctx)
		if err != nil {
			return err
		}
		if backupReader, err = gcsClient.Bucket(bucketName).Object(objectName).NewReader(ctx); err != nil {
			return err
		}
		defer backupReader.Close()
	} else {
		var err error
		if backupReader, err = os.Open(c.Conf.Src); err != nil {
			return err
		}
	}

	gzipReader, err := gzip.NewReader(backupReader)
	if err != nil {
		return fmt.Errorf("gzip.NewReader: %w", err)
	}

	backupBy, err := io.ReadAll(gzipReader)
	if err != nil {
		return fmt.Errorf("io.ReadAll: %w", err)
	}
	if err := gzipReader.Close(); err != nil {
		return fmt.Errorf("zr.Close: %w", err)
	}

	grafanaBackup := GrafanaBackup{}
	if err := json.Unmarshal(backupBy, &grafanaBackup); err != nil {
		return err
	}

	log.Printf("found %d dashboard(s), %d datasource(s), and %d folder(s)", len(grafanaBackup.Dashboards), len(grafanaBackup.Datasources), len(grafanaBackup.Folders))

	folderIDMap := map[string]int{}
	// upsert folders
	for _, folder := range grafanaBackup.Folders {
		log.Printf("importing folder %s", folder.Title)
		if _, err := c.Conf.Client().GetFolderByUID(ctx, folder.UID); err == nil {
			folder.Overwrite = true
			if folder, err = c.Conf.Client().UpdateFolderByUID(ctx, folder); err != nil {
				return fmt.Errorf("UpdateFolderByUID %s %s: %w", folder.UID, folder.Title, err)
			}
		} else {
			if folder, err = c.Conf.Client().CreateFolder(ctx, folder); err != nil {
				return fmt.Errorf("CreateFolder %s %s: %w", folder.UID, folder.Title, err)
			}
		}
		folderIDMap[folder.Title] = folder.ID
	}
	log.Printf("imported folders %v", folderIDMap)

	// upsert datasources
	for _, datasource := range grafanaBackup.Datasources {
		log.Printf("importing datasource %s", datasource.Name)
		if _, err := c.Conf.Client().GetDatasource(ctx, datasource.ID); err == nil {
			if _, err := c.Conf.Client().UpdateDatasource(ctx, datasource); err != nil {
				return fmt.Errorf("UpdateDatasource %d %s: %w", datasource.ID, datasource.Name, err)
			}
		} else {
			if _, err := c.Conf.Client().CreateDatasource(ctx, datasource); err != nil {
				return fmt.Errorf("CreateDatasource %d %s: %w", datasource.ID, datasource.Name, err)
			}
		}
	}

	log.Printf("imported datasources")

	// overwrite dashboards
	for _, dashboardBackup := range grafanaBackup.Dashboards {
		dashboard := dashboardBackup.Dashboard
		dashboardMeta := dashboardBackup.Meta
		folderID := folderIDMap[dashboardMeta.FolderTitle]

		log.Printf("importing dashboard (%s)%s - folder %s %d", dashboard.UID, dashboard.Title, dashboardMeta.FolderTitle, folderID)

		// TODO(dm): map the folders and datasources to the new IDs

		dashboard.ID = 0
		// TODO(dm): remove the hack once the SDK is patched, without it keeps appending new annotations to the dashboard
		dashboard.Annotations = struct {
			List []sdk.Annotation `json:"list"`
		}{}
		if _, err := c.Conf.Client().SetDashboard(ctx, dashboard, sdk.SetDashboardParams{
			FolderID:  folderID,
			Overwrite: true,
		}); err != nil {
			return err
		}
	}

	return nil
}
