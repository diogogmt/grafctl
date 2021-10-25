package command

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/diogogmt/grafctl/pkg/grafsdk"
	"github.com/diogogmt/grafctl/pkg/simplejson"
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
		ShortUsage:  "grafctl import",
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

	c.Conf.logd("reading backup from %q", c.Conf.Src)

	var backupReader io.ReadCloser
	if bucketName != "" {
		gcsClient, err := storage.NewClient(ctx)
		if err != nil {
			return fmt.Errorf("storage.NewClient: %w", err)
		}
		if backupReader, err = gcsClient.Bucket(bucketName).Object(objectName).NewReader(ctx); err != nil {
			return fmt.Errorf("Bucket.NewReader: %w", err)
		}
		defer backupReader.Close()
	} else {
		var err error
		if backupReader, err = os.Open(c.Conf.Src); err != nil {
			return fmt.Errorf("os.Open: %w", err)
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
		return fmt.Errorf("json.Unmarshal: %w", err)
	}

	c.Conf.logd("found %d datasource(s), %d folder(s), and %d dashboard(s)", len(grafanaBackup.Datasources), len(grafanaBackup.Folders), len(grafanaBackup.Dashboards))

	// upsert datasources
	for _, datasource := range grafanaBackup.Datasources {
		if existingDS, err := c.Conf.Client().GetDatasourceByName(ctx, datasource.Name); err == nil {
			c.Conf.logd("datasource %d:%s:%s already exists, updating in place", datasource.ID, datasource.UID, datasource.Name)
			datasource.ID = existingDS.ID
			if err := c.Conf.Client().UpdateDatasource(ctx, datasource); err != nil {
				return fmt.Errorf("UpdateDatasource %d %s: %w", datasource.ID, datasource.Name, err)
			}
		} else {
			c.Conf.logd("datasource %d:%s:%s does not exist, creating new one", datasource.ID, datasource.UID, datasource.Name)
			datasource.ID = 0
			if _, err := c.Conf.Client().CreateDatasource(ctx, datasource); err != nil {
				return fmt.Errorf("CreateDatasource %d %s: %w", datasource.ID, datasource.Name, err)
			}
		}
	}
	c.Conf.logd("imported datasources")

	// populate the map with the title of all backup folders
	folderTitleIDMap := map[string]int64{}
	for _, backupFolder := range grafanaBackup.Folders {
		folderTitleIDMap[backupFolder.Title] = 0
	}
	// assign the existing folder id to the title map
	folders, err := c.Conf.Client().ListFolders(ctx)
	if err != nil {
		return err
	}
	for _, folder := range folders {
		folderTitleIDMap[folder.Title] = folder.ID
	}
	// create folders that do not exist
	for title, id := range folderTitleIDMap {
		if id != 0 {
			continue
		}
		c.Conf.logd("folder %s does not exist, creating new one", title)
		folder, err := c.Conf.Client().CreateFolder(ctx, title)
		if err != nil {
			return err
		}
		folderTitleIDMap[title] = folder.ID
	}

	// create a map with the backup folder IDs and the new folder IDs
	folderBackupIDMap := map[int64]int64{}
	for _, backupFolder := range grafanaBackup.Folders {
		folderBackupIDMap[backupFolder.ID] = folderTitleIDMap[backupFolder.Title]
	}

	// dashboards
	for _, dashboardFull := range grafanaBackup.Dashboards {
		dashboard := dashboardFull.Dashboard
		dashboardMeta := dashboardFull.Meta
		dashboard.Del("id") // delete references to numeric id
		uid := dashboard.Get("uid").MustString()
		title := dashboard.Get("title").MustString()
		folderTitle := dashboardMeta.Get("folderTitle").MustString()
		folderID := folderTitleIDMap[folderTitle]
		c.Conf.logd("importing dashboard %s:%q from folder %d:%q", uid, title, folderID, folderTitle)
		dashboard.Set("folderId", folderID)

		// dashboard list panels have a reference to the numeric folder id
		// we track the new folder id's in a map so we can update the panel references
		for _, p := range dashboard.Get("panels").MustArray() {
			panel := simplejson.NewFromAny(p)
			if panel.Get("type").MustString() != "dashlist" {
				continue
			}
			oldFolderID := panel.Get("folderId").MustInt64()
			newFolderID := folderBackupIDMap[panel.Get("folderId").MustInt64()]
			c.Conf.logd("updating dash list folder ID from %d do %d", oldFolderID, newFolderID)
			panel.Set("folderId", newFolderID)
		}

		if err := c.Conf.Client().SaveDashboard(ctx, &grafsdk.DashboardSavePayload{
			Dashboard: dashboard,
			Overwrite: true,
			FolderID:  folderID,
		}); err != nil {
			return fmt.Errorf("SaveDashboard: %w", err)
		}
	}

	return nil
}
