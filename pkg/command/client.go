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

	queriesDirAbs := queriesDir
	if !filepath.IsAbs(queriesDir) {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		queriesDirAbs = filepath.Join(wd, queriesDir)
	}

	queries := QueryManager{}
	if err := filepath.Walk(queriesDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".sql" || filepath.Ext(path) == ".promql" {
			query, err := NewQueryFromFile(path)
			if err != nil {
				log.Printf("NewQueryFromFile: %s", err)
				return err
			}
			queries[strings.TrimLeft(strings.ReplaceAll(path, queriesDirAbs, ""), "/")] = query
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

		for _, part := range strings.Split(*panel.Description, "\n") {
			queryNames := []string{}

			queryParts := strings.Split(part, "query=")
			if len(queryParts) == 2 {
				queryNames = append(queryNames, queryParts[1])
			}

			// TODO(dm): support multiple targets?
			targets := panel.GetTargets()
			if targets == nil {
				log.Printf("[%s:%s] panel has no targets found", panel.Type, panel.Title)
				continue
			} else if len(*targets) == 0 {
				log.Printf("[%s:%s] panel has no targets found", panel.Type, panel.Title)
				continue
			}

			t := *targets
			for i, queryName := range queryNames {
				q := queries.Get(queryName)
				if q == nil {
					log.Printf("[%s:%s] query %s not found", panel.Type, panel.Title, queryName)
					continue
				}

				if i < len(t) {
					switch q.Type {
					case SQL:
						t[i].RawSql = q.Raw
						log.Printf("[%s:%s] query[%d] %s", panel.Type, panel.Title, i, queryName)
					case Prometheus:
						t[i].Expr = q.Raw
						log.Printf("[%s:%s] query[%d] %s", panel.Type, panel.Title, i, queryName)
					}
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
