package command

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana-tools/sdk"
)

type Client struct {
	*sdk.Client
}

func NewClient(apiURL string, apiKey string) *Client {
	// TODO(dm): validate URL and client
	return &Client{
		Client: sdk.NewClient(apiURL, apiKey, sdk.DefaultHTTPClient),
	}
}

func (c *Client) SyncDashboard(ctx context.Context, uid string, queriesDir string) error {
	// TODO(dm): check if queriesDir exist, if not filepath.Walk panics
	board, boardProps, err := c.GetDashboardByUID(ctx, uid)
	if err != nil {
		return err
	}

	queriesDirAbs := queriesDir
	if !filepath.IsAbs(queriesDir) {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		queriesDirAbs = filepath.Join(wd, queriesDir)
	}

	queries := map[string]string{}
	if err := filepath.Walk(queriesDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() || filepath.Ext(path) != ".sql" {
			return nil
		}
		queries[strings.TrimLeft(strings.ReplaceAll(path, queriesDirAbs, ""), "/")] = path
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
		queryName := ""
		for _, part := range strings.Split(*panel.Description, "\n") {
			queryParts := strings.Split(part, "query=")
			if len(queryParts) != 2 {
				continue
			}
			queryName = queryParts[1]
		}
		if queryName == "" {
			continue
		}

		var queryPath string
		// support query name with and without .sql extension
		if queryPath, _ = queries[queryName]; queryPath == "" {
			if queryPath, _ = queries[fmt.Sprintf("%s.sql", queryName)]; queryPath == "" {
				log.Printf("[%s:%s] query %s not found", panel.Type, panel.Title, queryName)
				continue
			}
		}
		queryBy, err := ioutil.ReadFile(queryPath)
		if err != nil {
			return err
		}

		// TODO(dm): support multiple targets?
		targets := panel.GetTargets()
		if targets != nil && len(*targets) > 0 {
			t := *targets
			t[0].RawSql = string(queryBy)
		}
		log.Printf("[%s:%s] query %s", panel.Type, panel.Title, queryName)
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
