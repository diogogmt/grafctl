package command

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/diogogmt/grafctl/pkg/grafsdk"
	"github.com/diogogmt/grafctl/pkg/simplejson"
	"github.com/stretchr/testify/assert"
)

// This test was redundant with TestParseQueryPaths - removed

func TestParseQueryPaths(t *testing.T) {
	client := &Client{
		Client:  &grafsdk.Client{},
		apiURL:  "http://localhost:3000",
		apiKey:  "test-key",
		verbose: true,
	}

	// Test valid query paths
	validDesc := "query=queries/panel1\nquery=queries/panel2"
	paths := client.parseQueryPaths(validDesc)
	assert.Equal(t, 2, len(paths))
	assert.Equal(t, "queries/panel1", paths[0])
	assert.Equal(t, "queries/panel2", paths[1])

	// Test empty description
	paths = client.parseQueryPaths("")
	assert.Equal(t, 0, len(paths))

	// Test invalid descriptions
	paths = client.parseQueryPaths("query=")
	assert.Equal(t, 0, len(paths))

	paths = client.parseQueryPaths("query=  ")
	assert.Equal(t, 0, len(paths))

	paths = client.parseQueryPaths("some text without query=")
	assert.Equal(t, 0, len(paths))

	// Test mixed valid and invalid
	mixedDesc := "query=queries/valid\nquery=\nquery=queries/another"
	paths = client.parseQueryPaths(mixedDesc)
	assert.Equal(t, 2, len(paths))
	assert.Equal(t, "queries/valid", paths[0])
	assert.Equal(t, "queries/another", paths[1])
}

func TestExportPanelQueriesOverwrite(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "grafctl-export-overwrite-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a mock client
	client := &Client{
		Client:  &grafsdk.Client{},
		apiURL:  "http://localhost:3000",
		apiKey:  "test-key",
		verbose: true,
	}

	// Create queries directory
	queriesDir := filepath.Join(tempDir, "queries")
	err = os.MkdirAll(queriesDir, 0755)
	assert.NoError(t, err)

	// Create an existing file with different content
	existingFile := filepath.Join(queriesDir, "panel1.sql")
	err = os.WriteFile(existingFile, []byte("EXISTING CONTENT"), 0644)
	assert.NoError(t, err)

	// Test SQL panel with overwrite=false (should skip existing file)
	sqlPanel := createMockSQLPanel()
	err = client.exportPanelQueries(sqlPanel, tempDir, false)
	assert.NoError(t, err)

	// Verify the existing file was not overwritten
	content, err := os.ReadFile(existingFile)
	assert.NoError(t, err)
	assert.Equal(t, "EXISTING CONTENT", string(content))

	// Test SQL panel with overwrite=true (should overwrite existing file)
	err = client.exportPanelQueries(sqlPanel, tempDir, true)
	assert.NoError(t, err)

	// Verify the file was overwritten
	content, err = os.ReadFile(existingFile)
	assert.NoError(t, err)
	assert.Equal(t, "SELECT * FROM test_table", string(content))
}

func TestExportPanelQueriesSingleTarget(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "grafctl-export-single-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a mock client
	client := &Client{
		Client:  &grafsdk.Client{},
		apiURL:  "http://localhost:3000",
		apiKey:  "test-key",
		verbose: true,
	}

	// Test SQL panel (single target)
	sqlPanel := createMockSQLPanel()
	err = client.exportPanelQueries(sqlPanel, tempDir, true)
	assert.NoError(t, err)

	// Verify SQL file was created without refId suffix (single target)
	sqlPath := filepath.Join(tempDir, "queries", "panel1.sql")
	_, err = os.Stat(sqlPath)
	assert.NoError(t, err)

	sqlContent, err := os.ReadFile(sqlPath)
	assert.NoError(t, err)
	assert.Equal(t, "SELECT * FROM test_table", string(sqlContent))

	// Test PromQL panel (single target)
	promqlPanel := createMockPromQLPanel()
	err = client.exportPanelQueries(promqlPanel, tempDir, true)
	assert.NoError(t, err)

	// Verify PromQL file was created without refId suffix (single target)
	promqlPath := filepath.Join(tempDir, "queries", "panel2.promql")
	_, err = os.Stat(promqlPath)
	assert.NoError(t, err)

	promqlContent, err := os.ReadFile(promqlPath)
	assert.NoError(t, err)
	assert.Equal(t, "up{job=\"test\"}", string(promqlContent))
}

func TestExportPanelQueriesMultiTarget(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "grafctl-export-multi-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a mock client
	client := &Client{
		Client:  &grafsdk.Client{},
		apiURL:  "http://localhost:3000",
		apiKey:  "test-key",
		verbose: true,
	}

	// Test multi-target panel
	multiTargetPanel := createMockMultiTargetPanel()
	err = client.exportPanelQueries(multiTargetPanel, tempDir, true)
	assert.NoError(t, err)

	// Verify multiple PromQL files were created with refId suffixes
	promqlPath1 := filepath.Join(tempDir, "queries", "cpu_usage_f.promql")
	_, err = os.Stat(promqlPath1)
	assert.NoError(t, err)

	promqlPath2 := filepath.Join(tempDir, "queries", "cpu_usage_b.promql")
	_, err = os.Stat(promqlPath2)
	assert.NoError(t, err)

	promqlPath3 := filepath.Join(tempDir, "queries", "cpu_usage_a.promql")
	_, err = os.Stat(promqlPath3)
	assert.NoError(t, err)

	// Verify content of first file
	promqlContent1, err := os.ReadFile(promqlPath1)
	assert.NoError(t, err)
	assert.Equal(t, "avg by (mode)(irate(node_cpu_seconds_total{mode='idle',service=\"analyzer\"})) * 100", string(promqlContent1))

	// Verify content of second file
	promqlContent2, err := os.ReadFile(promqlPath2)
	assert.NoError(t, err)
	assert.Equal(t, "avg by (service)(irate(node_cpu_seconds_total{mode='user',service=\"analyzer\"})) * 100", string(promqlContent2))

	// Verify content of third file
	promqlContent3, err := os.ReadFile(promqlPath3)
	assert.NoError(t, err)
	assert.Equal(t, "avg by (service)(irate(node_cpu_seconds_total{mode=\"system\",service=\"analyzer\"})) * 100", string(promqlContent3))
}

func TestExportPanelQueriesBackwardCompatibility(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "grafctl-export-backward-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a mock client
	client := &Client{
		Client:  &grafsdk.Client{},
		apiURL:  "http://localhost:3000",
		apiKey:  "test-key",
		verbose: true,
	}

	// Test panel with old query= format
	panel := simplejson.New()
	panel.Set("type", "stat")
	panel.Set("title", "Backward Compatible Panel")
	panel.Set("description", "query=queries/old_panel")

	datasource := map[string]interface{}{"type": "postgres"}
	panel.Set("datasource", datasource)

	target := map[string]interface{}{"rawSql": "SELECT * FROM old_table", "refId": "A"}
	targets := []interface{}{target}
	panel.Set("targets", targets)

	err = client.exportPanelQueries(panel, tempDir, true)
	assert.NoError(t, err)

	// Verify file was created with the old path format
	sqlPath := filepath.Join(tempDir, "queries", "old_panel.sql")
	_, err = os.Stat(sqlPath)
	assert.NoError(t, err)

	sqlContent, err := os.ReadFile(sqlPath)
	assert.NoError(t, err)
	assert.Equal(t, "SELECT * FROM old_table", string(sqlContent))
}

func createMockSQLPanel() *simplejson.Json {
	panel := simplejson.New()
	panel.Set("type", "stat")
	panel.Set("title", "Test Panel 1")
	panel.Set("description", "queries/panel1")

	datasource := map[string]interface{}{"type": "postgres"}
	panel.Set("datasource", datasource)

	target := map[string]interface{}{"rawSql": "SELECT * FROM test_table", "refId": "A"}
	targets := []interface{}{target}
	panel.Set("targets", targets)

	return panel
}

func createMockPromQLPanel() *simplejson.Json {
	panel := simplejson.New()
	panel.Set("type", "graph")
	panel.Set("title", "Test Panel 2")
	panel.Set("description", "queries/panel2")

	datasource := map[string]interface{}{"type": "prometheus"}
	panel.Set("datasource", datasource)

	target := map[string]interface{}{"expr": "up{job=\"test\"}", "refId": "A"}
	targets := []interface{}{target}
	panel.Set("targets", targets)

	return panel
}

func createMockMultiTargetPanel() *simplejson.Json {
	panel := simplejson.New()
	panel.Set("type", "timeseries")
	panel.Set("title", "CPU Usage")
	panel.Set("description", "queries/cpu_usage")

	datasource := map[string]interface{}{"type": "prometheus"}
	panel.Set("datasource", datasource)

	targets := []interface{}{
		map[string]interface{}{
			"expr":   "avg by (mode)(irate(node_cpu_seconds_total{mode='idle',service=\"analyzer\"})) * 100",
			"refId":  "F",
			"format": "time_series",
		},
		map[string]interface{}{
			"expr":   "avg by (service)(irate(node_cpu_seconds_total{mode='user',service=\"analyzer\"})) * 100",
			"refId":  "B",
			"format": "time_series",
		},
		map[string]interface{}{
			"expr":   "avg by (service)(irate(node_cpu_seconds_total{mode=\"system\",service=\"analyzer\"})) * 100",
			"refId":  "A",
			"format": "time_series",
		},
	}
	panel.Set("targets", targets)

	return panel
}
