package command

import (
	"context"
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
		if targets == nil || (targets != nil && len(*targets) == 0) {
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
