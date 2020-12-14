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
	"strings"

	"cloud.google.com/go/storage"
	"github.com/grafana-tools/sdk"
	"github.com/peterbourgon/ff/v2/ffcli"
)

// DashboardImportConfig has the config for the dashboardImport command and a reference to the root command config
type DashboardImportConfig struct {
	*DashboardConfig

	Src string
}

// DashboardImportCmd wraps the dashboardImport config and a ffcli.Command
type DashboardImportCmd struct {
	Conf *DashboardImportConfig

	*ffcli.Command
}

// NewDashboardImportCmd creates a new DashboardImportCmd
func NewDashboardImportCmd(dashConf *DashboardConfig) *DashboardImportCmd {
	conf := DashboardImportConfig{
		DashboardConfig: dashConf,
	}
	cmd := DashboardImportCmd{
		Conf: &conf,
	}
	fs := flag.NewFlagSet("grafctl dashboard backup", flag.ExitOnError)
	cmd.RegisterFlags(fs)

	cmd.Command = &ffcli.Command{
		Name:        "import",
		ShortUsage:  "grafc dash import",
		ShortHelp:   "Import grafana dashboards",
		FlagSet:     fs,
		Exec:        cmd.Exec,
		Subcommands: []*ffcli.Command{},
	}
	return &cmd
}

// RegisterFlags registers a set of flags for the dashboardImport command
func (c *DashboardImportCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Conf.Src, "src", "", "location where to read the backup from; either the path to a local dir or the remote bucket URL, eg; gs://grafana-backup-bucket/monitoring-2020-12-20.json")
}

// Exec executes the dashboardImport command
func (c *DashboardImportCmd) Exec(ctx context.Context, args []string) error {
	if c.Conf.Src == "" {
		return fmt.Errorf("missing -src")
	}

	var bucketName string
	var objectName string
	if strings.HasPrefix(c.Conf.Src, "gs://") {
		parts := strings.Split(c.Conf.Src, "gs://")
		if len(parts) != 2 {
			return fmt.Errorf("invalid URL")
		}
		parts = strings.Split(parts[1], "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid URL")
		}
		bucketName = parts[0]
		objectName = parts[1]
	}

	fmt.Printf("fetching dump from %q\n", c.Conf.Src)

	var tarBy []byte
	if bucketName != "" {
		gcsClient, err := storage.NewClient(ctx)
		if err != nil {
			return err
		}
		objectReader, err := gcsClient.Bucket(bucketName).Object(objectName).NewReader(ctx)
		if err != nil {
			return err
		}
		defer objectReader.Close()

		if tarBy, err = ioutil.ReadAll(objectReader); err != nil {
			return err
		}
	} else {
		var err error
		tarBy, err = ioutil.ReadFile(c.Conf.Src)
		if err != nil {
			return err
		}
	}

	tarReader := tar.NewReader(bytes.NewReader(tarBy))

	dashboardDumps := []DashboardDump{}
	for {
		if _, err := tarReader.Next(); err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		fileBuffer := bytes.Buffer{}
		if _, err := io.Copy(&fileBuffer, tarReader); err != nil {
			return err
		}
		dashboardDump := DashboardDump{}
		if err := json.Unmarshal(fileBuffer.Bytes(), &dashboardDump); err != nil {
			return nil
		}
		dashboardDumps = append(dashboardDumps, dashboardDump)
	}

	foldersMap := map[string]sdk.Folder{}
	folders, err := c.Conf.Client().GetAllFolders(ctx)
	if err != nil {
		return err
	}

	for _, folder := range folders {
		fmt.Printf("folder %d %s %s\n", folder.ID, folder.UID, folder.Title)
		foldersMap[folder.Title] = folder
	}

	for _, dashboardDump := range dashboardDumps {
		fmt.Printf("dashboard %d %s %s - folder %d %s\n", dashboardDump.Dashboard.ID, dashboardDump.Dashboard.UID, dashboardDump.Dashboard.Title, dashboardDump.Meta.FolderID, dashboardDump.Meta.FolderTitle)
		folder, ok := foldersMap[dashboardDump.Meta.FolderTitle]
		if !ok {
			fmt.Printf("creating new folder\n")
			var err error
			if folder, err = c.Conf.Client().CreateFolder(ctx, sdk.Folder{Title: dashboardDump.Meta.FolderTitle}); err != nil {
				return err
			}
		}
		dashboardDump.Dashboard.ID = 0
		// TODO(dm): remove the hack once the SDK is patched, without it keeps appending new annotations to the dashboard
		dashboardDump.Dashboard.Annotations = struct {
			List []sdk.Annotation `json:"list"`
		}{}
		if _, err := c.Conf.Client().SetDashboard(ctx, dashboardDump.Dashboard, sdk.SetDashboardParams{
			FolderID:  folder.ID,
			Overwrite: true,
		}); err != nil {
			return err
		}
	}

	return nil
}
