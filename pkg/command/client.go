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
	"github.com/diogogmt/grafctl/pkg/grafsdk"
	"github.com/diogogmt/grafctl/pkg/simplejson"
	"github.com/diogogmt/grafctl/pkg/utils"
)

type BackupProvider string

var (
	GCSBackupProvider   = BackupProvider("gcs")
	LocalBackupProvider = BackupProvider("local")
)

const dataSourceTypePrometheus = "prometheus"
const dataSourceTypeStackDriver = "stackdriver"
const defaultMinStep = "10s"

type GrafanaBackup struct {
	Datasources []*grafsdk.Datasource        `json:"datasources"`
	Folders     []*grafsdk.Folder            `json:"folders"`
	Dashboards  []*grafsdk.DashboardWithMeta `json:"dashboards"`
}

type Client struct {
	*grafsdk.Client
	apiURL  string
	apiKey  string
	verbose bool
}

func NewClient(apiURL string, apiKey string, verbose bool) *Client {
	return &Client{
		Client:  grafsdk.New(apiURL, apiKey),
		apiURL:  apiURL,
		apiKey:  apiKey,
		verbose: verbose,
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
	var err error

	// backup datasources
	if grafanaBackup.Datasources, err = c.ListDatasources(ctx); err != nil {
		return err
	}

	// backup folders
	if grafanaBackup.Folders, err = c.ListFolders(ctx); err != nil {
		return err
	}

	// backup dashboards
	dashSearchResults, err := c.Search(ctx, grafsdk.DashTypeSearchOption())
	if err != nil {
		return err
	}
	for _, dashSearchResult := range dashSearchResults {
		dashboard, err := c.GetDashboardByUID(ctx, dashSearchResult.UID)
		if err != nil {
			return err
		}
		grafanaBackup.Dashboards = append(grafanaBackup.Dashboards, dashboard)
	}

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
	case LocalBackupProvider:
		p := filepath.Join(dest, backupName)
		if err := os.WriteFile(p, backupGzippedBy, 0644); err != nil {
			return err
		}
	default:
		return fmt.Errorf("provider %q not supported", provider)
	}

	return nil
}

func (c *Client) SyncDashboard(ctx context.Context, uid string, queriesDir string) error {
	// TODO(dm): check if queriesDir exist, if not filepath.Walk panics
	dashboardFull, err := c.GetDashboardByUID(ctx, uid)
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

	for _, panelBy := range dashboardFull.Dashboard.Get("panels").MustArray() {
		panel := simplejson.NewFromAny(panelBy)
		if err := c.updatePanelTargets(queryManager, panel); err != nil {
			return err
		}
		// older versions of row panels can have sub-panels
		for _, subPanelBy := range panel.Get("panels").MustArray() {
			if err := c.updatePanelTargets(queryManager, simplejson.NewFromAny(subPanelBy)); err != nil {
				return err
			}
		}
	}

	if err := c.SaveDashboard(ctx, &grafsdk.DashboardSavePayload{
		Dashboard: dashboardFull.Dashboard,
		Overwrite: true,
		FolderID:  dashboardFull.Meta.Get("folderId").MustInt64(),
	}); err != nil {
		return fmt.Errorf("SaveDashboard: %w", err)
	}

	return nil
}

func (c *Client) updatePanelTargets(queryManager *QueryManager, panel *simplejson.Json) error {
	panelType := panel.Get("type").MustString()
	panelTitle := panel.Get("title").MustString()
	panelDesc := panel.Get("description").MustString()
	datasource := panel.Get("datasource").Get("type").MustString()

	if panelDesc == "" {
		return nil
	}
	targetsBy := panel.Get("targets").MustArray()
	if len(targetsBy) <= 0 {
		c.logd("no targets found for panel %s:%q", panelType, panelTitle)
		return nil
	}

	queries := []*Query{}
	for _, part := range strings.Split(panelDesc, "\n") {
		queryParts := strings.Split(part, "query=")
		if len(queryParts) != 2 {
			continue
		}
		queryName := queryParts[1]
		query := queryManager.Get(queryName)
		if query == nil {
			c.logd("[%s:%s] query %s not found", panelType, panelTitle, queryName)
			continue
		}
		queries = append(queries, query)
	}

	if len(targetsBy) != len(queries) {
		c.logd("found %d query(s) but only has %d target(s)", len(queries), len(targetsBy))
		return nil
	}

	for i, query := range queries {
		target := simplejson.NewFromAny(targetsBy[i])
		switch query.Type {
		case SQL:
			target.Set("rawSql", query.Raw)
		case PromQL:
			switch datasource {
			case dataSourceTypePrometheus:
				target.Set("expr", query.Raw)
			case dataSourceTypeStackDriver:
				projectName, err := target.Get("promQLQuery").Get("projectName").String()
				if err != nil {
					return err
				}
				step, err := target.Get("promQLQuery").Get("step").String()
				if err != nil {
					return err
				}

				// Default values for min step on Grafana is 10s
				if step == "" {
					step = defaultMinStep
				}

				promqlQuery := grafsdk.PromQLQuery{
					Expression:  query.Raw,
					ProjectName: projectName,
					Step:        step,
				}
				target.Set("promQLQuery", promqlQuery)
			}
		}
		c.logd("target updated: [%s:%s] query[%d] %s", panelType, panelTitle, i, query.Name)
	}
	return nil
}

func (c *Client) ExportDashboardQueries(ctx context.Context, uid string, queriesDir string, overwrite bool) error {
	dashboardFull, err := c.GetDashboardByUID(ctx, uid)
	if err != nil {
		return err
	}

	// Always create queries subdirectory
	queriesSubdir := filepath.Join(queriesDir, "queries")
	if err := os.MkdirAll(queriesSubdir, 0755); err != nil {
		return err
	}

	// Track descriptions to detect duplicates
	descriptionCounts := make(map[string]int)
	descriptionPanels := make(map[string][]string)

	// First pass: collect all descriptions and their panel info
	for _, panelBy := range dashboardFull.Dashboard.Get("panels").MustArray() {
		panel := simplejson.NewFromAny(panelBy)
		c.collectPanelDescriptions(panel, descriptionCounts, descriptionPanels)
		// Handle sub-panels in row panels (older versions)
		for _, subPanelBy := range panel.Get("panels").MustArray() {
			c.collectPanelDescriptions(simplejson.NewFromAny(subPanelBy), descriptionCounts, descriptionPanels)
		}
	}

	// Log duplicate descriptions
	for desc, count := range descriptionCounts {
		if count > 1 {
			panels := descriptionPanels[desc]
			c.logd("found %d panels with same description '%s': %v", count, desc, panels)
			return fmt.Errorf("found %d panels with same description '%s': %v", count, desc, panels)
		}
	}

	// Second pass: export queries
	for _, panelBy := range dashboardFull.Dashboard.Get("panels").MustArray() {
		panel := simplejson.NewFromAny(panelBy)
		if err := c.exportPanelQueries(panel, queriesSubdir, overwrite); err != nil {
			return err
		}
		// Handle sub-panels in row panels (older versions)
		for _, subPanelBy := range panel.Get("panels").MustArray() {
			if err := c.exportPanelQueries(simplejson.NewFromAny(subPanelBy), queriesSubdir, overwrite); err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Client) collectPanelDescriptions(panel *simplejson.Json, descriptionCounts map[string]int, descriptionPanels map[string][]string) {
	panelType := panel.Get("type").MustString()
	panelTitle := panel.Get("title").MustString()
	panelDesc := panel.Get("description").MustString()

	if panelType == "row" {
		// Row panels can have empty descriptions
		return
	}

	if panelDesc == "" {
		c.logd("panel %s:%q has empty description", panelType, panelTitle)
		return
	}

	// Check if description is just "query=" or similar invalid format
	queryPaths := c.parseQueryPaths(panelDesc)
	if len(queryPaths) == 0 {
		c.logd("panel %s:%q has invalid description format: %q", panelType, panelTitle, panelDesc)
		return
	}

	// Track each query path
	for _, queryPath := range queryPaths {
		descriptionCounts[queryPath]++
		panelInfo := fmt.Sprintf("%s:%s", panelType, panelTitle)
		descriptionPanels[queryPath] = append(descriptionPanels[queryPath], panelInfo)
	}
}

func (c *Client) parseQueryPaths(panelDesc string) []string {
	queryPaths := []string{}
	for _, part := range strings.Split(panelDesc, "\n") {
		queryParts := strings.Split(part, "query=")
		if len(queryParts) != 2 {
			continue
		}
		queryPath := strings.TrimSpace(queryParts[1])
		// Skip empty or invalid query paths
		if queryPath == "" || queryPath == "query=" {
			continue
		}
		queryPaths = append(queryPaths, queryPath)
	}
	return queryPaths
}

func (c *Client) exportPanelQueries(panel *simplejson.Json, queriesDir string, overwrite bool) error {
	panelType := panel.Get("type").MustString()
	panelTitle := panel.Get("title").MustString()
	panelDesc := panel.Get("description").MustString()
	datasource := panel.Get("datasource").Get("type").MustString()

	if panelDesc == "" {
		c.logd("no description found for panel %s:%q", panelType, panelTitle)
		return nil
	}

	targetsBy := panel.Get("targets").MustArray()
	if len(targetsBy) <= 0 {
		c.logd("no targets found for panel %s:%q", panelType, panelTitle)
		return nil
	}

	// Parse panel description to get query file paths
	queryPaths := c.parseQueryPaths(panelDesc)
	if len(queryPaths) == 0 {
		c.logd("no valid query paths found for panel %s:%q (description: %q)", panelType, panelTitle, panelDesc)
		return nil
	}

	if len(targetsBy) != len(queryPaths) {
		c.logd("found %d query path(s) but only has %d target(s) for panel %s:%q", len(queryPaths), len(targetsBy), panelType, panelTitle)
		return nil
	}

	// Export each target to its corresponding query file
	for i, queryPath := range queryPaths {
		target := simplejson.NewFromAny(targetsBy[i])
		if err := c.exportTargetToFile(target, datasource, queryPath, queriesDir, overwrite); err != nil {
			return err
		}
		c.logd("query exported: [%s:%s] query[%d] %s", panelType, panelTitle, i, queryPath)
	}

	return nil
}

func (c *Client) exportTargetToFile(target *simplejson.Json, datasource string, queryPath string, queriesDir string, overwrite bool) error {
	var queryContent string
	var fileExtension string

	switch datasource {
	case dataSourceTypePrometheus, dataSourceTypeStackDriver:
		queryContent = target.Get("expr").MustString()
		fileExtension = ".promql"
	default:
		// Treat all other datasources as SQL
		queryContent = target.Get("rawSql").MustString()
		fileExtension = ".sql"
	}

	if queryContent == "" {
		c.logd("no query content found for datasource %s (path: %s, extension: %s)", datasource, queryPath, fileExtension)
		return nil
	}

	// Always write to queries subdirectory
	fullPath := filepath.Join(queriesDir, queryPath+fileExtension)

	// Check if file exists and skip if overwrite is false
	if !overwrite {
		if _, err := os.Stat(fullPath); err == nil {
			c.logd("skipping existing file: %s", fullPath)
			return nil
		}
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write the query to file
	if err := os.WriteFile(fullPath, []byte(queryContent), 0644); err != nil {
		return err
	}

	return nil
}

func (c *Client) logd(format string, args ...interface{}) {
	if !c.verbose {
		return
	}
	log.Printf(format, args...)
}
