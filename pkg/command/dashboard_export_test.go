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
