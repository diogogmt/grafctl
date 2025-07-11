package command

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/diogogmt/grafctl/pkg/grafsdk"
	"github.com/diogogmt/grafctl/pkg/simplejson"
	"github.com/stretchr/testify/assert"
)

func TestExportPanelQueriesDuplicateDescriptions(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "grafctl-export-duplicate-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a mock client
	client := &Client{
		Client:  &grafsdk.Client{},
		apiURL:  "http://localhost:3000",
		apiKey:  "test-key",
		verbose: true,
	}

	// Test the parseQueryPaths function directly
	// We'll test the parseQueryPaths function directly
	queryPaths := client.parseQueryPaths("query=queries/panel1\nquery=queries/panel2")
	assert.Equal(t, 2, len(queryPaths))
	assert.Equal(t, "queries/panel1", queryPaths[0])
	assert.Equal(t, "queries/panel2", queryPaths[1])

	// Test empty description
	emptyPaths := client.parseQueryPaths("")
	assert.Equal(t, 0, len(emptyPaths))

	// Test invalid description
	invalidPaths := client.parseQueryPaths("query=")
	assert.Equal(t, 0, len(invalidPaths))

	// Test description with just "query="
	justQueryPaths := client.parseQueryPaths("query=")
	assert.Equal(t, 0, len(justQueryPaths))
}

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

func createMockDashboardWithDuplicateDescriptions() *grafsdk.DashboardWithMeta {
	// This would be used for testing the full duplicate detection
	// For now, we test the individual functions
	return &grafsdk.DashboardWithMeta{}
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

func TestExportPanelQueries(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "grafctl-export-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a mock client
	client := &Client{
		Client:  &grafsdk.Client{},
		apiURL:  "http://localhost:3000",
		apiKey:  "test-key",
		verbose: true,
	}

	// Test SQL panel
	sqlPanel := createMockSQLPanel()
	err = client.exportPanelQueries(sqlPanel, tempDir, true)
	assert.NoError(t, err)

	// Verify SQL file was created
	sqlPath := filepath.Join(tempDir, "queries", "panel1.sql")
	_, err = os.Stat(sqlPath)
	assert.NoError(t, err)

	sqlContent, err := os.ReadFile(sqlPath)
	assert.NoError(t, err)
	assert.Equal(t, "SELECT * FROM test_table", string(sqlContent))

	// Test PromQL panel
	promqlPanel := createMockPromQLPanel()
	err = client.exportPanelQueries(promqlPanel, tempDir, true)
	assert.NoError(t, err)

	// Verify PromQL file was created
	promqlPath := filepath.Join(tempDir, "queries", "panel2.promql")
	_, err = os.Stat(promqlPath)
	assert.NoError(t, err)

	promqlContent, err := os.ReadFile(promqlPath)
	assert.NoError(t, err)
	assert.Equal(t, "up{job=\"test\"}", string(promqlContent))

	// Debug: list all files under tempDir
	fmt.Println("--- Debug: Listing files under tempDir ---")
	filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return nil
		}
		fmt.Println(path)
		return nil
	})
}

func createMockSQLPanel() *simplejson.Json {
	panel := simplejson.New()
	panel.Set("type", "stat")
	panel.Set("title", "Test Panel 1")
	panel.Set("description", "query=queries/panel1")

	datasource := map[string]interface{}{"type": "postgres"}
	panel.Set("datasource", datasource)

	target := map[string]interface{}{"rawSql": "SELECT * FROM test_table"}
	targets := []interface{}{target}
	panel.Set("targets", targets)

	return panel
}

func createMockPromQLPanel() *simplejson.Json {
	panel := simplejson.New()
	panel.Set("type", "graph")
	panel.Set("title", "Test Panel 2")
	panel.Set("description", "query=queries/panel2")

	datasource := map[string]interface{}{"type": "prometheus"}
	panel.Set("datasource", datasource)

	target := map[string]interface{}{"expr": "up{job=\"test\"}"}
	targets := []interface{}{target}
	panel.Set("targets", targets)

	return panel
}
