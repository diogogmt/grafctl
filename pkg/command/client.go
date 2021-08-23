package command

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/diogogmt/grafctl/pkg/utils"
	"github.com/grafana-tools/sdk"
)

type BackupProvider string

var (
	GCSBackupProvider   = BackupProvider("gcs")
	LocalBackupProvider = BackupProvider("local")
)

type GrafanaBackup struct {
	Dashboards  []BoardBackup    `json:"dashboards"`
	Datasources []sdk.Datasource `json:"datasources"`
	Folders     []sdk.Folder     `json:"folders"`
}

type BoardBackup struct {
	Dashboard sdk.Board           `json:"dashboard"`
	Meta      sdk.BoardProperties `json:"meta"`
}

type Client struct {
	*sdk.Client
	apiURL string
	apiKey string
}

func NewClient(apiURL string, apiKey string) *Client {
	return &Client{
		Client: sdk.NewClient(apiURL, apiKey, sdk.DefaultHTTPClient),
		apiURL: apiURL,
		apiKey: apiKey,
	}
}

func (c *Client) BackupGrafana(ctx context.Context, provider BackupProvider, dest string) error {
	var gcsBucket *storage.BucketHandle
	switch provider {
	case GCSBackupProvider:
		if dest == "" {
			return fmt.Errorf("missing bucket location")
		}
		gcsClient, err := storage.NewClient(ctx)
		if err != nil {
			return err
		}
		gcsBucket = gcsClient.Bucket(dest)
		if _, err := gcsBucket.Attrs(ctx); err != nil {
			return fmt.Errorf("error getting bucket %q attributes: %s", dest, err)
		}

	case LocalBackupProvider:
		// nop
	default:
		return fmt.Errorf("provider %q not supported", provider)
	}

	grafanaBackup := GrafanaBackup{}

	// backup dashboards
	foundBoards, err := c.SearchDashboards(ctx, "", false)
	if err != nil {
		return err
	}
	for _, foundBoard := range foundBoards {
		board, boardMeta, err := c.GetDashboardByUID(ctx, foundBoard.UID)
		if err != nil {
			return err
		}
		grafanaBackup.Dashboards = append(grafanaBackup.Dashboards, BoardBackup{
			Dashboard: board,
			Meta:      boardMeta,
		})
	}

	// backup datasources
	datasources, err := c.GetAllDatasources(ctx)
	if err != nil {
		return err
	}
	grafanaBackup.Datasources = datasources

	// backup folders
	folders, err := c.GetAllFolders(ctx)
	if err != nil {
		return err
	}
	grafanaBackup.Folders = folders

	u, err := url.Parse(c.apiURL)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	backupName := fmt.Sprintf("%s-%s-%d.json.gz", strings.ReplaceAll(u.Host, ".", "_"), now.Format("2006-01-02"), now.UnixNano())

	backupBy, err := json.Marshal(grafanaBackup)
	if err != nil {
		return err
	}

	backupGzippedBy, err := utils.Gzip(ctx, backupBy)
	if err != nil {
		return err
	}
	switch provider {
	case GCSBackupProvider:
		ctx, cancel := context.WithTimeout(ctx, time.Second*15)
		defer cancel()
		objectWriter := gcsBucket.Object(backupName).NewWriter(ctx)
		if _, err = io.Copy(objectWriter, bytes.NewReader(backupGzippedBy)); err != nil {
			return err
		}
		if err := objectWriter.Close(); err != nil {
			return err
		}
		fmt.Printf("gs://%s/%s\n", dest, backupName)
	case LocalBackupProvider:
		p := filepath.Join(dest, backupName)
		if err := os.WriteFile(p, backupGzippedBy, 0644); err != nil {
			return err
		}
		fmt.Printf("%s\n", p)
	default:
		return fmt.Errorf("provider %q not supported", provider)
	}

	return nil
}

func (c *Client) SyncDashboard(ctx context.Context, uid string, queriesDir string) error {
	// TODO(dm): check if queriesDir exist, if not filepath.Walk panics
	board, boardProps, err := c.GetDashboardByUID(ctx, uid)
	if err != nil {
		return err
	}

	queryManager, err := NewQueryManager(queriesDir)
	if err != nil {
		return err
	}

	if err := filepath.Walk(queriesDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if queryManager.SupportedQueryFile(path) {
			err := queryManager.Put(path)
			if err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	panels := []*sdk.Panel{}
	for _, panel := range board.Panels {
		if panel.RowPanel != nil {
			for _, nestedPanel := range panel.RowPanel.Panels {
				nestedPanel := nestedPanel
				panels = append(panels, &nestedPanel)
			}
		}
		panels = append(panels, panel)
	}

	for _, panel := range panels {
		if panel.Description == nil {
			continue
		}

		targets := panel.GetTargets()
		if targets == nil {
			log.Printf("[%s:%s] panel has no targets found", panel.Type, panel.Title)
			continue
		}

		queries := []*Query{}
		for _, part := range strings.Split(*panel.Description, "\n") {
			queryParts := strings.Split(part, "query=")
			if len(queryParts) == 2 {
				queryName := queryParts[1]
				query := queryManager.Get(queryName)
				if query == nil {
					log.Printf("[%s:%s] query %s not found", panel.Type, panel.Title, queryName)
					continue
				}
				queries = append(queries, query)
			}

		}

		for i, query := range queries {
			t := *panel.GetTargets()

			if i < len(t) {
				switch query.Type {
				case SQL:
					t[i].RawSql = query.Raw
					log.Printf("target updated: [%s:%s] query[%d] %s", panel.Type, panel.Title, i, query.Name)
				case Prometheus:
					t[i].Expr = query.Raw
					log.Printf("target updated: [%s:%s] query[%d] %s", panel.Type, panel.Title, i, query.Name)
				}
			} else {
				switch query.Type {
				case SQL:
					panel.AddTarget(&sdk.Target{
						RawSql: query.Raw,
					})
					log.Printf("target created: [%s:%s] query[%d] %s", panel.Type, panel.Title, i, query.Name)
				case Prometheus:
					panel.AddTarget(&sdk.Target{
						Expr: query.Raw,
					})
					log.Printf("target created: [%s:%s] query[%d] %s", panel.Type, panel.Title, i, query.Name)
				}
			}
		}
	}

	params := sdk.SetDashboardParams{
		FolderID:  boardProps.FolderID,
		Overwrite: false,
	}
	board.Annotations = struct {
		List []sdk.Annotation `json:"list"`
	}{}
	if _, err := c.SetDashboard(ctx, board, params); err != nil {
		return err
	}
	return nil
}
