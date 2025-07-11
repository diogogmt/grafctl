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
	"regexp"
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

func (c *Client) ExportDashboard(ctx context.Context, uid string, queriesDir string, overwrite bool) error {
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
	case dataSourceTypePrometheus:
		queryContent = target.Get("expr").MustString()
		fileExtension = ".promql"
	case dataSourceTypeStackDriver:
		promqlQuery := target.Get("promQLQuery")
		if promqlQuery != nil {
			queryContent = promqlQuery.Get("expr").MustString()
		} else {
			queryContent = target.Get("expr").MustString()
		}
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

func (c *Client) UpdateDashboardDescriptions(ctx context.Context, uid string, overwrite bool, dryRun bool) error {
	dashboardFull, err := c.GetDashboardByUID(ctx, uid)
	if err != nil {
		return err
	}

	dashboardTitle := dashboardFull.Dashboard.Get("title").MustString()
	if dashboardTitle == "" {
		return fmt.Errorf("dashboard has no title")
	}

	// Create backup if not dry run
	if !dryRun {
		backupName := fmt.Sprintf("backup-%s-%d.json", uid, time.Now().Unix())
		backupPath := filepath.Join(".", backupName)
		backupData, err := json.MarshalIndent(dashboardFull, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal backup: %w", err)
		}
		if err := os.WriteFile(backupPath, backupData, 0644); err != nil {
			return fmt.Errorf("failed to write backup: %w", err)
		}
		c.logd("created backup: %s", backupPath)
	}

	panelsUpdated := 0
	panelsSkipped := 0

	// Process all panels
	for _, panelBy := range dashboardFull.Dashboard.Get("panels").MustArray() {
		panel := simplejson.NewFromAny(panelBy)
		updated, skipped, err := c.updatePanelDescription(panel, dashboardTitle, "", overwrite, dryRun)
		if err != nil {
			return err
		}
		if updated {
			panelsUpdated++
		}
		if skipped {
			panelsSkipped++
		}

		// Handle sub-panels in row panels
		for _, subPanelBy := range panel.Get("panels").MustArray() {
			subPanel := simplejson.NewFromAny(subPanelBy)
			rowTitle := panel.Get("title").MustString()
			updated, skipped, err := c.updatePanelDescription(subPanel, dashboardTitle, rowTitle, overwrite, dryRun)
			if err != nil {
				return err
			}
			if updated {
				panelsUpdated++
			}
			if skipped {
				panelsSkipped++
			}
		}
	}

	if dryRun {
		c.logd("DRY RUN: would update %d panels, skip %d panels", panelsUpdated, panelsSkipped)
	} else {
		// Save the updated dashboard
		if err := c.SaveDashboard(ctx, &grafsdk.DashboardSavePayload{
			Dashboard: dashboardFull.Dashboard,
			Overwrite: true,
			FolderID:  dashboardFull.Meta.Get("folderId").MustInt64(),
		}); err != nil {
			return fmt.Errorf("SaveDashboard: %w", err)
		}
		c.logd("updated %d panels, skipped %d panels", panelsUpdated, panelsSkipped)
	}

	return nil
}

func (c *Client) updatePanelDescription(panel *simplejson.Json, dashboardTitle, rowTitle string, overwrite, dryRun bool) (bool, bool, error) {
	panelType := panel.Get("type").MustString()
	panelTitle := panel.Get("title").MustString()
	currentDesc := panel.Get("description").MustString()

	// Check if we should update this panel
	shouldUpdate := overwrite || c.isInvalidDescription(currentDesc)
	if !shouldUpdate {
		return false, true, nil
	}

	// Generate new description
	newDesc := c.generatePanelDescription(dashboardTitle, rowTitle, panelType, panelTitle)
	if newDesc == currentDesc {
		return false, true, nil
	}

	// Update the panel description
	if !dryRun {
		panel.Set("description", newDesc)
	}

	action := "would update"
	if !dryRun {
		action = "updated"
	}

	if rowTitle != "" {
		c.logd("%s panel description: [%s:%s] in row [%s] -> %s", action, panelType, panelTitle, rowTitle, newDesc)
	} else {
		c.logd("%s panel description: [%s:%s] -> %s", action, panelType, panelTitle, newDesc)
	}

	return true, false, nil
}

func (c *Client) isInvalidDescription(desc string) bool {
	if desc == "" {
		return true
	}

	// Check if description is just "query=" or similar invalid format
	queryPaths := c.parseQueryPaths(desc)
	return len(queryPaths) == 0
}

func (c *Client) generatePanelDescription(dashboardTitle, rowTitle, panelType, panelTitle string) string {
	// Sanitize titles
	sanitizedDashboardTitle := c.sanitizeTitle(dashboardTitle)
	sanitizedPanelTitle := c.sanitizeTitle(panelTitle)

	// Get prefix for panel type
	prefix := c.getPanelTypePrefix(panelType)

	// Build the path
	var path string
	if rowTitle != "" {
		sanitizedRowTitle := c.sanitizeTitle(rowTitle)
		path = fmt.Sprintf("%s/%s/%s-%s", sanitizedDashboardTitle, sanitizedRowTitle, prefix, sanitizedPanelTitle)
	} else {
		path = fmt.Sprintf("%s/%s-%s", sanitizedDashboardTitle, prefix, sanitizedPanelTitle)
	}

	return fmt.Sprintf("query=%s", path)
}

func (c *Client) sanitizeTitle(title string) string {
	// Convert to lowercase
	title = strings.ToLower(title)

	// Replace spaces and special characters with underscores
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	title = reg.ReplaceAllString(title, "_")

	// Remove leading/trailing underscores
	title = strings.Trim(title, "_")

	// Ensure it's not empty
	if title == "" {
		title = "untitled"
	}

	return title
}

func (c *Client) getPanelTypePrefix(panelType string) string {
	// Map panel types to prefixes
	prefixMap := map[string]string{
		"table":      "table",
		"graph":      "graph",
		"stat":       "stat",
		"timeseries": "timeseries",
		"heatmap":    "heatmap",
		"barchart":   "barchart",
		"piechart":   "piechart",
		"gauge":      "gauge",
		"singlestat": "singlestat",
		"text":       "text",
		"row":        "row",
		"alertlist":  "alertlist",
		"dashlist":   "dashlist",
		"logs":       "logs",
		"news":       "news",
		"pluginlist": "pluginlist",
	}

	if prefix, exists := prefixMap[panelType]; exists {
		return prefix
	}

	// Default prefix for unknown types
	return "panel"
}

func (c *Client) logd(format string, args ...interface{}) {
	if !c.verbose {
		return
	}
	log.Printf(format, args...)
}
